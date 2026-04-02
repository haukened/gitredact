package rewriter

import (
	"errors"
	"fmt"
	"io"
	"path"

	"gitredact/internal/output"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage"
)

// Options controls what the rewriter does to each blob and which paths it keeps.
// A nil BlobMapper means blobs are passed through unchanged.
// A nil PathFilter means all paths are kept.
type Options struct {
	BlobMapper func(filePath string, content []byte) ([]byte, bool)
	PathFilter func(filePath string) bool
	Silent     bool // suppress per-commit progress output
}

type historyRewriter struct {
	repo      *git.Repository
	storer    storage.Storer
	blobCache map[plumbing.Hash]plumbing.Hash
	treeCache map[plumbing.Hash]plumbing.Hash
	commitMap map[plumbing.Hash]plumbing.Hash
	opts      Options
}

// run is the core entry point used by Replace and DeletePath.
func run(root string, includeTags bool, opts Options) error {
	repo, err := git.PlainOpen(root)
	if err != nil {
		return fmt.Errorf("could not open repository: %w", err)
	}

	rw := &historyRewriter{
		repo:      repo,
		storer:    repo.Storer,
		blobCache: make(map[plumbing.Hash]plumbing.Hash),
		treeCache: make(map[plumbing.Hash]plumbing.Hash),
		commitMap: make(map[plumbing.Hash]plumbing.Hash),
		opts:      opts,
	}

	type refEntry struct {
		ref    *plumbing.Reference
		tagObj *object.Tag // non-nil for annotated tags
	}

	var refs []refEntry

	branches, err := repo.Branches()
	if err != nil {
		return fmt.Errorf("could not list branches: %w", err)
	}
	if iterErr := branches.ForEach(func(ref *plumbing.Reference) error {
		refs = append(refs, refEntry{ref: ref})
		return nil
	}); iterErr != nil {
		return fmt.Errorf("could not iterate branches: %w", iterErr)
	}

	if includeTags {
		tags, err := repo.Tags()
		if err != nil {
			return fmt.Errorf("could not list tags: %w", err)
		}
		if iterErr := tags.ForEach(func(ref *plumbing.Reference) error {
			tagObj, tagErr := repo.TagObject(ref.Hash())
			if tagErr != nil && !errors.Is(tagErr, plumbing.ErrObjectNotFound) {
				return fmt.Errorf("could not inspect tag %s: %w", ref.Name(), tagErr)
			}
			refs = append(refs, refEntry{ref: ref, tagObj: tagObj})
			return nil
		}); iterErr != nil {
			return fmt.Errorf("could not iterate tags: %w", iterErr)
		}
	}

	// Rewrite all commits reachable from each ref (memoized recursion handles order).
	for _, entry := range refs {
		var commitHash plumbing.Hash
		if entry.tagObj != nil {
			if entry.tagObj.TargetType != plumbing.CommitObject {
				continue
			}
			commitHash = entry.tagObj.Target
		} else {
			commitHash = entry.ref.Hash()
		}
		if _, rewriteErr := rw.rewriteCommit(commitHash); rewriteErr != nil {
			return fmt.Errorf("rewriting commit %s: %w", commitHash, rewriteErr)
		}
	}

	// Update refs to point to the new commit hashes.
	for _, entry := range refs {
		var oldCommitHash plumbing.Hash
		if entry.tagObj != nil {
			if entry.tagObj.TargetType != plumbing.CommitObject {
				continue
			}
			oldCommitHash = entry.tagObj.Target
		} else {
			oldCommitHash = entry.ref.Hash()
		}

		newCommitHash, ok := rw.commitMap[oldCommitHash]
		if !ok {
			newCommitHash = oldCommitHash
		}

		if entry.tagObj != nil {
			// Annotated tag: skip if commit is identical.
			if newCommitHash == oldCommitHash {
				continue
			}
			newTagObj := *entry.tagObj
			newTagObj.Hash = plumbing.ZeroHash
			newTagObj.PGPSignature = ""
			newTagObj.Target = newCommitHash

			encoded := &plumbing.MemoryObject{}
			if encErr := newTagObj.Encode(encoded); encErr != nil {
				return fmt.Errorf("could not encode tag %s: %w", entry.ref.Name(), encErr)
			}
			newTagHash, storeErr := rw.storer.SetEncodedObject(encoded)
			if storeErr != nil {
				return fmt.Errorf("could not store tag %s: %w", entry.ref.Name(), storeErr)
			}
			newRef := plumbing.NewHashReference(entry.ref.Name(), newTagHash)
			if setErr := rw.storer.SetReference(newRef); setErr != nil {
				return fmt.Errorf("could not update tag ref %s: %w", entry.ref.Name(), setErr)
			}
		} else {
			if newCommitHash == oldCommitHash {
				continue
			}
			newRef := plumbing.NewHashReference(entry.ref.Name(), newCommitHash)
			if setErr := rw.storer.SetReference(newRef); setErr != nil {
				return fmt.Errorf("could not update ref %s: %w", entry.ref.Name(), setErr)
			}
		}
	}

	return nil
}

// rewriteCommit returns the new hash for oldHash (which may equal oldHash if
// no content changed). Results are memoized.
func (rw *historyRewriter) rewriteCommit(oldHash plumbing.Hash) (plumbing.Hash, error) {
	if h, ok := rw.commitMap[oldHash]; ok {
		return h, nil
	}

	commit, err := rw.repo.CommitObject(oldHash)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("could not load commit %s: %w", oldHash, err)
	}

	// Process parents first (oldest-first via recursion).
	newParentHashes := make([]plumbing.Hash, len(commit.ParentHashes))
	parentsChanged := false
	for i, parentHash := range commit.ParentHashes {
		newParent, parentErr := rw.rewriteCommit(parentHash)
		if parentErr != nil {
			return plumbing.ZeroHash, parentErr
		}
		newParentHashes[i] = newParent
		if newParent != parentHash {
			parentsChanged = true
		}
	}

	newTreeHash, err := rw.rewriteTree(commit.TreeHash, "")
	if err != nil {
		return plumbing.ZeroHash, err
	}

	if newTreeHash == commit.TreeHash && !parentsChanged {
		rw.commitMap[oldHash] = oldHash
		return oldHash, nil
	}

	newCommit := *commit
	newCommit.Hash = plumbing.ZeroHash
	newCommit.PGPSignature = ""
	newCommit.TreeHash = newTreeHash
	newCommit.ParentHashes = newParentHashes

	encoded := &plumbing.MemoryObject{}
	if encErr := newCommit.Encode(encoded); encErr != nil {
		return plumbing.ZeroHash, fmt.Errorf("could not encode commit: %w", encErr)
	}

	newHash, storeErr := rw.storer.SetEncodedObject(encoded)
	if storeErr != nil {
		return plumbing.ZeroHash, fmt.Errorf("could not store commit: %w", storeErr)
	}

	rw.commitMap[oldHash] = newHash
	if !rw.opts.Silent {
		output.Print("  commit %.8s -> %.8s", oldHash, newHash)
	}
	return newHash, nil
}

// rewriteTree returns the new hash for the tree at oldHash. Results are memoized.
// currentPath is the repo-relative path prefix for entries in this tree (used
// for PathFilter and BlobMapper). Must use "/" as separator (git convention).
func (rw *historyRewriter) rewriteTree(oldHash plumbing.Hash, currentPath string) (plumbing.Hash, error) {
	if h, ok := rw.treeCache[oldHash]; ok {
		return h, nil
	}

	tree, err := rw.repo.TreeObject(oldHash)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("could not load tree %s: %w", oldHash, err)
	}

	changed := false
	newEntries := make([]object.TreeEntry, 0, len(tree.Entries))

	for _, entry := range tree.Entries {
		entryPath := path.Join(currentPath, entry.Name)

		switch entry.Mode {
		case filemode.Dir:
			newSubTreeHash, subErr := rw.rewriteTree(entry.Hash, entryPath)
			if subErr != nil {
				return plumbing.ZeroHash, subErr
			}
			if newSubTreeHash != entry.Hash {
				changed = true
			}
			newEntries = append(newEntries, object.TreeEntry{
				Name: entry.Name,
				Mode: entry.Mode,
				Hash: newSubTreeHash,
			})

		case filemode.Submodule:
			// Pass submodule gitlinks through unchanged.
			newEntries = append(newEntries, entry)

		default:
			// Blob (Regular, Executable, Symlink, Deprecated).
			if rw.opts.PathFilter != nil && !rw.opts.PathFilter(entryPath) {
				changed = true
				// Drop this entry - path is being removed.
				continue
			}
			if rw.opts.BlobMapper != nil {
				newBlobHash, blobChanged, blobErr := rw.rewriteBlob(entry.Hash, entryPath)
				if blobErr != nil {
					return plumbing.ZeroHash, blobErr
				}
				if blobChanged {
					changed = true
				}
				newEntries = append(newEntries, object.TreeEntry{
					Name: entry.Name,
					Mode: entry.Mode,
					Hash: newBlobHash,
				})
				continue
			}
			newEntries = append(newEntries, entry)
		}
	}

	if !changed {
		rw.treeCache[oldHash] = oldHash
		return oldHash, nil
	}

	newTree := &object.Tree{Entries: newEntries}
	encoded := &plumbing.MemoryObject{}
	if encErr := newTree.Encode(encoded); encErr != nil {
		return plumbing.ZeroHash, fmt.Errorf("could not encode tree: %w", encErr)
	}

	newHash, storeErr := rw.storer.SetEncodedObject(encoded)
	if storeErr != nil {
		return plumbing.ZeroHash, fmt.Errorf("could not store tree: %w", storeErr)
	}

	rw.treeCache[oldHash] = newHash
	return newHash, nil
}

// rewriteBlob returns the new hash for oldHash after applying BlobMapper.
// Returns (newHash, changed, error). Results are memoized by blob hash;
// BlobMapper must be path-independent for this caching to be correct.
func (rw *historyRewriter) rewriteBlob(oldHash plumbing.Hash, filePath string) (plumbing.Hash, bool, error) {
	if h, ok := rw.blobCache[oldHash]; ok {
		return h, h != oldHash, nil
	}

	blob, err := rw.repo.BlobObject(oldHash)
	if err != nil {
		return plumbing.ZeroHash, false, fmt.Errorf("could not load blob %s: %w", oldHash, err)
	}

	reader, err := blob.Reader()
	if err != nil {
		return plumbing.ZeroHash, false, fmt.Errorf("could not open blob %s: %w", oldHash, err)
	}
	content, readErr := io.ReadAll(reader)
	reader.Close()
	if readErr != nil {
		return plumbing.ZeroHash, false, fmt.Errorf("could not read blob %s: %w", oldHash, readErr)
	}

	newContent, changed := rw.opts.BlobMapper(filePath, content)
	if !changed {
		rw.blobCache[oldHash] = oldHash
		return oldHash, false, nil
	}

	encoded := &plumbing.MemoryObject{}
	encoded.SetType(plumbing.BlobObject)
	if _, writeErr := encoded.Write(newContent); writeErr != nil {
		return plumbing.ZeroHash, false, fmt.Errorf("could not write blob content: %w", writeErr)
	}

	newHash, storeErr := rw.storer.SetEncodedObject(encoded)
	if storeErr != nil {
		return plumbing.ZeroHash, false, fmt.Errorf("could not store blob: %w", storeErr)
	}

	rw.blobCache[oldHash] = newHash
	return newHash, true, nil
}
