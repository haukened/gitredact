package filterrepo

import (
	"fmt"

	"gitredact/internal/exitcodes"
	"gitredact/internal/gitutil"
)

// RunDeletePath invokes git-filter-repo to remove the given repo-relative path
// from all reachable history. When includeTags is false, only refs/heads/* are
// rewritten.
func RunDeletePath(root, path string, includeTags bool) error {
	args := []string{"git-filter-repo", "--path", path, "--invert-paths", "--force"}
	if !includeTags {
		args = append(args, "--refs", "refs/heads/*")
	}
	_, stderr, err := gitutil.Run(root, args...)
	if err != nil {
		return &gitutil.ExecError{
			Code:    exitcodes.RewriteExecution,
			Message: fmt.Sprintf("git-filter-repo delete-path failed: %s", stderr),
		}
	}
	return nil
}
