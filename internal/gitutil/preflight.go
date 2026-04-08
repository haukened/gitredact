package gitutil

import (
	"errors"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type StringMatchLocation struct {
	Path              string
	FirstMatchCommit  string
	FirstMatchSummary string
	FirstMatchTime    time.Time
}

type reachableCommit struct {
	Hash       plumbing.Hash
	TreeHash   plumbing.Hash
	Message    string
	CommitTime time.Time
}

// FindStringMatchesInHistory returns one entry per repo-relative path whose
// contents contain target somewhere in reachable history. For each path, the
// earliest matching commit by commit timestamp is returned.
func FindStringMatchesInHistory(root, target string) ([]StringMatchLocation, error) {
	repo, err := git.PlainOpen(root)
	if err != nil {
		return nil, err
	}

	commits, err := listReachableCommits(repo)
	if err != nil {
		return nil, err
	}

	needle := []byte(target)
	rootTrees := commitTreeHashes(commits)

	// Phase 1: scan each unique blob at most once and record which blob hashes match.
	//
	// Correctness note: Git object IDs are content-addressed. If a blob hash does not
	// contain target, then no commit/tree/path that references that blob can contain
	// target at that file version. Conversely, if any reachable commit references a
	// matching blob, scanning that blob once is sufficient to detect the match.
	matchingBlobs, err := findMatchingBlobs(repo, rootTrees, needle)
	if err != nil {
		return nil, err
	}
	if len(matchingBlobs) == 0 {
		return nil, nil
	}

	// Phase 2: map matching blobs back to paths, recording the earliest matching
	// commit timestamp per path.
	byPath, err := earliestMatchesByPath(repo, commits, matchingBlobs)
	if err != nil {
		return nil, err
	}

	results := make([]StringMatchLocation, 0, len(byPath))
	for _, m := range byPath {
		results = append(results, m)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})
	return results, nil
}

// StringExistsInHistory returns true if target appears as a substring in any
// blob reachable from any ref.
func StringExistsInHistory(root, target string) (bool, error) {
	repo, err := git.PlainOpen(root)
	if err != nil {
		return false, err
	}

	commits, err := listReachableCommits(repo)
	if err != nil {
		return false, err
	}

	needle := []byte(target)
	rootTrees := commitTreeHashes(commits)
	return anyMatchingBlob(repo, rootTrees, needle)
}

// PathExistsInHistory returns true if the given repo-relative path appears in
// any commit reachable from any ref. Commits are deduplicated by hash.
func PathExistsInHistory(root, target string) (bool, error) {
	repo, err := git.PlainOpen(root)
	if err != nil {
		return false, err
	}

	iter, err := repo.Log(&git.LogOptions{All: true})
	if err != nil {
		return false, err
	}

	seen := make(map[plumbing.Hash]bool)
	found := false

	iterErr := iter.ForEach(func(c *object.Commit) error {
		if seen[c.Hash] {
			return nil
		}
		seen[c.Hash] = true
		_, err := c.File(target)
		if err == nil {
			found = true
			return object.ErrCanceled
		}
		return nil
	})

	if iterErr != nil && iterErr != object.ErrCanceled {
		return false, iterErr
	}
	return found, nil
}

func listReachableCommits(repo *git.Repository) ([]reachableCommit, error) {
	iter, err := repo.Log(&git.LogOptions{All: true})
	if err != nil {
		return nil, err
	}

	seen := make(map[plumbing.Hash]struct{})
	commits := []reachableCommit(nil)

	iterErr := iter.ForEach(func(c *object.Commit) error {
		if _, ok := seen[c.Hash]; ok {
			return nil
		}
		seen[c.Hash] = struct{}{}
		commits = append(commits, reachableCommit{
			Hash:       c.Hash,
			TreeHash:   c.TreeHash,
			Message:    c.Message,
			CommitTime: c.Committer.When,
		})
		return nil
	})
	if iterErr != nil {
		return nil, iterErr
	}

	// Deterministic ordering makes tests stable and lets path->earliest-commit logic
	// short-circuit once a path has been assigned.
	sort.Slice(commits, func(i, j int) bool {
		if commits[i].CommitTime.Equal(commits[j].CommitTime) {
			return commits[i].Hash.String() < commits[j].Hash.String()
		}
		return commits[i].CommitTime.Before(commits[j].CommitTime)
	})

	return commits, nil
}

func commitTreeHashes(commits []reachableCommit) []plumbing.Hash {
	out := make([]plumbing.Hash, 0, len(commits))
	seen := make(map[plumbing.Hash]struct{}, len(commits))
	for _, c := range commits {
		if _, ok := seen[c.TreeHash]; ok {
			continue
		}
		seen[c.TreeHash] = struct{}{}
		out = append(out, c.TreeHash)
	}
	return out
}

func anyMatchingBlob(repo *git.Repository, rootTrees []plumbing.Hash, needle []byte) (bool, error) {
	_, found, err := scanTreesForNeedle(repo, rootTrees, needle, true)
	return found, err
}

func findMatchingBlobs(repo *git.Repository, rootTrees []plumbing.Hash, needle []byte) (map[plumbing.Hash]struct{}, error) {
	matching, _, err := scanTreesForNeedle(repo, rootTrees, needle, false)
	return matching, err
}

// scanTreesForNeedle traverses trees starting from rootTrees, deduping commits
// (at the caller), trees, and blobs by object hash. Each unique blob is scanned
// at most once.
//
// When earlyExit is true, it returns as soon as any match is found.
func scanTreesForNeedle(repo *git.Repository, rootTrees []plumbing.Hash, needle []byte, earlyExit bool) (map[plumbing.Hash]struct{}, bool, error) {
	if len(needle) == 0 {
		// strings.Contains(_, "") is true; preserve that behavior with fast exit.
		// If the repo has any reachable blob, the answer is true.
		// (For FindStringMatchesInHistory, the caller will map all paths.)
		needle = []byte{}
	}

	seenTrees := make(map[plumbing.Hash]struct{}, len(rootTrees))
	seenBlobs := make(map[plumbing.Hash]struct{})
	matching := make(map[plumbing.Hash]struct{})

	stack := make([]plumbing.Hash, 0, len(rootTrees))
	for _, h := range rootTrees {
		if h == plumbing.ZeroHash {
			continue
		}
		stack = append(stack, h)
	}

	for len(stack) > 0 {
		h := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if _, ok := seenTrees[h]; ok {
			continue
		}
		seenTrees[h] = struct{}{}

		tree, err := repo.TreeObject(h)
		if err != nil {
			return nil, false, err
		}

		for _, entry := range tree.Entries {
			switch entry.Mode {
			case filemode.Dir:
				stack = append(stack, entry.Hash)
			case filemode.Submodule:
				// Gitlinks are not blobs; ignore.
				continue
			default:
				if _, ok := seenBlobs[entry.Hash]; ok {
					continue
				}
				seenBlobs[entry.Hash] = struct{}{}

				ok, err := blobContains(repo, entry.Hash, needle)
				if err != nil {
					return nil, false, err
				}
				if ok {
					if earlyExit {
						return nil, true, nil
					}
					matching[entry.Hash] = struct{}{}
				}
			}
		}
	}

	return matching, len(matching) > 0, nil
}

func blobContains(repo *git.Repository, blobHash plumbing.Hash, needle []byte) (bool, error) {
	if len(needle) == 0 {
		return true, nil
	}

	blob, err := repo.BlobObject(blobHash)
	if err != nil {
		return false, err
	}

	// Fast size check.
	if blob.Size < int64(len(needle)) {
		return false, nil
	}

	r, err := blob.Reader()
	if err != nil {
		return false, err
	}
	found, readErr := readerContains(r, needle)
	closeErr := r.Close()
	if readErr == nil {
		readErr = closeErr
	}
	return found, readErr
}

// readerContains returns true if needle occurs anywhere in r.
// It is streaming (does not read all content into memory).
func readerContains(r io.Reader, needle []byte) (bool, error) {
	if len(needle) == 0 {
		return true, nil
	}

	// Streaming Knuth–Morris–Pratt (KMP) to avoid reading whole blobs into memory
	// while still being correct for arbitrarily-long needles.
	lps := make([]int, len(needle))
	for i, j := 1, 0; i < len(needle); {
		if needle[i] == needle[j] {
			j++
			lps[i] = j
			i++
			continue
		}
		if j > 0 {
			j = lps[j-1]
			continue
		}
		lps[i] = 0
		i++
	}

	buf := make([]byte, 32*1024)
	j := 0

	for {
		n, err := r.Read(buf)
		if n > 0 {
			for _, b := range buf[:n] {
				for j > 0 && needle[j] != b {
					j = lps[j-1]
				}
				if needle[j] == b {
					j++
					if j == len(needle) {
						return true, nil
					}
				}
			}
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return false, nil
			}
			return false, err
		}
	}
}

type pathBlob struct {
	path string
	hash plumbing.Hash
}

func earliestMatchesByPath(repo *git.Repository, commits []reachableCommit, matchingBlobs map[plumbing.Hash]struct{}) (map[string]StringMatchLocation, error) {
	byPath := make(map[string]StringMatchLocation)
	treeCache := make(map[plumbing.Hash][]pathBlob) // memoized per-tree matching paths for this invocation

	for _, c := range commits {
		if c.TreeHash == plumbing.ZeroHash {
			continue
		}
		pairs, err := matchingBlobsInTree(repo, c.TreeHash, matchingBlobs, treeCache)
		if err != nil {
			return nil, err
		}
		for _, p := range pairs {
			// Preserve the existing behavior: treat target as a substring match, so
			// files are reported by their repo-relative path (git uses "/" always).
			existing, ok := byPath[p.path]
			match := StringMatchLocation{
				Path:              p.path,
				FirstMatchCommit:  c.Hash.String(),
				FirstMatchSummary: strings.TrimSpace(c.Message),
				FirstMatchTime:    c.CommitTime,
			}
			if !ok || c.CommitTime.Before(existing.FirstMatchTime) {
				byPath[p.path] = match
			}
		}
	}

	return byPath, nil
}

func matchingBlobsInTree(repo *git.Repository, treeHash plumbing.Hash, matchingBlobs map[plumbing.Hash]struct{}, cache map[plumbing.Hash][]pathBlob) ([]pathBlob, error) {
	if v, ok := cache[treeHash]; ok {
		return v, nil
	}

	tree, err := repo.TreeObject(treeHash)
	if err != nil {
		return nil, err
	}

	out := []pathBlob(nil)
	for _, entry := range tree.Entries {
		switch entry.Mode {
		case filemode.Dir:
			sub, err := matchingBlobsInTree(repo, entry.Hash, matchingBlobs, cache)
			if err != nil {
				return nil, err
			}
			for _, s := range sub {
				out = append(out, pathBlob{path: path.Join(entry.Name, s.path), hash: s.hash})
			}
		case filemode.Submodule:
			continue
		default:
			if _, ok := matchingBlobs[entry.Hash]; ok {
				out = append(out, pathBlob{path: entry.Name, hash: entry.Hash})
			}
		}
	}

	cache[treeHash] = out
	return out, nil
}
