package app

import (
	"fmt"
	"strings"
	"time"

	"gitredact/internal/exitcodes"
	"gitredact/internal/filterrepo"
	"gitredact/internal/gitutil"
	"gitredact/internal/output"
	"gitredact/internal/plan"
	"gitredact/internal/verify"
)

// DeletePathRequest holds all inputs for the delete-path operation.
type DeletePathRequest struct {
	Path        string
	RepoPath    string
	DryRun      bool
	Yes         bool
	IncludeTags bool
	AllowDirty  bool
	Backup      bool
	Silent      bool
}

// RunDeletePath orchestrates the full delete-path workflow.
func RunDeletePath(req DeletePathRequest) error {
	// 1. Dependency check
	if err := gitutil.CheckDeps(); err != nil {
		return err
	}

	// 2. Resolve repo root
	root, err := gitutil.ResolveRoot(req.RepoPath)
	if err != nil {
		return err
	}
	output.Verbose("repo root: %s", root)

	// 3. Dirty worktree check
	dirty, err := gitutil.IsDirty(root)
	if err != nil {
		return err
	}
	if dirty && !req.AllowDirty {
		return &gitutil.ExecError{
			Code:    exitcodes.DirtyWorktree,
			Message: "worktree has uncommitted changes; use --allow-dirty to proceed",
		}
	}

	// 4. Preflight: confirm path exists in reachable history
	output.Verbose("running preflight check...")
	stdout, _, err := gitutil.Run(root, "git", "log", "--all", "--oneline", "--", req.Path)
	if err != nil || strings.TrimSpace(stdout) == "" {
		return &gitutil.ExecError{
			Code:    exitcodes.NoMatchesFound,
			Message: fmt.Sprintf("preflight: path %q not found in reachable history", req.Path),
		}
	}

	// 5. Build plan
	backupRef := ""
	if req.Backup {
		backupRef = fmt.Sprintf("refs/gitredact-backup/%d", time.Now().Unix())
	}

	filterRepoCmd := fmt.Sprintf("git-filter-repo --path %s --invert-paths --force", req.Path)
	if !req.IncludeTags {
		filterRepoCmd += " --refs refs/heads/*"
	}

	p := plan.Plan{
		RepoRoot:      root,
		Operation:     "delete-path",
		Params:        map[string]string{"path": req.Path},
		IsDirty:       dirty,
		IncludeTags:   req.IncludeTags,
		BackupEnabled: req.Backup,
		BackupRef:     backupRef,
		Commands:      []string{filterRepoCmd},
	}

	// 6. Print plan; exit here if dry-run (zero side effects)
	plan.Print(p)
	if req.DryRun {
		output.Print("dry-run: no changes made")
		return nil
	}

	// 7. Interactive confirmation
	if !req.Yes {
		if err := confirm(); err != nil {
			return err
		}
	}

	// 8. Warnings
	output.Warn("Git history will be rewritten. All commit hashes will change.")
	output.Warn("Collaborators must re-clone or hard reset after you force-push.")

	// 9. Create backup ref (after confirmation, before rewrite)
	if req.Backup {
		output.Print("creating backup ref: %s", backupRef)
		_, stderr, err := gitutil.Run(root, "git", "update-ref", backupRef, "HEAD")
		if err != nil {
			return &gitutil.ExecError{
				Code:    exitcodes.RewriteExecution,
				Message: fmt.Sprintf("failed to create backup ref: %s", stderr),
			}
		}
	}

	// 10. Run filter-repo
	output.Print("executing rewrite...")
	if err := filterrepo.RunDeletePath(root, req.Path, req.IncludeTags, req.Silent); err != nil {
		return err
	}

	// 11. Strict verification
	if err := verify.VerifyDeletePath(root, req.Path); err != nil {
		return err
	}

	// 12. Success
	output.Print("rewrite complete and verified.")
	if req.Backup {
		output.Print("backup ref: %s", backupRef)
		output.Print("  to restore: git checkout -b restore-branch %s", backupRef)
	}
	return nil
}
