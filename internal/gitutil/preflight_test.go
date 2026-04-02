package gitutil

import (
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
)

func TestStringExistsInHistory_Found(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "secrets.txt", "my-api-key=abc123", "initial")

	found, err := StringExistsInHistory(dir, "my-api-key")
	if err != nil {
		t.Fatalf("StringExistsInHistory: unexpected error: %v", err)
	}
	if !found {
		t.Error("StringExistsInHistory: expected string to be found")
	}
}

func TestStringExistsInHistory_NotFound(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "file.txt", "normal content only", "initial")

	found, err := StringExistsInHistory(dir, "my-api-key")
	if err != nil {
		t.Fatalf("StringExistsInHistory: unexpected error: %v", err)
	}
	if found {
		t.Error("StringExistsInHistory: expected string not to be found")
	}
}

func TestStringExistsInHistory_MultipleCommits(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "file.txt", "harmless", "first")
	commitFile(t, dir, repo, "secrets.txt", "TOKEN=supersecret", "second")

	found, err := StringExistsInHistory(dir, "supersecret")
	if err != nil {
		t.Fatalf("StringExistsInHistory: unexpected error: %v", err)
	}
	if !found {
		t.Error("StringExistsInHistory: expected string found in second commit")
	}
}

func TestStringExistsInHistory_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := StringExistsInHistory(dir, "anything")
	if err == nil {
		t.Fatal("StringExistsInHistory(non-git): expected error, got nil")
	}
}

func TestPathExistsInHistory_Found(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "key.txt", "key content", "initial")

	found, err := PathExistsInHistory(dir, "key.txt")
	if err != nil {
		t.Fatalf("PathExistsInHistory: unexpected error: %v", err)
	}
	if !found {
		t.Error("PathExistsInHistory: expected path to be found")
	}
}

func TestPathExistsInHistory_NotFound(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "other.txt", "other content", "initial")

	found, err := PathExistsInHistory(dir, "key.txt")
	if err != nil {
		t.Fatalf("PathExistsInHistory: unexpected error: %v", err)
	}
	if found {
		t.Error("PathExistsInHistory: expected path not to be found")
	}
}

func TestPathExistsInHistory_MultipleCommits(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "normal.txt", "safe", "first")
	commitFile(t, dir, repo, "secret.pem", "private key", "second")

	found, err := PathExistsInHistory(dir, "secret.pem")
	if err != nil {
		t.Fatalf("PathExistsInHistory: unexpected error: %v", err)
	}
	if !found {
		t.Error("PathExistsInHistory: expected path found in second commit")
	}
}

func TestPathExistsInHistory_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := PathExistsInHistory(dir, "file.txt")
	if err == nil {
		t.Fatal("PathExistsInHistory(non-git): expected error, got nil")
	}
}

// TestStringExistsInHistory_DuplicateBlobs ensures the blob-deduplication
// path (seen[f.Hash]) is exercised. Two commits share the same blob hash when
// two different files have identical content.
func TestStringExistsInHistory_DuplicateBlobs(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "a.txt", "identical-content", "first")
	commitFile(t, dir, repo, "b.txt", "identical-content", "second") // same blob hash as a.txt

	// Looking for something not present; just confirm no error and correct result.
	found, err := StringExistsInHistory(dir, "no-such-string")
	if err != nil {
		t.Fatalf("StringExistsInHistory (dup blobs): unexpected error: %v", err)
	}
	if found {
		t.Error("StringExistsInHistory (dup blobs): expected not found")
	}
}

// TestPathExistsInHistory_DuplicateCommits ensures the commit-deduplication
// path (seen[c.Hash]) is exercised by having two branches point to the same commit.
func TestPathExistsInHistory_DuplicateCommits(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "file.txt", "content", "initial")

	// Create a second branch pointing at the same HEAD commit.
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	secondRef := plumbing.NewHashReference("refs/heads/second", head.Hash())
	if err := repo.Storer.SetReference(secondRef); err != nil {
		t.Fatalf("SetReference: %v", err)
	}

	// "file.txt" should still be found even with two branches on same commit.
	found, err := PathExistsInHistory(dir, "file.txt")
	if err != nil {
		t.Fatalf("PathExistsInHistory (dup commits): unexpected error: %v", err)
	}
	if !found {
		t.Error("PathExistsInHistory (dup commits): expected path to be found")
	}
}
