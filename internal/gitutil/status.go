package gitutil

import "gitredact/internal/exitcodes"

// IsDirty returns true if the working tree has uncommitted changes.
func IsDirty(root string) (bool, error) {
	stdout, _, err := Run(root, "git", "status", "--porcelain")
	if err != nil {
		return false, &ExecError{
			Code:    exitcodes.RepoValidation,
			Message: "failed to check worktree status: " + err.Error(),
		}
	}
	return stdout != "", nil
}
