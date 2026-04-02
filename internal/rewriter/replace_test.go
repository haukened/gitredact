package rewriter

import (
	"strings"
	"testing"
)

func TestReplace_Basic(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "secret.txt", "my-api-key=abc123", "initial")

	if err := Replace(dir, "my-api-key", "REDACTED", false, true); err != nil {
		t.Fatalf("Replace: unexpected error: %v", err)
	}

	got := fileContents(t, dir, "secret.txt")
	if strings.Contains(got, "my-api-key") {
		t.Errorf("Replace: original string still present: %q", got)
	}
	if !strings.Contains(got, "REDACTED") {
		t.Errorf("Replace: replacement not found: %q", got)
	}
}

func TestReplace_NoMatch(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "safe.txt", "harmless content", "initial")

	origCommit := headCommit(t, dir)
	if err := Replace(dir, "my-api-key", "REDACTED", false, true); err != nil {
		t.Fatalf("Replace (no match): unexpected error: %v", err)
	}

	// Commit hash must be unchanged when nothing needed replacing.
	afterCommit := headCommit(t, dir)
	if afterCommit.Hash != origCommit.Hash {
		t.Errorf("Replace (no match): commit hash changed unexpectedly")
	}
}

func TestReplace_MultipleCommits(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "a.txt", "secret=first", "first")
	commitFile(t, dir, repo, "b.txt", "secret=second", "second")

	if err := Replace(dir, "secret", "XXXX", false, true); err != nil {
		t.Fatalf("Replace (multi-commit): unexpected error: %v", err)
	}

	for _, name := range []string{"a.txt", "b.txt"} {
		got := fileContents(t, dir, name)
		if strings.Contains(got, "secret") {
			t.Errorf("Replace (multi-commit): %q still contains 'secret': %q", name, got)
		}
	}
}

func TestReplace_NestedPath(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "config/app.yaml", "password: hunter2", "initial")

	if err := Replace(dir, "hunter2", "REDACTED", false, true); err != nil {
		t.Fatalf("Replace (nested): unexpected error: %v", err)
	}

	got := fileContents(t, dir, "config/app.yaml")
	if strings.Contains(got, "hunter2") {
		t.Errorf("Replace (nested): original still present: %q", got)
	}
}

func TestReplace_WithVerboseOutput(t *testing.T) {
	// Tests the non-silent path.
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "f.txt", "token=abc", "init")

	if err := Replace(dir, "token", "REDACTED", false, false); err != nil {
		t.Fatalf("Replace (verbose): unexpected error: %v", err)
	}
}

func TestReplace_InvalidRepo(t *testing.T) {
	dir := t.TempDir()
	err := Replace(dir, "x", "y", false, true)
	if err == nil {
		t.Fatal("Replace(non-git): expected error, got nil")
	}
}
