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

// ---- maskSecret ----

func TestMaskSecret_LongString(t *testing.T) {
	s := "abcdefghijklmnopqrstuvwxyz123456" // 32 chars
	got := maskSecret(s)
	if !strings.Contains(got, "32 chars") {
		t.Errorf("maskSecret long: expected length in output, got %q", got)
	}
	if !strings.Contains(got, "3456") {
		t.Errorf("maskSecret long: expected last 4 chars, got %q", got)
	}
}

func TestMaskSecret_ShortString(t *testing.T) {
	s := "hunter2" // 7 chars, last 4 = "ter2"
	got := maskSecret(s)
	if !strings.Contains(got, "7 chars") {
		t.Errorf("maskSecret short: expected length, got %q", got)
	}
	if !strings.Contains(got, "ter2") {
		t.Errorf("maskSecret short: expected last 4 chars \"ter2\", got %q", got)
	}
}

func TestMaskSecret_ExactlyFourChars(t *testing.T) {
	s := "abcd"
	got := maskSecret(s)
	if !strings.Contains(got, "4 chars") {
		t.Errorf("maskSecret 4: expected length, got %q", got)
	}
	if !strings.Contains(got, "abcd") {
		t.Errorf("maskSecret 4: expected full string as tail, got %q", got)
	}
}

func TestMaskSecret_SingleChar(t *testing.T) {
	s := "x"
	got := maskSecret(s)
	if !strings.Contains(got, "1 chars") {
		t.Errorf("maskSecret 1: expected length, got %q", got)
	}
	if !strings.Contains(got, "x") {
		t.Errorf("maskSecret 1: expected char, got %q", got)
	}
}

// ---- PrintCompact ----

func TestPrintCompact_Replace_ShowsRepoAndMaskedSecret(t *testing.T) {
	p := Plan{
		RepoRoot:      "/some/repo",
		Operation:     "replace",
		Params:        map[string]string{"from": "mysecret12345678", "to": "REDACTED"},
		BackupEnabled: false,
	}
	got := captureStdout(t, func() { PrintCompact(p) })

	if !strings.Contains(got, "/some/repo") {
		t.Errorf("PrintCompact replace: expected repo root, got:\n%s", got)
	}
	if !strings.Contains(got, "replace:") {
		t.Errorf("PrintCompact replace: expected 'replace:' label, got:\n%s", got)
	}
	if !strings.Contains(got, "REDACTED") {
		t.Errorf("PrintCompact replace: expected to value, got:\n%s", got)
	}
	// Secret itself must not appear verbatim.
	if strings.Contains(got, "mysecret12345678") {
		t.Errorf("PrintCompact replace: secret must not appear verbatim, got:\n%s", got)
	}
	// Last 4 chars of secret should appear.
	if !strings.Contains(got, "5678") {
		t.Errorf("PrintCompact replace: expected last 4 chars of secret, got:\n%s", got)
	}
}

func TestPrintCompact_DeletePath_ShowsRemoveLine(t *testing.T) {
	p := Plan{
		RepoRoot:  "/other/repo",
		Operation: "delete-path",
		Params:    map[string]string{"path": "secrets/key.pem"},
	}
	got := captureStdout(t, func() { PrintCompact(p) })

	if !strings.Contains(got, "remove:") {
		t.Errorf("PrintCompact delete-path: expected 'remove:' label, got:\n%s", got)
	}
	if !strings.Contains(got, "secrets/key.pem") {
		t.Errorf("PrintCompact delete-path: expected path, got:\n%s", got)
	}
}

func TestPrintCompact_BackupShownWhenEnabled(t *testing.T) {
	p := Plan{
		RepoRoot:      "/repo",
		Operation:     "replace",
		Params:        map[string]string{"from": "secret", "to": "REDACTED"},
		BackupEnabled: true,
		BackupRef:     "refs/gitredact-backup/1234567890",
	}
	got := captureStdout(t, func() { PrintCompact(p) })

	if !strings.Contains(got, "refs/gitredact-backup/1234567890") {
		t.Errorf("PrintCompact backup enabled: expected backup ref, got:\n%s", got)
	}
}

func TestPrintCompact_BackupNotShownWhenDisabled(t *testing.T) {
	p := Plan{
		RepoRoot:      "/repo",
		Operation:     "delete-path",
		Params:        map[string]string{"path": "file.txt"},
		BackupEnabled: false,
	}
	got := captureStdout(t, func() { PrintCompact(p) })

	if strings.Contains(got, "backup") {
		t.Errorf("PrintCompact backup disabled: expected no backup line, got:\n%s", got)
	}
}

func TestPrintCompact_IncludeTagsShownWhenTrue(t *testing.T) {
	p := Plan{
		RepoRoot:    "/repo",
		Operation:   "delete-path",
		Params:      map[string]string{"path": "file.txt"},
		IncludeTags: true,
	}
	got := captureStdout(t, func() { PrintCompact(p) })

	if !strings.Contains(got, "include-tags") {
		t.Errorf("PrintCompact include-tags=true: expected include-tags line, got:\n%s", got)
	}
}

func TestPrintCompact_IncludeTagsHiddenWhenFalse(t *testing.T) {
	p := Plan{
		RepoRoot:    "/repo",
		Operation:   "delete-path",
		Params:      map[string]string{"path": "file.txt"},
		IncludeTags: false,
	}
	got := captureStdout(t, func() { PrintCompact(p) })

	if strings.Contains(got, "include-tags") {
		t.Errorf("PrintCompact include-tags=false: expected no include-tags line, got:\n%s", got)
	}
}

func TestPrintCompact_NoPlanHeader(t *testing.T) {
	p := Plan{
		RepoRoot:  "/repo",
		Operation: "delete-path",
		Params:    map[string]string{"path": "file.txt"},
	}
	got := captureStdout(t, func() { PrintCompact(p) })

	if strings.Contains(got, "gitredact plan") {
		t.Errorf("PrintCompact: must not contain old plan header, got:\n%s", got)
	}
	if strings.Contains(got, "---") {
		t.Errorf("PrintCompact: must not contain --- delimiters, got:\n%s", got)
	}
}
