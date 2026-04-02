package gitutil

import (
	"fmt"
	"os"
	"strings"

	"gitredact/internal/exitcodes"
)

// RepoError is returned when a path cannot be resolved to a Git repository.
type RepoError struct {
	Path    string
	Message string
}

func (e *RepoError) Error() string { return e.Message }

// ResolveRoot resolves the given path to the root of the containing Git
// repository. If path is empty, the current working directory is used.
func ResolveRoot(path string) (string, error) {
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return "", &RepoError{
				Path:    path,
				Message: fmt.Sprintf("could not determine working directory: %v", err),
			}
		}
	}

	stdout, _, err := Run(path, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", &ExecError{
			Code:    exitcodes.RepoValidation,
			Message: fmt.Sprintf("%s is not inside a Git repository", path),
		}
	}

	root := strings.TrimSpace(stdout)
	if root == "" {
		return "", &ExecError{
			Code:    exitcodes.RepoValidation,
			Message: fmt.Sprintf("could not resolve repo root for %s", path),
		}
	}
	return root, nil
}
