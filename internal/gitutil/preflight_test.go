package gitutil

import (
	"testing"
	"time"

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

// TestStringExistsInHistory_DuplicateBlobs ensures identical blob content
// appearing under multiple paths/commits does not affect correctness.
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

func TestFindStringMatchesInHistory_ReturnsAffectedFiles(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "a.txt", "prefix secret suffix", "first")
	commitFile(t, dir, repo, "b.txt", "secret", "second")

	matches, err := FindStringMatchesInHistory(dir, "secret")
	if err != nil {
		t.Fatalf("FindStringMatchesInHistory: unexpected error: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("len(matches) = %d, want 2", len(matches))
	}
	if matches[0].Path != "a.txt" {
		t.Errorf("matches[0].Path = %q, want %q", matches[0].Path, "a.txt")
	}
	if matches[1].Path != "b.txt" {
		t.Errorf("matches[1].Path = %q, want %q", matches[1].Path, "b.txt")
	}
}

func TestFindStringMatchesInHistory_EarliestByCommitTimestamp(t *testing.T) {
	dir, repo := initRepo(t)

	later := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	earlier := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)

	firstHash := commitFileAt(t, dir, repo, "a.txt", "secret", "later-time", later)
	secondHash := commitFileAt(t, dir, repo, "a.txt", "still secret", "earlier-time", earlier)

	matches, err := FindStringMatchesInHistory(dir, "secret")
	if err != nil {
		t.Fatalf("FindStringMatchesInHistory: unexpected error: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("len(matches) = %d, want 1", len(matches))
	}
	if matches[0].Path != "a.txt" {
		t.Fatalf("matches[0].Path = %q, want %q", matches[0].Path, "a.txt")
	}
	if matches[0].FirstMatchCommit != secondHash.String() {
		t.Fatalf("FirstMatchCommit = %q, want %q (first was %q)", matches[0].FirstMatchCommit, secondHash.String(), firstHash.String())
	}
	if !matches[0].FirstMatchTime.Equal(earlier) {
		t.Fatalf("FirstMatchTime = %v, want %v", matches[0].FirstMatchTime, earlier)
	}
}

func TestStringExistsInHistory_FindsOnNonHeadBranch(t *testing.T) {
	dir, repo := initRepo(t)

	commitFile(t, dir, repo, "safe.txt", "safe", "safe")
	head1, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}

	commitFile(t, dir, repo, "secrets.txt", "token=secret", "secret")
	head2, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}

	// Move the current branch back to head1, and create a different branch pointing
	// to head2. The secret commit is now only reachable from the new ref.
	if err := repo.Storer.SetReference(plumbing.NewHashReference(head1.Name(), head1.Hash())); err != nil {
		t.Fatalf("SetReference(head1): %v", err)
	}
	if err := repo.Storer.SetReference(plumbing.NewHashReference("refs/heads/other", head2.Hash())); err != nil {
		t.Fatalf("SetReference(other): %v", err)
	}

	found, err := StringExistsInHistory(dir, "token=secret")
	if err != nil {
		t.Fatalf("StringExistsInHistory: unexpected error: %v", err)
	}
	if !found {
		t.Fatal("StringExistsInHistory: expected match on non-HEAD branch")
	}
}

func TestStringExistsInHistory_SubdirFile(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "dir/subdir/secret.txt", "the secret is here", "nested")

	found, err := StringExistsInHistory(dir, "secret")
	if err != nil {
		t.Fatalf("StringExistsInHistory: unexpected error: %v", err)
	}
	if !found {
		t.Fatal("StringExistsInHistory: expected match in nested path")
	}
}
