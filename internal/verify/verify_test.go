package verify

import (
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"gitredact/internal/exitcodes"
	"gitredact/internal/gitutil"
)

// ---- VerifyReplace ----

func TestVerifyReplace_Pass_NoTarget(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "file.txt", "safe content only", "initial")

	if err := VerifyReplace(dir, "secret", false); err != nil {
		t.Fatalf("VerifyReplace (pass): unexpected error: %v", err)
	}
}

func TestVerifyReplace_Fail_TargetPresent(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "file.txt", "password=hunter2", "initial")

	err := VerifyReplace(dir, "hunter2", false)
	if err == nil {
		t.Fatal("VerifyReplace (fail): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "verification FAILED") {
		t.Errorf("VerifyReplace (fail): unexpected error message: %v", err)
	}
	execErr, ok := err.(*gitutil.ExecError)
	if !ok {
		t.Fatalf("expected *gitutil.ExecError, got %T", err)
	}
	if execErr.Code != exitcodes.VerificationFailed {
		t.Errorf("code = %d, want %d", execErr.Code, exitcodes.VerificationFailed)
	}
}

func TestVerifyReplace_WithLightweightTag(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "file.txt", "safe content", "initial")

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	// Create a lightweight tag.
	tagRef := plumbing.NewHashReference("refs/tags/v0.1", head.Hash())
	if err := repo.Storer.SetReference(tagRef); err != nil {
		t.Fatalf("SetReference: %v", err)
	}

	if err := VerifyReplace(dir, "secret", true); err != nil {
		t.Fatalf("VerifyReplace (lightweight tag): unexpected error: %v", err)
	}
}

func TestVerifyReplace_FailWithTag_TargetPresent(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "file.txt", "token=abc123", "initial")

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	_, err = repo.CreateTag("v1.0", head.Hash(), &git.CreateTagOptions{
		Tagger:  &object.Signature{Name: "T", Email: "t@t.com", When: time.Now()},
		Message: "release",
	})
	if err != nil {
		t.Fatalf("CreateTag: %v", err)
	}

	err = VerifyReplace(dir, "abc123", true)
	if err == nil {
		t.Fatal("VerifyReplace (fail with tag): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "verification FAILED") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestVerifyReplace_MultipleCommits_AllClean(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "a.txt", "safe-a", "first")
	commitFile(t, dir, repo, "b.txt", "safe-b", "second")

	if err := VerifyReplace(dir, "secret", false); err != nil {
		t.Fatalf("VerifyReplace (multi clean): unexpected error: %v", err)
	}
}

func TestVerifyReplace_MultipleCommits_SomeContainTarget(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "a.txt", "safe content", "first")
	commitFile(t, dir, repo, "b.txt", "token=EXPOSED", "second")

	err := VerifyReplace(dir, "EXPOSED", false)
	if err == nil {
		t.Fatal("VerifyReplace (multi, some dirty): expected error, got nil")
	}
}

func TestVerifyReplace_InvalidRepo(t *testing.T) {
	dir := t.TempDir()
	err := VerifyReplace(dir, "anything", false)
	if err == nil {
		t.Fatal("VerifyReplace(non-git): expected error, got nil")
	}
}

// TestVerifyReplace_DuplicateBlobs exercises the seenBlobs deduplication path
// and the hitBlobs fast-path within VerifyReplace.
func TestVerifyReplace_DuplicateBlobs(t *testing.T) {
	dir, repo := initRepo(t)
	// Two commits with the same file content → same blob hash.
	commitFile(t, dir, repo, "a.txt", "token=EXPOSED", "first")
	commitFile(t, dir, repo, "b.txt", "token=EXPOSED", "second")

	// "EXPOSED" is in both commits through shared blob; should fail verification.
	err := VerifyReplace(dir, "EXPOSED", false)
	if err == nil {
		t.Fatal("VerifyReplace (dup blobs): expected failure, got nil")
	}
}

// TestVerifyReplace_MultipleRefs exercises the seenCommits deduplication path
// by creating two branch refs pointing at the same commit.
func TestVerifyReplace_MultipleRefs(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "file.txt", "safe content", "initial")

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	secondRef := plumbing.NewHashReference("refs/heads/second", head.Hash())
	if err := repo.Storer.SetReference(secondRef); err != nil {
		t.Fatalf("SetReference: %v", err)
	}

	// No target present; should pass even with two branches on the same commit.
	if err := VerifyReplace(dir, "not-present", false); err != nil {
		t.Fatalf("VerifyReplace (multi refs): unexpected error: %v", err)
	}
}

// ---- VerifyDeletePath ----

func TestVerifyDeletePath_Pass_PathAbsent(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "other.txt", "content", "initial")

	if err := VerifyDeletePath(dir, "secret.pem", false); err != nil {
		t.Fatalf("VerifyDeletePath (pass): unexpected error: %v", err)
	}
}

func TestVerifyDeletePath_Fail_PathPresent(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "secret.pem", "private key", "initial")

	err := VerifyDeletePath(dir, "secret.pem", false)
	if err == nil {
		t.Fatal("VerifyDeletePath (fail): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "verification FAILED") {
		t.Errorf("VerifyDeletePath (fail): unexpected message: %v", err)
	}
	execErr, ok := err.(*gitutil.ExecError)
	if !ok {
		t.Fatalf("expected *gitutil.ExecError, got %T", err)
	}
	if execErr.Code != exitcodes.VerificationFailed {
		t.Errorf("code = %d, want %d", execErr.Code, exitcodes.VerificationFailed)
	}
}

func TestVerifyDeletePath_WithLightweightTag(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "safe.txt", "data", "initial")

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	tagRef := plumbing.NewHashReference("refs/tags/v0.1", head.Hash())
	if err := repo.Storer.SetReference(tagRef); err != nil {
		t.Fatalf("SetReference: %v", err)
	}

	if err := VerifyDeletePath(dir, "missing.txt", true); err != nil {
		t.Fatalf("VerifyDeletePath (lightweight tag): unexpected error: %v", err)
	}
}

func TestVerifyDeletePath_FailWithAnnotatedTag(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "key.pem", "data", "initial")

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	_, err = repo.CreateTag("v1.0", head.Hash(), &git.CreateTagOptions{
		Tagger:  &object.Signature{Name: "T", Email: "t@t.com", When: time.Now()},
		Message: "release",
	})
	if err != nil {
		t.Fatalf("CreateTag: %v", err)
	}

	err = VerifyDeletePath(dir, "key.pem", true)
	if err == nil {
		t.Fatal("VerifyDeletePath (fail with tag): expected error, got nil")
	}
}

func TestVerifyDeletePath_InvalidRepo(t *testing.T) {
	dir := t.TempDir()
	err := VerifyDeletePath(dir, "file.txt", false)
	if err == nil {
		t.Fatal("VerifyDeletePath(non-git): expected error, got nil")
	}
}

// TestVerifyDeletePath_MultipleRefs exercises seenCommits deduplication in
// VerifyDeletePath by creating two branch refs on the same commit.
func TestVerifyDeletePath_MultipleRefs(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "safe.txt", "data", "initial")

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	secondRef := plumbing.NewHashReference("refs/heads/second", head.Hash())
	if err := repo.Storer.SetReference(secondRef); err != nil {
		t.Fatalf("SetReference: %v", err)
	}

	// "missing.txt" not in history; should pass.
	if err := VerifyDeletePath(dir, "missing.txt", false); err != nil {
		t.Fatalf("VerifyDeletePath (multi refs): unexpected error: %v", err)
	}
}
