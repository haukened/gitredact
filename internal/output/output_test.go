package output

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

func TestSetVerbose_DisabledByDefault(t *testing.T) {
	defer SetVerbose(false)
	SetVerbose(false)
	got := captureStdout(t, func() { Verbose("should not appear") })
	if got != "" {
		t.Errorf("verbose=false: expected no output, got %q", got)
	}
}

func TestSetVerbose_Enabled(t *testing.T) {
	defer SetVerbose(false)
	SetVerbose(true)
	got := captureStdout(t, func() { Verbose("should appear") })
	if !strings.Contains(got, "should appear") {
		t.Errorf("verbose=true: expected output, got %q", got)
	}
}

func TestPrint(t *testing.T) {
	got := captureStdout(t, func() { Print("hello %s", "world") })
	if !strings.Contains(got, "hello world\n") {
		t.Errorf("Print: got %q", got)
	}
}

func TestPrint_NoArgs(t *testing.T) {
	got := captureStdout(t, func() { Print("bare message") })
	if !strings.Contains(got, "bare message\n") {
		t.Errorf("Print no-args: got %q", got)
	}
}

func TestVerbose_Off(t *testing.T) {
	defer SetVerbose(false)
	SetVerbose(false)
	got := captureStdout(t, func() { Verbose("quiet %d", 1) })
	if got != "" {
		t.Errorf("Verbose off: expected empty, got %q", got)
	}
}

func TestVerbose_On(t *testing.T) {
	defer SetVerbose(false)
	SetVerbose(true)
	got := captureStdout(t, func() { Verbose("loud %d", 7) })
	if !strings.Contains(got, "loud 7\n") {
		t.Errorf("Verbose on: got %q", got)
	}
}

func TestWarn(t *testing.T) {
	got := captureStdout(t, func() { Warn("bad thing %d", 42) })
	if !strings.HasPrefix(got, "WARNING:") {
		t.Errorf("Warn: expected WARNING prefix, got %q", got)
	}
	if !strings.Contains(got, "bad thing 42") {
		t.Errorf("Warn: expected message content, got %q", got)
	}
}

func TestSection(t *testing.T) {
	got := captureStdout(t, func() { Section("My Section") })
	if !strings.Contains(got, "My Section") {
		t.Errorf("Section: expected title, got %q", got)
	}
	if !strings.Contains(got, "===") {
		t.Errorf("Section: expected === delimiters, got %q", got)
	}
}
