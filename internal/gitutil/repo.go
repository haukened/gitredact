package gitutil

import (
	"fmt"
	"os"

	git "github.com/go-git/go-git/v5"

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

	repo, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		if err == git.ErrRepositoryNotExists {
			return "", &ExecError{
				Code:    exitcodes.RepoValidation,
				Message: fmt.Sprintf("%s is not inside a Git repository", path),
			}
		}
		return "", &ExecError{
			Code:    exitcodes.RepoValidation,
			Message: fmt.Sprintf("could not open repository at %s: %v", path, err),
		}
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", &ExecError{
			Code:    exitcodes.RepoValidation,
			Message: fmt.Sprintf("could not access worktree: %v", err),
		}
	}

	return wt.Filesystem.Root(), nil
}
