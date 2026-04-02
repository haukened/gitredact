package plan

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	r.Close()
	return buf.String()
}

func TestPrint_BackupEnabled(t *testing.T) {
	p := Plan{
		RepoRoot:      "/some/repo",
		Operation:     "replace",
		Params:        map[string]string{"from": "secret", "to": "REDACTED"},
		IsDirty:       false,
		IncludeTags:   true,
		BackupEnabled: true,
		BackupRef:     "refs/gitredact-backup/1234567890",
		Commands:      []string{"gitredact rewriter.Replace (pure Go)"},
	}
	got := captureStdout(t, func() { Print(p) })

	checks := []string{
		"/some/repo",
		"replace",
		"secret",
		"REDACTED",
		"enabled",
		"refs/gitredact-backup/1234567890",
		"gitredact rewriter.Replace (pure Go)",
	}
	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("Print (backup enabled): expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestPrint_BackupDisabled(t *testing.T) {
	p := Plan{
		RepoRoot:      "/other/repo",
		Operation:     "delete-path",
		Params:        map[string]string{"path": "secrets/key.pem"},
		IsDirty:       true,
		IncludeTags:   false,
		BackupEnabled: false,
		Commands:      []string{"cmd1", "cmd2"},
	}
	got := captureStdout(t, func() { Print(p) })

	if !strings.Contains(got, "disabled") {
		t.Errorf("Print (backup disabled): expected 'disabled', got:\n%s", got)
	}
	if !strings.Contains(got, "cmd1") {
		t.Errorf("Print (backup disabled): expected cmd1, got:\n%s", got)
	}
	if !strings.Contains(got, "cmd2") {
		t.Errorf("Print (backup disabled): expected cmd2, got:\n%s", got)
	}
}

func TestPrint_MultipleParams(t *testing.T) {
	p := Plan{
		RepoRoot:  "/repo",
		Operation: "replace",
		Params: map[string]string{
			"from": "old",
			"to":   "new",
		},
		Commands: []string{"run something"},
	}
	got := captureStdout(t, func() { Print(p) })
	if !strings.Contains(got, "old") {
		t.Errorf("expected 'old' in output, got:\n%s", got)
	}
	if !strings.Contains(got, "new") {
		t.Errorf("expected 'new' in output, got:\n%s", got)
	}
}

func TestPrint_EmptyCommands(t *testing.T) {
	p := Plan{
		RepoRoot:  "/repo",
		Operation: "noop",
		Commands:  []string{},
	}
	got := captureStdout(t, func() { Print(p) })
	if !strings.Contains(got, "gitredact plan") {
		t.Errorf("expected plan header, got:\n%s", got)
	}
}
