# gitredact v1 Specification

## Goal
Build a Go CLI that rewrites Git history to:
1) replace a literal string across history, or
2) delete an exact repo-relative path across history.

The tool must be conservative, explicit, and hard to misuse. Use existing Git tooling; do not reimplement Git internals.

## CLI

Subcommands:

- gitredact replace --from <string> --to <string> [repo]
- gitredact delete-path --path <repo-relative-path> [repo]

Global flags:

- --dry-run
- --yes
- --include-tags
- --allow-dirty
- --verbose

Positional [repo] defaults to current working directory.

## Requirements

### General
- Language: Go
- CLI library: urfave/cli/v3
- External tools: git, git-filter-repo (must be on PATH)
- Minimal dependencies beyond stdlib and urfave/cli
- No TUI, no Cobra, no Git reimplementation

### Repo handling
- Resolve input path to Git repo root
- Fail if not a Git repository
- Print resolved repo root in plan and execution output

### Safety
- Default to safe behavior
- --dry-run prints plan and performs no mutations
- If not --dry-run and not --yes, require interactive confirmation
- Refuse to run on dirty worktree by default; allow with --allow-dirty
- Print warnings:
  - history will be rewritten (hashes change)
  - collaborators must re-clone or hard reset
  - secrets must be rotated if applicable

### Replace (literal only in v1)
- Flags: --from (required), --to (required)
- Literal matching only (no regex in v1)
- Use git-filter-repo --replace-text via a temp replacements file
- Clean up temp files

### Delete-path
- Flags: --path (required)
- Exact repo-relative path only (no globbing)
- Remove path from all reachable history using git-filter-repo

### Preflight
- For replace: detect if the string appears in reachable history; fail if none found
- For delete-path: detect if the path exists in reachable history; fail if none found
- Use practical git subprocesses (e.g., git grep / rev-list)

### Ref scope
- Rewrite local branches by default
- Include tags only if --include-tags is set
- Do not auto-push
- After success, print exact force-push commands for branches (and tags if included)

### Backup / Recovery
- Create a backup ref before rewrite, e.g. refs/gitredact-backup/<timestamp>
- Print the backup ref name

### Execution
- Build an explicit execution plan (even for non-dry-run)
- Plan includes: repo root, operation, parameters, dirty status, include-tags, backup ref, commands to run, post-push commands
- Show plan in --dry-run and before execution

### Verification
- After rewrite:
  - replace: verify target string no longer exists in reachable history
  - delete-path: verify path no longer exists in reachable history
- Fail with distinct exit code if verification fails

### Dependency checks
- Validate presence of git and git-filter-repo before execution
- Fail clearly if missing

### Output / UX
- Concise, explicit, developer-focused output
- Dry-run shows full plan and commands
- Execution shows progress, verification result, and next steps

### Exit codes
Use named constants and consistent mapping:
- 0 success
- 2 invalid usage
- 3 repo validation failure
- 4 dirty worktree refusal
- 5 no matches found in preflight
- 6 rewrite execution failure
- 7 user declined confirmation
- 8 verification failure
- 9 dependency missing

## Architecture (suggested)

/cmd/gitredact/main.go
/internal/cli/app.go
/internal/cli/replace.go
/internal/cli/delete_path.go
/internal/app/replace.go
/internal/app/delete_path.go
/internal/gitutil/repo.go
/internal/gitutil/status.go
/internal/gitutil/exec.go
/internal/filterrepo/replace.go
/internal/filterrepo/delete_path.go
/internal/verify/verify.go
/internal/plan/plan.go
/internal/output/output.go
/internal/exitcodes/exitcodes.go

Guidelines:
- CLI layer parses flags and builds typed requests
- App layer orchestrates validation, plan, execution, verification
- Subprocess execution isolated in gitutil/filterrepo packages
- Keep code explicit and reviewable; avoid unnecessary abstraction

## Non-goals (v1)
- No regex replacement
- No globbing for paths
- No automatic push
- No partial ref selection beyond include-tags
- No TUI
- No reimplementation of Git history rewriting

## Deliverables
- Complete Go source code
- go.mod
- README.md with build and usage
- Examples for both subcommands
- Notes on required external tools (git, git-filter-repo)

## Acceptance criteria
- Commands run as specified
- Safety behaviors enforced
- Preflight detects absence of targets
- Rewrite executes via git-filter-repo
- Verification passes for successful runs
- Output includes clear next-step push commands
- Code compiles and is organized as specified
