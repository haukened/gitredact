package gitutil

import (
	"testing"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"gitredact/internal/exitcodes"
)

func TestCreateBackupRef(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "README.md", "content", "initial")

	refName := "refs/gitredact-backup/test-1234567890"
	if err := CreateBackupRef(dir, refName); err != nil {
		t.Fatalf("CreateBackupRef: unexpected error: %v", err)
	}

	// Verify the ref was created and points to a non-zero hash.
	ref, err := repo.Reference(plumbing.ReferenceName(refName), false)
	if err != nil {
		t.Fatalf("repo.Reference: ref not found after creation: %v", err)
	}
	if ref.Hash().IsZero() {
		t.Error("CreateBackupRef: backup ref has zero hash")
	}
}

func TestCreateBackupRef_MatchesHEAD(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "file.txt", "data", "initial")

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("repo.Head: %v", err)
	}

	refName := "refs/gitredact-backup/head-check"
	if err := CreateBackupRef(dir, refName); err != nil {
		t.Fatalf("CreateBackupRef: unexpected error: %v", err)
	}

	// Open fresh to avoid any caching.
	repo2, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	ref, err := repo2.Reference(plumbing.ReferenceName(refName), false)
	if err != nil {
		t.Fatalf("repo.Reference: %v", err)
	}
	if ref.Hash() != head.Hash() {
		t.Errorf("backup ref hash %s != HEAD hash %s", ref.Hash(), head.Hash())
	}
}

func TestCreateBackupRef_NoCommits(t *testing.T) {
	// A repo with no commits has no HEAD, so CreateBackupRef must fail.
	dir, _ := initRepo(t)
	err := CreateBackupRef(dir, "refs/gitredact-backup/no-head")
	if err == nil {
		t.Fatal("CreateBackupRef(no commits): expected error, got nil")
	}
	execErr, ok := err.(*ExecError)
	if !ok {
		t.Fatalf("expected *ExecError, got %T: %v", err, err)
	}
	if execErr.Code != exitcodes.RewriteExecution {
		t.Errorf("code = %d, want %d", execErr.Code, exitcodes.RewriteExecution)
	}
}

func TestCreateBackupRef_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	err := CreateBackupRef(dir, "refs/gitredact-backup/test")
	if err == nil {
		t.Fatal("CreateBackupRef(non-git): expected error, got nil")
	}
	execErr, ok := err.(*ExecError)
	if !ok {
		t.Fatalf("expected *ExecError, got %T: %v", err, err)
	}
	if execErr.Code != exitcodes.RewriteExecution {
		t.Errorf("code = %d, want %d", execErr.Code, exitcodes.RewriteExecution)
	}
}
