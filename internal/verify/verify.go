package verify

import (
	"fmt"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"gitredact/internal/exitcodes"
	"gitredact/internal/gitutil"
	"gitredact/internal/output"
)

// VerifyReplace checks that target no longer appears in any blob reachable from
// local branches (and tags when includeTags is true). Blobs are deduplicated by
// hash so identical content is only read once. This intentionally mirrors the
// rewriter's scope so that remote-tracking refs (refs/remotes/*) and backup refs
// (refs/gitredact-backup/*) are not mistakenly treated as failures.
func VerifyReplace(root, target string, includeTags bool) error {
	output.Print("running thorough verification (may take a moment on large repos)...")

	repo, err := git.PlainOpen(root)
	if err != nil {
		return fmt.Errorf("verification: could not open repository: %w", err)
	}

	seenCommits := make(map[plumbing.Hash]bool)
	seenBlobs := make(map[plumbing.Hash]bool)
	hitBlobs := make(map[plumbing.Hash]bool)

	var hits []string

	walkRef := func(ref *plumbing.Reference) error {
		commitHash := ref.Hash()
		// Dereference annotated tags.
		if obj, tagErr := repo.TagObject(commitHash); tagErr == nil {
			if obj.TargetType != plumbing.CommitObject {
				return nil
			}
			commitHash = obj.Target
		}
		iter, iterErr := repo.Log(&git.LogOptions{From: commitHash})
		if iterErr != nil {
			return nil // skip unresolvable refs
		}
		return iter.ForEach(func(c *object.Commit) error {
			if seenCommits[c.Hash] {
				return nil
			}
			seenCommits[c.Hash] = true
			files, err := c.Files()
			if err != nil {
				return nil
			}
			commitHit := false
			if err := files.ForEach(func(f *object.File) error {
				if seenBlobs[f.Hash] {
					if hitBlobs[f.Hash] {
						commitHit = true
					}
					return nil
				}
				seenBlobs[f.Hash] = true
				contents, err := f.Contents()
				if err != nil {
					return nil // binary or unreadable — skip
				}
				if strings.Contains(contents, target) {
					hitBlobs[f.Hash] = true
					commitHit = true
				}
				return nil
			}); err != nil {
				return err
			}
			if commitHit {
				hits = append(hits, c.Hash.String()[:8])
			}
			return nil
		})
	}

	branches, err := repo.Branches()
	if err != nil {
		return fmt.Errorf("verification: could not list branches: %w", err)
	}
	if refErr := branches.ForEach(walkRef); refErr != nil {
		return fmt.Errorf("verification: error walking branches: %w", refErr)
	}

	if includeTags {
		tags, err := repo.Tags()
		if err != nil {
			return fmt.Errorf("verification: could not list tags: %w", err)
		}
		if refErr := tags.ForEach(walkRef); refErr != nil {
			return fmt.Errorf("verification: error walking tags: %w", refErr)
		}
	}

	if len(hits) > 0 {
		return &gitutil.ExecError{
			Code: exitcodes.VerificationFailed,
			Message: fmt.Sprintf(
				"verification FAILED: target string still found in %d commit(s):\n  %s",
				len(hits), strings.Join(hits, "\n  "),
			),
		}
	}
	return nil
}

// VerifyDeletePath checks that target path no longer appears in any commit
// reachable from local branches (and tags when includeTags is true). This
// intentionally mirrors the rewriter's scope so that remote-tracking refs
// (refs/remotes/*) and backup refs (refs/gitredact-backup/*) are not
// mistakenly treated as failures.
func VerifyDeletePath(root, path string, includeTags bool) error {
	output.Print("verifying path removed from history...")

	repo, err := git.PlainOpen(root)
	if err != nil {
		return fmt.Errorf("verification: could not open repository: %w", err)
	}

	seenCommits := make(map[plumbing.Hash]bool)
	var hits []string

	walkRef := func(ref *plumbing.Reference) error {
		commitHash := ref.Hash()
		if obj, tagErr := repo.TagObject(commitHash); tagErr == nil {
			if obj.TargetType != plumbing.CommitObject {
				return nil
			}
			commitHash = obj.Target
		}
		iter, iterErr := repo.Log(&git.LogOptions{From: commitHash})
		if iterErr != nil {
			return nil
		}
		return iter.ForEach(func(c *object.Commit) error {
			if seenCommits[c.Hash] {
				return nil
			}
			seenCommits[c.Hash] = true
			_, fileErr := c.File(path)
			if fileErr == nil {
				hits = append(hits, c.Hash.String()[:8])
			}
			return nil
		})
	}

	branches, err := repo.Branches()
	if err != nil {
		return fmt.Errorf("verification: could not list branches: %w", err)
	}
	if refErr := branches.ForEach(walkRef); refErr != nil {
		return fmt.Errorf("verification: error walking branches: %w", refErr)
	}

	if includeTags {
		tags, err := repo.Tags()
		if err != nil {
			return fmt.Errorf("verification: could not list tags: %w", err)
		}
		if refErr := tags.ForEach(walkRef); refErr != nil {
			return fmt.Errorf("verification: error walking tags: %w", refErr)
		}
	}

	if len(hits) > 0 {
		return &gitutil.ExecError{
			Code: exitcodes.VerificationFailed,
			Message: fmt.Sprintf(
				"verification FAILED: path %q still appears in %d commit(s):\n  %s",
				path, len(hits), strings.Join(hits, "\n  "),
			),
		}
	}
	return nil
}
