package gitutil

import (
	"fmt"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"gitredact/internal/exitcodes"
)

// CreateBackupRef creates a new ref at refName pointing to the current HEAD.
// refName should be a full ref name, e.g. "refs/gitredact-backup/1234567890".
func CreateBackupRef(root, refName string) error {
	repo, err := git.PlainOpen(root)
	if err != nil {
		return &ExecError{
			Code:    exitcodes.RewriteExecution,
			Message: fmt.Sprintf("failed to open repository for backup: %v", err),
		}
	}

	head, err := repo.Head()
	if err != nil {
		return &ExecError{
			Code:    exitcodes.RewriteExecution,
			Message: fmt.Sprintf("failed to read HEAD for backup: %v", err),
		}
	}

	ref := plumbing.NewHashReference(plumbing.ReferenceName(refName), head.Hash())
	if err := repo.Storer.SetReference(ref); err != nil {
		return &ExecError{
			Code:    exitcodes.RewriteExecution,
			Message: fmt.Sprintf("failed to create backup ref %s: %v", refName, err),
		}
	}

	return nil
}
