package verify

import (
	"fmt"
	"os/exec"
	"strings"

	"gitredact/internal/exitcodes"
	"gitredact/internal/gitutil"
	"gitredact/internal/output"
)

func VerifyReplace(root, target string) error {
	output.Print("running thorough verification (may take a moment on large repos)...")
	revList := exec.Command("git", "rev-list", "--all")
	revList.Dir = root
	grep := exec.Command("xargs", "git", "grep", "-l", "--", target)
	grep.Dir = root
	var buf strings.Builder
	var pipeErr error
	grep.Stdin, pipeErr = revList.StdoutPipe()
	if pipeErr != nil {
		return fmt.Errorf("verification: failed to create pipe: %w", pipeErr)
	}
	grep.Stdout = &buf
	if err := revList.Start(); err != nil {
		return fmt.Errorf("verification: failed to start git rev-list: %w", err)
	}
	if err := grep.Start(); err != nil {
		_ = revList.Wait()
		return fmt.Errorf("verification: failed to start git grep: %w", err)
	}
	_ = revList.Wait()
	_ = grep.Wait()
	result := strings.TrimSpace(buf.String())
	if result != "" {
		return &gitutil.ExecError{
			Code:    exitcodes.VerificationFailed,
			Message: "verification FAILED: target string still found in reachable history",
		}
	}
	return nil
}

func VerifyDeletePath(root, path string) error {
	output.Print("verifying path removed from history...")
	stdout, _, err := gitutil.Run(root, "git", "log", "--all", "--oneline", "--", path)
	if err != nil {
		return fmt.Errorf("verification: git log failed: %w", err)
	}
	if stdout != "" {
		return &gitutil.ExecError{
			Code:    exitcodes.VerificationFailed,
			Message: fmt.Sprintf("verification FAILED: path %q still appears in reachable history", path),
		}
	}
	return nil
}
