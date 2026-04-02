package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitredact/internal/exitcodes"
	"gitredact/internal/gitutil"
)

// ---- confirm() tests ----

func TestConfirm_Yes(t *testing.T) {
	setStdin(t, "y\n")
	if err := confirm(); err != nil {
		t.Errorf("confirm('y'): unexpected error: %v", err)
	}
}

func TestConfirm_YesFull(t *testing.T) {
	setStdin(t, "yes\n")
	if err := confirm(); err != nil {
		t.Errorf("confirm('yes'): unexpected error: %v", err)
	}
}

func TestConfirm_YesUppercase(t *testing.T) {
	setStdin(t, "YES\n")
	if err := confirm(); err != nil {
		t.Errorf("confirm('YES'): unexpected error: %v", err)
	}
}

func TestConfirm_No(t *testing.T) {
	setStdin(t, "n\n")
	err := confirm()
	if err == nil {
		t.Fatal("confirm('n'): expected error, got nil")
	}
	execErr, ok := err.(*gitutil.ExecError)
	if !ok {
		t.Fatalf("expected *gitutil.ExecError, got %T", err)
	}
	if execErr.Code != exitcodes.UserDeclined {
		t.Errorf("code = %d, want %d", execErr.Code, exitcodes.UserDeclined)
	}
}

func TestConfirm_Empty(t *testing.T) {
	setStdin(t, "\n")
	err := confirm()
	if err == nil {
		t.Fatal("confirm(''): expected error, got nil")
	}
}

func TestConfirm_EOF(t *testing.T) {
	// An empty pipe (immediate EOF) should decline.
	setStdin(t, "")
	err := confirm()
	if err == nil {
		t.Fatal("confirm(EOF): expected error, got nil")
	}
}

// ---- RunReplace tests ----

func TestRunReplace_InvalidRepo(t *testing.T) {
	dir := t.TempDir()
	err := RunReplace(ReplaceRequest{
		From:     "secret",
		To:       "REDACTED",
		RepoPath: dir,
		Yes:      true,
		DryRun:   true,
	})
	if err == nil {
		t.Fatal("RunReplace(non-git): expected error, got nil")
	}
}

func TestRunReplace_StringNotInHistory(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "file.txt", "safe content", "initial")

	err := RunReplace(ReplaceRequest{
		From:     "my-secret",
		To:       "REDACTED",
		RepoPath: dir,
		Yes:      true,
		DryRun:   true,
	})
	if err == nil {
		t.Fatal("RunReplace(string not found): expected error, got nil")
	}
	execErr, ok := err.(*gitutil.ExecError)
	if !ok {
		t.Fatalf("expected *gitutil.ExecError, got %T: %v", err, err)
	}
	if execErr.Code != exitcodes.NoMatchesFound {
		t.Errorf("code = %d, want %d", execErr.Code, exitcodes.NoMatchesFound)
	}
}

func TestRunReplace_DirtyWorktree_Rejected(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "file.txt", "token=secret", "initial")
	// Make the worktree dirty without staging.
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("modified"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := RunReplace(ReplaceRequest{
		From:       "secret",
		To:         "REDACTED",
		RepoPath:   dir,
		Yes:        true,
		AllowDirty: false,
	})
	if err == nil {
		t.Fatal("RunReplace(dirty, AllowDirty=false): expected error, got nil")
	}
	execErr, ok := err.(*gitutil.ExecError)
	if !ok {
		t.Fatalf("expected *gitutil.ExecError, got %T: %v", err, err)
	}
	if execErr.Code != exitcodes.DirtyWorktree {
		t.Errorf("code = %d, want %d", execErr.Code, exitcodes.DirtyWorktree)
	}
}

func TestRunReplace_DryRun(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "config.txt", "api-key=abc123", "initial")

	err := RunReplace(ReplaceRequest{
		From:     "abc123",
		To:       "REDACTED",
		RepoPath: dir,
		Yes:      true,
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("RunReplace(dry-run): unexpected error: %v", err)
	}

	// The string must still be in the file (dry-run makes no changes).
	content, readErr := os.ReadFile(filepath.Join(dir, "config.txt"))
	if readErr != nil {
		t.Fatalf("ReadFile: %v", readErr)
	}
	if !strings.Contains(string(content), "abc123") {
		t.Error("RunReplace(dry-run): file was unexpectedly modified")
	}
}

func TestRunReplace_DirtyAllowed_DryRun(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "file.txt", "token=secret", "initial")
	// Make dirty.
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("modified"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := RunReplace(ReplaceRequest{
		From:       "secret",
		To:         "REDACTED",
		RepoPath:   dir,
		Yes:        true,
		DryRun:     true,
		AllowDirty: true,
	})
	if err != nil {
		t.Fatalf("RunReplace(dirty, allowed, dry-run): unexpected error: %v", err)
	}
}

func TestRunReplace_FullRewrite(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "secret.txt", "password=hunter2", "initial")

	err := RunReplace(ReplaceRequest{
		From:     "hunter2",
		To:       "REDACTED",
		RepoPath: dir,
		Yes:      true,
		DryRun:   false,
		Silent:   true,
	})
	if err != nil {
		t.Fatalf("RunReplace(full): unexpected error: %v", err)
	}
}

func TestRunReplace_FullRewrite_WithBackup(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "creds.txt", "token=mysecret", "initial")

	err := RunReplace(ReplaceRequest{
		From:     "mysecret",
		To:       "REDACTED",
		RepoPath: dir,
		Yes:      true,
		DryRun:   false,
		Backup:   true,
		Silent:   true,
	})
	if err != nil {
		t.Fatalf("RunReplace(backup): unexpected error: %v", err)
	}
}

func TestRunReplace_Confirm_Accepted(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "data.txt", "sensitivevalue", "initial")
	setStdin(t, "y\n")

	err := RunReplace(ReplaceRequest{
		From:     "sensitivevalue",
		To:       "REDACTED",
		RepoPath: dir,
		Yes:      false, // triggers confirm()
		DryRun:   false,
		Silent:   true,
	})
	if err != nil {
		t.Fatalf("RunReplace(confirm yes): unexpected error: %v", err)
	}
}

func TestRunReplace_Confirm_Declined(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "data.txt", "sensitivevalue", "initial")
	setStdin(t, "n\n")

	err := RunReplace(ReplaceRequest{
		From:     "sensitivevalue",
		To:       "REDACTED",
		RepoPath: dir,
		Yes:      false, // triggers confirm()
		DryRun:   false,
	})
	if err == nil {
		t.Fatal("RunReplace(confirm declined): expected error, got nil")
	}
	execErr, ok := err.(*gitutil.ExecError)
	if !ok {
		t.Fatalf("expected *gitutil.ExecError, got %T", err)
	}
	if execErr.Code != exitcodes.UserDeclined {
		t.Errorf("code = %d, want %d", execErr.Code, exitcodes.UserDeclined)
	}
}
