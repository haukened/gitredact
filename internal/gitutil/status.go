package gitutil

import (
	"fmt"

	git "github.com/go-git/go-git/v5"

	"gitredact/internal/exitcodes"
)

// IsDirty returns true if the working tree has uncommitted changes.
func IsDirty(root string) (bool, error) {
	repo, err := git.PlainOpen(root)
	if err != nil {
		return false, &ExecError{
			Code:    exitcodes.RepoValidation,
			Message: fmt.Sprintf("failed to open repository: %v", err),
		}
	}
	wt, err := repo.Worktree()
	if err != nil {
		return false, &ExecError{
			Code:    exitcodes.RepoValidation,
			Message: fmt.Sprintf("failed to access worktree: %v", err),
		}
	}
	status, err := wt.Status()
	if err != nil {
		return false, &ExecError{
			Code:    exitcodes.RepoValidation,
			Message: "failed to check worktree status: " + err.Error(),
		}
	}
	return !status.IsClean(), nil
}
