package plan

import (
	"fmt"
	"os"
)

type Plan struct {
	RepoRoot      string
	Operation     string
	Params        map[string]string
	IsDirty       bool
	IncludeTags   bool
	BackupEnabled bool
	BackupRef     string
	Commands      []string
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
	fmt.Fprintln(os.Stdout)
}
