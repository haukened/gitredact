package app

import (
	"os"
	"path/filepath"
	"testing"

	"gitredact/internal/exitcodes"
	"gitredact/internal/gitutil"
)

func TestRunDeletePath_InvalidRepo(t *testing.T) {
	dir := t.TempDir()
	err := RunDeletePath(DeletePathRequest{
		Path:     "secret.pem",
		RepoPath: dir,
		Yes:      true,
		DryRun:   true,
	})
	if err == nil {
		t.Fatal("RunDeletePath(non-git): expected error, got nil")
	}
}

func TestRunDeletePath_PathNotInHistory(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "other.txt", "safe content", "initial")

	err := RunDeletePath(DeletePathRequest{
		Path:     "secret.pem",
		RepoPath: dir,
		Yes:      true,
		DryRun:   true,
	})
	if err == nil {
		t.Fatal("RunDeletePath(path not found): expected error, got nil")
	}
	execErr, ok := err.(*gitutil.ExecError)
	if !ok {
		t.Fatalf("expected *gitutil.ExecError, got %T: %v", err, err)
	}
	if execErr.Code != exitcodes.NoMatchesFound {
		t.Errorf("code = %d, want %d", execErr.Code, exitcodes.NoMatchesFound)
	}
}

func TestRunDeletePath_DirtyWorktree_Rejected(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "secret.pem", "key data", "initial")
	// Make dirty.
	if err := os.WriteFile(filepath.Join(dir, "secret.pem"), []byte("modified"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := RunDeletePath(DeletePathRequest{
		Path:       "secret.pem",
		RepoPath:   dir,
		Yes:        true,
		AllowDirty: false,
	})
	if err == nil {
		t.Fatal("RunDeletePath(dirty, AllowDirty=false): expected error, got nil")
	}
	execErr, ok := err.(*gitutil.ExecError)
	if !ok {
		t.Fatalf("expected *gitutil.ExecError, got %T: %v", err, err)
	}
	if execErr.Code != exitcodes.DirtyWorktree {
		t.Errorf("code = %d, want %d", execErr.Code, exitcodes.DirtyWorktree)
	}
}

func TestRunDeletePath_DryRun(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "private.key", "key content", "initial")

	err := RunDeletePath(DeletePathRequest{
		Path:     "private.key",
		RepoPath: dir,
		Yes:      true,
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("RunDeletePath(dry-run): unexpected error: %v", err)
	}
}

func TestRunDeletePath_DirtyAllowed_DryRun(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "key.pem", "data", "initial")
	// Make dirty.
	if err := os.WriteFile(filepath.Join(dir, "key.pem"), []byte("modified"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := RunDeletePath(DeletePathRequest{
		Path:       "key.pem",
		RepoPath:   dir,
		Yes:        true,
		DryRun:     true,
		AllowDirty: true,
	})
	if err != nil {
		t.Fatalf("RunDeletePath(dirty, allowed, dry-run): unexpected error: %v", err)
	}
}

func TestRunDeletePath_FullRewrite(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "creds.pem", "private key data", "initial")

	err := RunDeletePath(DeletePathRequest{
		Path:     "creds.pem",
		RepoPath: dir,
		Yes:      true,
		DryRun:   false,
		Silent:   true,
	})
	if err != nil {
		t.Fatalf("RunDeletePath(full): unexpected error: %v", err)
	}
}

func TestRunDeletePath_FullRewrite_WithBackup(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "api.key", "key=abc123", "initial")

	err := RunDeletePath(DeletePathRequest{
		Path:     "api.key",
		RepoPath: dir,
		Yes:      true,
		DryRun:   false,
		Backup:   true,
		Silent:   true,
	})
	if err != nil {
		t.Fatalf("RunDeletePath(backup): unexpected error: %v", err)
	}
}

func TestRunDeletePath_Confirm_Accepted(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "secret.txt", "data", "initial")
	setStdin(t, "y\n")

	err := RunDeletePath(DeletePathRequest{
		Path:     "secret.txt",
		RepoPath: dir,
		Yes:      false, // triggers confirm()
		DryRun:   false,
		Silent:   true,
	})
	if err != nil {
		t.Fatalf("RunDeletePath(confirm yes): unexpected error: %v", err)
	}
}

func TestRunDeletePath_Confirm_Declined(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "secret.txt", "data", "initial")
	setStdin(t, "n\n")

	err := RunDeletePath(DeletePathRequest{
		Path:     "secret.txt",
		RepoPath: dir,
		Yes:      false, // triggers confirm()
		DryRun:   false,
	})
	if err == nil {
		t.Fatal("RunDeletePath(confirm declined): expected error, got nil")
	}
	execErr, ok := err.(*gitutil.ExecError)
	if !ok {
		t.Fatalf("expected *gitutil.ExecError, got %T", err)
	}
	if execErr.Code != exitcodes.UserDeclined {
		t.Errorf("code = %d, want %d", execErr.Code, exitcodes.UserDeclined)
	}
}
