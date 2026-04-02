package rewriter

import (
	"testing"

	git "github.com/go-git/go-git/v5"
)

func TestDeletePath_Basic(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "secret.pem", "private key content", "initial")

	if err := DeletePath(dir, "secret.pem", false, true); err != nil {
		t.Fatalf("DeletePath: unexpected error: %v", err)
	}

	// The file must not exist in the rewritten HEAD commit.
	r, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	head, err := r.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	c, err := r.CommitObject(head.Hash())
	if err != nil {
		t.Fatalf("CommitObject: %v", err)
	}
	if _, fileErr := c.File("secret.pem"); fileErr == nil {
		t.Error("DeletePath: file still present in rewritten HEAD")
	}
}

func TestDeletePath_FileNotPresent(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "other.txt", "safe content", "initial")

	origCommit := headCommit(t, dir)
	if err := DeletePath(dir, "secret.pem", false, true); err != nil {
		t.Fatalf("DeletePath (not present): unexpected error: %v", err)
	}

	// Nothing to delete, commit hash should be unchanged.
	afterCommit := headCommit(t, dir)
	if afterCommit.Hash != origCommit.Hash {
		t.Errorf("DeletePath (not present): commit changed unexpectedly")
	}
}

func TestDeletePath_NestedPath(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "config/secrets.yaml", "token: abc123", "initial")
	commitFile(t, dir, repo, "config/safe.yaml", "version: 1", "second")

	if err := DeletePath(dir, "config/secrets.yaml", false, true); err != nil {
		t.Fatalf("DeletePath (nested): unexpected error: %v", err)
	}

	// secrets.yaml should be gone, safe.yaml should remain.
	r, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	head, _ := r.Head()
	c, _ := r.CommitObject(head.Hash())
	if _, fileErr := c.File("config/secrets.yaml"); fileErr == nil {
		t.Error("DeletePath (nested): secrets.yaml still present")
	}
	if _, fileErr := c.File("config/safe.yaml"); fileErr != nil {
		t.Errorf("DeletePath (nested): safe.yaml unexpectedly removed: %v", fileErr)
	}
}

func TestDeletePath_WithVerboseOutput(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "drop.txt", "data", "init")

	if err := DeletePath(dir, "drop.txt", false, false); err != nil {
		t.Fatalf("DeletePath (verbose): unexpected error: %v", err)
	}
}

func TestDeletePath_InvalidRepo(t *testing.T) {
	dir := t.TempDir()
	err := DeletePath(dir, "file.txt", false, true)
	if err == nil {
		t.Fatal("DeletePath(non-git): expected error, got nil")
	}
}
