package filterrepo

import (
	"testing"

	git "github.com/go-git/go-git/v5"

	"gitredact/internal/exitcodes"
	"gitredact/internal/gitutil"
)

func TestRunDeletePath_Basic(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "remove-me.key", "private key data", "initial")

	if err := RunDeletePath(dir, "remove-me.key", false, true); err != nil {
		t.Fatalf("RunDeletePath: unexpected error: %v", err)
	}

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
	if _, fileErr := c.File("remove-me.key"); fileErr == nil {
		t.Error("RunDeletePath: file still present in rewritten HEAD")
	}
}

func TestRunDeletePath_WithTags(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "drop.txt", "sensitive", "initial")

	if err := RunDeletePath(dir, "drop.txt", true, true); err != nil {
		t.Fatalf("RunDeletePath (includeTags): unexpected error: %v", err)
	}

	r, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	head, _ := r.Head()
	c, _ := r.CommitObject(head.Hash())
	if _, fileErr := c.File("drop.txt"); fileErr == nil {
		t.Error("RunDeletePath (includeTags): file still present")
	}
}

func TestRunDeletePath_InvalidRepo(t *testing.T) {
	dir := t.TempDir()
	err := RunDeletePath(dir, "file.txt", false, true)
	if err == nil {
		t.Fatal("RunDeletePath(non-git): expected error, got nil")
	}
	execErr, ok := err.(*gitutil.ExecError)
	if !ok {
		t.Fatalf("expected *gitutil.ExecError, got %T: %v", err, err)
	}
	if execErr.Code != exitcodes.RewriteExecution {
		t.Errorf("code = %d, want %d", execErr.Code, exitcodes.RewriteExecution)
	}
}
