package gitutil

import (
	"os"
	"path/filepath"
	"testing"

	"gitredact/internal/exitcodes"
)

func TestRepoError_Error(t *testing.T) {
	e := &RepoError{Path: "/some/path", Message: "not a git repo"}
	if got := e.Error(); got != "not a git repo" {
		t.Errorf("RepoError.Error() = %q, want %q", got, "not a git repo")
	}
}

func TestResolveRoot_ExplicitGitRepo(t *testing.T) {
	dir, _ := initRepo(t)
	root, err := ResolveRoot(dir)
	if err != nil {
		t.Fatalf("ResolveRoot(%q): unexpected error: %v", dir, err)
	}
	if root == "" {
		t.Error("ResolveRoot: expected non-empty root")
	}
}

func TestResolveRoot_SubdirOfGitRepo(t *testing.T) {
	dir, _ := initRepo(t)
	subdir := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	root, err := ResolveRoot(subdir)
	if err != nil {
		t.Fatalf("ResolveRoot(subdir): unexpected error: %v", err)
	}
	if root == "" {
		t.Error("ResolveRoot(subdir): expected non-empty root")
	}
}

func TestResolveRoot_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := ResolveRoot(dir)
	if err == nil {
		t.Fatal("ResolveRoot(non-git): expected error, got nil")
	}
	execErr, ok := err.(*ExecError)
	if !ok {
		t.Fatalf("ResolveRoot(non-git): expected *ExecError, got %T: %v", err, err)
	}
	if execErr.Code != exitcodes.RepoValidation {
		t.Errorf("ResolveRoot(non-git): code = %d, want %d", execErr.Code, exitcodes.RepoValidation)
	}
}

func TestResolveRoot_EmptyPath_InsideGitRepo(t *testing.T) {
	dir, _ := initRepo(t)
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(old); err != nil {
			t.Errorf("chdir back: %v", err)
		}
	}()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q): %v", dir, err)
	}
	root, err := ResolveRoot("")
	if err != nil {
		t.Fatalf("ResolveRoot(\"\"): unexpected error: %v", err)
	}
	if root == "" {
		t.Error("ResolveRoot(\"\"): expected non-empty root")
	}
}
