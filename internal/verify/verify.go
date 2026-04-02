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
// any ref. Blobs are deduplicated by hash so identical content is only read once.
func VerifyReplace(root, target string) error {
	output.Print("running thorough verification (may take a moment on large repos)...")

	repo, err := git.PlainOpen(root)
	if err != nil {
		return fmt.Errorf("verification: could not open repository: %w", err)
	}

	refs, err := repo.References()
	if err != nil {
		return fmt.Errorf("verification: could not list refs: %w", err)
	}

	seenCommits := make(map[plumbing.Hash]bool)
	seenBlobs := make(map[plumbing.Hash]bool)

	var found bool

	refErr := refs.ForEach(func(ref *plumbing.Reference) error {
		if found {
			return nil
		}
		commitHash := ref.Hash()
		// Dereference annotated tags.
		if obj, err := repo.TagObject(commitHash); err == nil {
			if obj.TargetType != plumbing.CommitObject {
				return nil
			}
			commitHash = obj.Target
		}

		iter, err := repo.Log(&git.LogOptions{From: commitHash})
		if err != nil {
			return nil // skip unresolvable refs
		}
		return iter.ForEach(func(c *object.Commit) error {
			if found || seenCommits[c.Hash] {
				return nil
			}
			seenCommits[c.Hash] = true
			files, err := c.Files()
			if err != nil {
				return nil
			}
			return files.ForEach(func(f *object.File) error {
				if found || seenBlobs[f.Hash] {
					return nil
				}
				seenBlobs[f.Hash] = true
				contents, err := f.Contents()
				if err != nil {
					return nil // binary or unreadable — skip
				}
				if strings.Contains(contents, target) {
					found = true
				}
				return nil
			})
		})
	})

	if refErr != nil {
		return fmt.Errorf("verification: error walking refs: %w", refErr)
	}

	if found {
		return &gitutil.ExecError{
			Code:    exitcodes.VerificationFailed,
			Message: "verification FAILED: target string still found in reachable history",
		}
	}
	return nil
}

// VerifyDeletePath checks that target path no longer appears in any commit
// reachable from any ref. Commits are deduplicated by hash.
func VerifyDeletePath(root, path string) error {
	output.Print("verifying path removed from history...")

	repo, err := git.PlainOpen(root)
	if err != nil {
		return fmt.Errorf("verification: could not open repository: %w", err)
	}

	refs, err := repo.References()
	if err != nil {
		return fmt.Errorf("verification: could not list refs: %w", err)
	}

	seenCommits := make(map[plumbing.Hash]bool)
	var found bool

	refErr := refs.ForEach(func(ref *plumbing.Reference) error {
		if found {
			return nil
		}
		commitHash := ref.Hash()
		if obj, err := repo.TagObject(commitHash); err == nil {
			if obj.TargetType != plumbing.CommitObject {
				return nil
			}
			commitHash = obj.Target
		}

		iter, err := repo.Log(&git.LogOptions{From: commitHash})
		if err != nil {
			return nil
		}
		return iter.ForEach(func(c *object.Commit) error {
			if found || seenCommits[c.Hash] {
				return nil
			}
			seenCommits[c.Hash] = true
			_, err := c.File(path)
			if err == nil {
				found = true
			}
			return nil
		})
	})

	if refErr != nil {
		return fmt.Errorf("verification: error walking refs: %w", refErr)
	}

	if found {
		return &gitutil.ExecError{
			Code:    exitcodes.VerificationFailed,
			Message: fmt.Sprintf("verification FAILED: path %q still appears in reachable history", path),
		}
	}
	return nil
}
