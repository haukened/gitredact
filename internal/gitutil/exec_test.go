package gitutil

import "testing"

func TestExecError_Error(t *testing.T) {
	e := &ExecError{Code: 5, Message: "something went wrong"}
	if got := e.Error(); got != "something went wrong" {
		t.Errorf("ExecError.Error() = %q, want %q", got, "something went wrong")
	}
}

func TestExecError_ZeroCode(t *testing.T) {
	e := &ExecError{Code: 0, Message: ""}
	if got := e.Error(); got != "" {
		t.Errorf("ExecError.Error() with empty message = %q, want %q", got, "")
	}
}
