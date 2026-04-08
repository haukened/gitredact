package plan

import (
	"fmt"
	"os"
	"strings"
)

type AffectedFile struct {
	Path              string
	FirstMatchCommit  string
	FirstMatchSummary string
}

type Plan struct {
	RepoRoot      string
	Operation     string
	Params        map[string]string
	IsDirty       bool
	IncludeTags   bool
	BackupEnabled bool
	BackupRef     string
	Commands      []string
	AffectedFiles []AffectedFile
}

// maskSecret returns a display-safe representation of s: the character count
// and the last up to 4 characters, e.g. `<32 chars, ends "3a9f">`.
func maskSecret(s string) string {
	n := len(s)
	tail := s
	if n > 4 {
		tail = s[n-4:]
	}
	return fmt.Sprintf(`<%d chars, ends "%s">`, n, tail)
}

// PrintCompact writes a compact summary of the plan to stdout.
// Only non-default/active options are shown.
func PrintCompact(p Plan) {
	fmt.Fprintf(os.Stdout, "repo:    %s\n", p.RepoRoot)

	switch p.Operation {
	case "replace":
		from := p.Params["from"]
		to := p.Params["to"]
		fmt.Fprintf(os.Stdout, "replace: %s → %q\n", maskSecret(from), to)
	case "delete-path":
		fmt.Fprintf(os.Stdout, "remove:  %s\n", p.Params["path"])
	default:
		for k, v := range p.Params {
			fmt.Fprintf(os.Stdout, "  %-12s %s\n", k+":", v)
		}
	}

	if p.IncludeTags {
		fmt.Fprintf(os.Stdout, "include-tags: true\n")
	}
	if p.IsDirty {
		fmt.Fprintf(os.Stdout, "dirty:   allowed\n")
	}
	if p.BackupEnabled {
		fmt.Fprintf(os.Stdout, "backup:  %s\n", p.BackupRef)
	}
	if len(p.AffectedFiles) > 0 {
		fmt.Fprintf(os.Stdout, "affected files: %d\n", len(p.AffectedFiles))
		for _, f := range p.AffectedFiles {
			commit := f.FirstMatchCommit
			if len(commit) > 8 {
				commit = commit[:8]
			}
			summary := strings.Split(strings.TrimSpace(f.FirstMatchSummary), "\n")[0]
			fmt.Fprintf(os.Stdout, "  - %s\n", f.Path)
			fmt.Fprintf(os.Stdout, "    first match: %s %s\n", commit, summary)
		}
	}
	fmt.Fprintln(os.Stdout)
}
