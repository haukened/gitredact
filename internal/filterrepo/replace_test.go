package filterrepo

import (
	"strings"
	"testing"

	git "github.com/go-git/go-git/v5"

	"gitredact/internal/exitcodes"
	"gitredact/internal/gitutil"
)

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

func TestRunReplace_Basic(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "config.txt", "api-key=hunter2", "initial")

	if err := RunReplace(dir, "hunter2", "REDACTED", false, true); err != nil {
		t.Fatalf("RunReplace: unexpected error: %v", err)
	}

	got := fileContents(t, dir, "config.txt")
	if strings.Contains(got, "hunter2") {
		t.Errorf("RunReplace: original still present: %q", got)
	}
	if !strings.Contains(got, "REDACTED") {
		t.Errorf("RunReplace: replacement not found: %q", got)
	}
}

func TestRunReplace_WithTags(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "secret.txt", "token=abc", "initial")

	if err := RunReplace(dir, "abc", "REDACTED", true, true); err != nil {
		t.Fatalf("RunReplace (includeTags): unexpected error: %v", err)
	}

	got := fileContents(t, dir, "secret.txt")
	if strings.Contains(got, "abc") {
		t.Errorf("RunReplace (includeTags): original still present: %q", got)
	}
}

func TestRunReplace_InvalidRepo(t *testing.T) {
	dir := t.TempDir()
	err := RunReplace(dir, "from", "to", false, true)
	if err == nil {
		t.Fatal("RunReplace(non-git): expected error, got nil")
	}
	execErr, ok := err.(*gitutil.ExecError)
	if !ok {
		t.Fatalf("expected *gitutil.ExecError, got %T: %v", err, err)
	}
	if execErr.Code != exitcodes.RewriteExecution {
		t.Errorf("code = %d, want %d", execErr.Code, exitcodes.RewriteExecution)
	}
}
