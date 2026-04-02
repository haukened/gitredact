package filterrepo

import (
	"fmt"

	"gitredact/internal/exitcodes"
	"gitredact/internal/gitutil"
	"gitredact/internal/rewriter"
)

// RunReplace rewrites history by replacing all occurrences of from with to
// using the pure-Go rewriter (no external dependencies).
func RunReplace(root, from, to string, includeTags, silent bool) error {
	if err := rewriter.Replace(root, from, to, includeTags, silent); err != nil {
		return &gitutil.ExecError{
			Code:    exitcodes.RewriteExecution,
			Message: fmt.Sprintf("rewrite failed: %s", err),
		}
	}
	return nil
}
