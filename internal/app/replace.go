package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"gitredact/internal/exitcodes"
	"gitredact/internal/filterrepo"
	"gitredact/internal/gitutil"
	"gitredact/internal/output"
	"gitredact/internal/plan"
	"gitredact/internal/verify"
)

// ReplaceRequest holds all inputs for the replace operation.
type ReplaceRequest struct {
	From        string
	To          string
	RepoPath    string
	DryRun      bool
	Yes         bool
	IncludeTags bool
	AllowDirty  bool
	Backup      bool
	ShowFiles   bool
	Silent      bool
}

// RunReplace orchestrates the full replace workflow.
func RunReplace(req ReplaceRequest) error {
	// 1. Resolve repo root
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

	// 4. Preflight: confirm the string exists somewhere in reachable history
	output.Verbose("running preflight check...")
	matches := []gitutil.StringMatchLocation(nil)
	if req.ShowFiles {
		matches, err = gitutil.FindStringMatchesInHistory(root, req.From)
		if err != nil {
			return &gitutil.ExecError{
				Code:    exitcodes.NoMatchesFound,
				Message: fmt.Sprintf("preflight: could not scan history: %v", err),
			}
		}
		if len(matches) == 0 {
			return &gitutil.ExecError{
				Code:    exitcodes.NoMatchesFound,
				Message: fmt.Sprintf("preflight: string %q not found in reachable history", req.From),
			}
		}
	} else {
		found, err := gitutil.StringExistsInHistory(root, req.From)
		if err != nil {
			return &gitutil.ExecError{
				Code:    exitcodes.NoMatchesFound,
				Message: fmt.Sprintf("preflight: could not scan history: %v", err),
			}
		}
		if !found {
			return &gitutil.ExecError{
				Code:    exitcodes.NoMatchesFound,
				Message: fmt.Sprintf("preflight: string %q not found in reachable history", req.From),
			}
		}
	}

	// 5. Build plan
	backupRef := ""
	if req.Backup {
		backupRef = fmt.Sprintf("refs/gitredact-backup/%d", time.Now().Unix())
	}

	rewriteCmd := "gitredact rewriter.Replace (pure Go)"
	if !req.IncludeTags {
		rewriteCmd += " [branches only]"
	}

	affectedFiles := []plan.AffectedFile(nil)
	if req.ShowFiles {
		affectedFiles = make([]plan.AffectedFile, 0, len(matches))
		for _, m := range matches {
			affectedFiles = append(affectedFiles, plan.AffectedFile{
				Path:              m.Path,
				FirstMatchCommit:  m.FirstMatchCommit,
				FirstMatchSummary: m.FirstMatchSummary,
			})
		}
	}

	p := plan.Plan{
		RepoRoot:      root,
		Operation:     "replace",
		Params:        map[string]string{"from": req.From, "to": req.To},
		IsDirty:       dirty,
		IncludeTags:   req.IncludeTags,
		BackupEnabled: req.Backup,
		BackupRef:     backupRef,
		Commands:      []string{rewriteCmd},
		AffectedFiles: affectedFiles,
	}

	// 6. Print plan; exit here if dry-run (zero side effects)
	if !req.Silent {
		plan.PrintCompact(p)
	}
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

	// 8. Create backup ref (after confirmation, before rewrite)
	if req.Backup {
		output.Print("creating backup ref: %s", backupRef)
		if err := gitutil.CreateBackupRef(root, backupRef); err != nil {
			return err
		}
	}

	// 9. Run pure-Go rewriter.
	if err := filterrepo.RunReplace(root, req.From, req.To, req.IncludeTags, req.Silent); err != nil {
		return err
	}

	// 10. Strict verification
	if err := verify.VerifyReplace(root, req.From, req.IncludeTags); err != nil {
		return err
	}

	// 11. Success
	output.Print("✓ rewrite complete and verified.")
	if req.Backup {
		output.Print("backup ref: %s", backupRef)
		output.Print("  to restore: git checkout -b restore-branch %s", backupRef)
	}

	// 12. Warnings (shown last so they're the final thing the user sees)
	output.Warn("All commit hashes will change — collaborators must re-clone or hard-reset after force-push.")
	output.Warn("If this is a secret, rotate it now — rewriting history does not invalidate it.")
	return nil
}

// confirm prompts the user for a yes/no answer.
func confirm() error {
	fmt.Print("This will permanently rewrite all branch history. Continue? [y/N] ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		ans := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if ans == "y" || ans == "yes" {
			return nil
		}
	}
	return &gitutil.ExecError{
		Code:    exitcodes.UserDeclined,
		Message: "user declined confirmation",
	}
}
