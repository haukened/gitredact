package gitutil

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
// commit with the given message. filename may contain path separators.
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
