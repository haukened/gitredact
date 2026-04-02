package filterrepo

import (
	"fmt"

	"gitredact/internal/exitcodes"
	"gitredact/internal/gitutil"
	"gitredact/internal/rewriter"
)

// RunDeletePath removes the given repo-relative path from all reachable history
// using the pure-Go rewriter (no external dependencies).
func RunDeletePath(root, path string, includeTags, silent bool) error {
	if err := rewriter.DeletePath(root, path, includeTags, silent); err != nil {
		return &gitutil.ExecError{
			Code:    exitcodes.RewriteExecution,
			Message: fmt.Sprintf("rewrite failed: %s", err),
		}
	}
	return nil
}
