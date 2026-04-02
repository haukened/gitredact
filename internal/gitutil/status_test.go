package gitutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsDirty_Clean(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "README.md", "hello", "initial")

	dirty, err := IsDirty(dir)
	if err != nil {
		t.Fatalf("IsDirty: unexpected error: %v", err)
	}
	if dirty {
		t.Error("IsDirty: expected clean repo, got dirty")
	}
}

func TestIsDirty_Dirty(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "README.md", "hello", "initial")

	// Modify a tracked file without staging.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("modified"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	dirty, err := IsDirty(dir)
	if err != nil {
		t.Fatalf("IsDirty: unexpected error: %v", err)
	}
	if !dirty {
		t.Error("IsDirty: expected dirty repo, got clean")
	}
}

func TestIsDirty_NewUntrackedFile(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "README.md", "hello", "initial")

	// Create untracked file.
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	dirty, err := IsDirty(dir)
	if err != nil {
		t.Fatalf("IsDirty: unexpected error: %v", err)
	}
	if !dirty {
		t.Error("IsDirty: expected dirty (untracked file), got clean")
	}
}

func TestIsDirty_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := IsDirty(dir)
	if err == nil {
		t.Fatal("IsDirty(non-git): expected error, got nil")
	}
}
