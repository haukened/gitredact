package gitutil

import (
	"fmt"
	"os/exec"
	"strings"

	"gitredact/internal/exitcodes"
)

type ExecError struct {
	Code    int
	Message string
}

func (e *ExecError) Error() string { return e.Message }

func Run(root string, args ...string) (string, string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = root
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	return strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String()), err
}

func CheckDeps() error {
	for _, bin := range []string{"git", "git-filter-repo"} {
		if _, err := exec.LookPath(bin); err != nil {
			return &ExecError{
				Code:    exitcodes.DependencyMissing,
				Message: fmt.Sprintf("required tool not found on PATH: %s", bin),
			}
		}
	}
	return nil
}
