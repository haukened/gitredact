package rewriter

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// initRepo creates a temporary git repository and returns its path and the
// Repository handle. No commits are made.
func initRepo(t *testing.T) (dir string, repo *git.Repository) {
	t.Helper()
	dir = t.TempDir()
	var err error
	repo, err = git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("git.PlainInit: %v", err)
	}
	return
}

// commitFile writes content to filename inside dir, stages it, and creates a
// commit. filename may use slash separators for nested paths.
func commitFile(t *testing.T, dir string, repo *git.Repository, filename, content, message string) {
	t.Helper()
	fullPath := filepath.Join(dir, filepath.FromSlash(filename))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}
	relPath := filepath.ToSlash(filename)
	if _, err := wt.Add(relPath); err != nil {
		t.Fatalf("wt.Add(%q): %v", relPath, err)
	}
	_, err = wt.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("wt.Commit(%q): %v", message, err)
	}
}

// fileContents returns the contents of filename in the HEAD commit of the
// repository rooted at dir.
func fileContents(t *testing.T, dir, filename string) string {
	t.Helper()
	repo, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		t.Fatalf("CommitObject: %v", err)
	}
	f, err := commit.File(filename)
	if err != nil {
		t.Fatalf("commit.File(%q): %v", filename, err)
	}
	contents, err := f.Contents()
	if err != nil {
		t.Fatalf("file.Contents: %v", err)
	}
	return contents
}

// headCommit returns the commit at HEAD for the repository rooted at dir.
func headCommit(t *testing.T, dir string) *object.Commit {
	t.Helper()
	repo, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		t.Fatalf("CommitObject: %v", err)
	}
	return commit
}
