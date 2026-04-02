package filterrepo

import (
	"fmt"
	"os"

	"gitredact/internal/exitcodes"
	"gitredact/internal/gitutil"
)

func WriteReplacementsFile(from, to string) (string, func(), error) {
	f, err := os.CreateTemp("", "gitredact-replace-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("could not create replacements temp file: %w", err)
	}
	line := fmt.Sprintf("literal:%s==>literal:%s\n", from, to)
	if _, err := fmt.Fprint(f, line); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", func() {}, fmt.Errorf("could not write replacements file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", func() {}, fmt.Errorf("could not close replacements file: %w", err)
	}
	cleanup := func() { _ = os.Remove(f.Name()) }
	return f.Name(), cleanup, nil
}

func RunReplace(root, tempFile string, includeTags bool) error {
	args := []string{"git-filter-repo", "--replace-text", tempFile, "--force"}
	if !includeTags {
		args = append(args, "--refs", "refs/heads/*")
	}
	_, stderr, err := gitutil.Run(root, args...)
	if err != nil {
		return &gitutil.ExecError{
			Code:    exitcodes.RewriteExecution,
			Message: fmt.Sprintf("git-filter-repo replace failed: %s", stderr),
		}
	}
	return nil
}
