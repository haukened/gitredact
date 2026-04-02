package gitutil

import (
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// StringExistsInHistory returns true if target appears as a substring in any
// blob reachable from any ref. Blobs are deduplicated by hash so identical
// content is only scanned once.
func StringExistsInHistory(root, target string) (bool, error) {
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
		files, err := c.Files()
		if err != nil {
			return err
		}
		return files.ForEach(func(f *object.File) error {
			if seen[f.Hash] {
				return nil
			}
			seen[f.Hash] = true
			contents, err := f.Contents()
			if err != nil {
				// Binary or unreadable file — skip.
				return nil
			}
			if strings.Contains(contents, target) {
				found = true
				return object.ErrCanceled
			}
			return nil
		})
	})

	if iterErr != nil && iterErr != object.ErrCanceled {
		return false, iterErr
	}
	return found, nil
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
