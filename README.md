![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/haukened/gitredact/gosec.yml?label=Security%20Scan)
![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/haukened/gitredact/ci.yml)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/haukened/gitredact)

# gitredact

A Go CLI that rewrites Git history to replace a literal string or delete an exact path across all reachable commits. Implemented in pure Go — no Python, no external tools required.

>[!WARNING]
> This tool is destructive by nature. Always run with `--dry-run` first to see what would be changed, and use `--backup` to create a safety ref before rewriting.

You will need to push changes after running, and collaborators will need to re-clone or hard-reset their copies. If the replaced string was a secret, **rotate it** — rewriting history does not invalidate credentials already in use.

## Requirements

- Go 1.21+

## Build

```sh
go build -o gitredact ./cmd/gitredact
```

## Usage

### Replace a literal string across history

```sh
# Dry-run (no changes made):
gitredact replace --dry-run --from "secret-token" --to "REDACTED"

# Execute with confirmation prompt:
gitredact replace --from "secret-token" --to "REDACTED"

# Execute without prompt, with backup ref:
gitredact replace --from "secret-token" --to "REDACTED" --yes --backup

# Run on a specific repo:
gitredact replace --from "old-host" --to "new-host" --yes /path/to/repo

# Suppress all output (errors only):
gitredact replace --from "secret-token" --to "REDACTED" --yes --silent
```

### Delete a file path from history

```sh
# Dry-run:
gitredact delete-path --dry-run --path secrets/credentials.txt

# Execute:
gitredact delete-path --path secrets/credentials.txt --yes

# Include tags in rewrite and create a backup:
gitredact delete-path --path secrets/credentials.txt --yes --include-tags --backup

# Suppress all output (errors only):
gitredact delete-path --path secrets/credentials.txt --yes --silent
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Print plan and exit; zero side effects |
| `--yes` | Skip interactive confirmation |
| `--include-tags` | Rewrite tags in addition to branches |
| `--allow-dirty` | Allow running on a dirty worktree |
| `--verbose` | Verbose output |
| `--backup` | Create a backup ref (`refs/gitredact-backup/<timestamp>`) before rewrite (opt-in; not created in dry-run) |
| `--silent` | Suppress all output; only errors are surfaced via exit code |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Invalid usage |
| 3 | Repo validation failure |
| 4 | Dirty worktree refusal |
| 5 | No matches found in preflight |
| 6 | Rewrite execution failure |
| 7 | User declined confirmation |
| 8 | Verification failure |
| 9 | Dependency missing |

## Safety Notes

- History is rewritten locally only. The tool does **not** push anything.
- After a successful run, collaborators must re-clone or hard-reset their copies.
- If the replaced string was a secret, **rotate it** — rewriting history does not invalidate credentials already in use.
- Use `--backup` to save a recovery ref before rewriting.
