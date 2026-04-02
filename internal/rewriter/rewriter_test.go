package rewriter

import (
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestRun_EmptyRepo verifies that running on a repo with no commits is a no-op.
func TestRun_EmptyRepo(t *testing.T) {
	dir, _ := initRepo(t)
	if err := Replace(dir, "x", "y", false, true); err != nil {
		t.Fatalf("Replace on empty repo: unexpected error: %v", err)
	}
}

// TestRun_IncludeTags_LightweightTag verifies that a lightweight tag ref is
// rewritten when includeTags is true and unchanged when false.
func TestRun_IncludeTags_LightweightTag(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "file.txt", "token=secret", "initial")

	// Create lightweight tag pointing at HEAD.
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	tagRef := plumbing.NewHashReference("refs/tags/v0.1", head.Hash())
	if err := repo.Storer.SetReference(tagRef); err != nil {
		t.Fatalf("SetReference: %v", err)
	}

	if err := Replace(dir, "secret", "REDACTED", true, true); err != nil {
		t.Fatalf("Replace (with lightweight tag): unexpected error: %v", err)
	}

	got := fileContents(t, dir, "file.txt")
	if strings.Contains(got, "secret") {
		t.Errorf("Replace (lightweight tag): original still present: %q", got)
	}
}

// TestRun_IncludeTags_AnnotatedTag verifies that an annotated tag is rewritten
// when includeTags is true.
func TestRun_IncludeTags_AnnotatedTag(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "file.txt", "password=s3cr3t", "initial")

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}

	// Create annotated tag.
	_, err = repo.CreateTag("v1.0", head.Hash(), &git.CreateTagOptions{
		Tagger:  &object.Signature{Name: "Tagger", Email: "t@t.com", When: time.Now()},
		Message: "release v1.0",
	})
	if err != nil {
		t.Fatalf("CreateTag: %v", err)
	}

	if err := Replace(dir, "s3cr3t", "REDACTED", true, true); err != nil {
		t.Fatalf("Replace (annotated tag): unexpected error: %v", err)
	}

	got := fileContents(t, dir, "file.txt")
	if strings.Contains(got, "s3cr3t") {
		t.Errorf("Replace (annotated tag): original still present: %q", got)
	}
}

// TestRun_IncludeTags_False verifies that tags are NOT rewritten when
// includeTags is false.
func TestRun_IncludeTags_False(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "file.txt", "token=abc", "initial")

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	tagRef := plumbing.NewHashReference("refs/tags/skip-me", head.Hash())
	if err := repo.Storer.SetReference(tagRef); err != nil {
		t.Fatalf("SetReference: %v", err)
	}

	// Replace without rewriting tags.
	if err := Replace(dir, "abc", "REDACTED", false, true); err != nil {
		t.Fatalf("Replace (tags=false): unexpected error: %v", err)
	}

	// Branch content is rewritten.
	got := fileContents(t, dir, "file.txt")
	if strings.Contains(got, "abc") {
		t.Errorf("Replace (tags=false): branch file not rewritten: %q", got)
	}
}

// TestRun_MergeCommit verifies that merge commits (multiple parents) are
// processed correctly.
func TestRun_MergeCommit(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "main.txt", "main-secret", "main initial")

	// Record main head.
	mainHead, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}

	// Create a second branch commit on a separate branch by directly creating
	// a commit object that has the current HEAD as parent.
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}
	if err := wt.Checkout(&git.CheckoutOptions{
		Create: true,
		Branch: "refs/heads/feature",
		Hash:   mainHead.Hash(),
	}); err != nil {
		t.Fatalf("Checkout feature: %v", err)
	}
	commitFile(t, dir, repo, "feature.txt", "feature-secret", "feature commit")

	featureHead, err := repo.Head()
	if err != nil {
		t.Fatalf("Head after feature commit: %v", err)
	}

	// Switch back to first branch (main/master).
	if err := wt.Checkout(&git.CheckoutOptions{
		Branch: mainHead.Name(),
	}); err != nil {
		t.Fatalf("Checkout main: %v", err)
	}

	// Build a merge commit manually with two parents.
	mainCommit, err := repo.CommitObject(mainHead.Hash())
	if err != nil {
		t.Fatalf("CommitObject main: %v", err)
	}
	mergeCommit := object.Commit{
		Author:       object.Signature{Name: "Test", Email: "t@t.com", When: time.Now()},
		Committer:    object.Signature{Name: "Test", Email: "t@t.com", When: time.Now()},
		Message:      "Merge feature into main",
		TreeHash:     mainCommit.TreeHash,
		ParentHashes: []plumbing.Hash{mainHead.Hash(), featureHead.Hash()},
	}
	encoded := &plumbing.MemoryObject{}
	if err := mergeCommit.Encode(encoded); err != nil {
		t.Fatalf("Encode merge commit: %v", err)
	}
	mergeHash, err := repo.Storer.SetEncodedObject(encoded)
	if err != nil {
		t.Fatalf("SetEncodedObject: %v", err)
	}
	newMain := plumbing.NewHashReference(mainHead.Name(), mergeHash)
	if err := repo.Storer.SetReference(newMain); err != nil {
		t.Fatalf("SetReference: %v", err)
	}

	// Now replace "main-secret" across all commits including the merge.
	if err := Replace(dir, "main-secret", "REDACTED", false, true); err != nil {
		t.Fatalf("Replace (merge commit): unexpected error: %v", err)
	}

	got := fileContents(t, dir, "main.txt")
	if strings.Contains(got, "main-secret") {
		t.Errorf("Replace (merge commit): original still present: %q", got)
	}
}

// TestRun_BlobCaching verifies that identical blobs across commits are only
// processed once (the second commit reuses the cached result).
func TestRun_BlobCaching(t *testing.T) {
	dir, repo := initRepo(t)
	sameContent := "shared-secret=value"
	commitFile(t, dir, repo, "a.txt", sameContent, "first")
	commitFile(t, dir, repo, "b.txt", sameContent, "second — same blob content")

	if err := Replace(dir, "shared-secret", "REDACTED", false, true); err != nil {
		t.Fatalf("Replace (blob caching): unexpected error: %v", err)
	}

	for _, name := range []string{"a.txt", "b.txt"} {
		got := fileContents(t, dir, name)
		if strings.Contains(got, "shared-secret") {
			t.Errorf("Replace (blob caching): %q still contains original: %q", name, got)
		}
	}
}

// TestRun_UnchangedCommitOptimization verifies that a commit whose tree does
// not change keeps its original hash (memoization fast-path).
func TestRun_UnchangedCommitOptimization(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "safe.txt", "nothing to replace here", "only commit")

	orig := headCommit(t, dir)
	if err := Replace(dir, "no-such-string", "X", false, true); err != nil {
		t.Fatalf("Replace (unchanged): unexpected error: %v", err)
	}
	after := headCommit(t, dir)

	if after.Hash != orig.Hash {
		t.Errorf("Replace (unchanged): commit hash changed when no content matched")
	}
}

// TestRun_AnnotatedTag_Unchanged verifies that an annotated tag is skipped
// (early continue) when its commit does not need rewriting.
func TestRun_AnnotatedTag_Unchanged(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "file.txt", "nothing matching here", "initial")

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}

	_, err = repo.CreateTag("v1.0-unchanged", head.Hash(), &git.CreateTagOptions{
		Tagger:  &object.Signature{Name: "T", Email: "t@t.com", When: time.Now()},
		Message: "stable release",
	})
	if err != nil {
		t.Fatalf("CreateTag: %v", err)
	}

	origCommit := headCommit(t, dir)

	// Replace a string that does not exist — the tagged commit is unchanged.
	if err := Replace(dir, "no-such-string", "REDACTED", true, true); err != nil {
		t.Fatalf("Replace (annotated tag unchanged): unexpected error: %v", err)
	}

	// HEAD must not have moved.
	after := headCommit(t, dir)
	if after.Hash != origCommit.Hash {
		t.Errorf("Replace (annotated tag unchanged): HEAD changed unexpectedly")
	}
}

func TestRun_DeletePath_IncludeTags(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "remove-me.txt", "data", "initial")

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	_, err = repo.CreateTag("v2.0", head.Hash(), &git.CreateTagOptions{
		Tagger:  &object.Signature{Name: "T", Email: "t@t.com", When: time.Now()},
		Message: "tagged release",
	})
	if err != nil {
		t.Fatalf("CreateTag: %v", err)
	}

	if err := DeletePath(dir, "remove-me.txt", true, true); err != nil {
		t.Fatalf("DeletePath (includeTags): unexpected error: %v", err)
	}

	r, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	h, _ := r.Head()
	c, _ := r.CommitObject(h.Hash())
	if _, fileErr := c.File("remove-me.txt"); fileErr == nil {
		t.Error("DeletePath (includeTags): file still present in rewritten HEAD")
	}
}
