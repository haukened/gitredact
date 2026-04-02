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

func Print(p Plan) {
	fmt.Fprintf(os.Stdout, "\n--- gitredact plan ---\n")
	fmt.Fprintf(os.Stdout, "repo:         %s\n", p.RepoRoot)
	fmt.Fprintf(os.Stdout, "operation:    %s\n", p.Operation)
	for k, v := range p.Params {
		fmt.Fprintf(os.Stdout, "  %-12s %s\n", k+":", v)
	}
	fmt.Fprintf(os.Stdout, "dirty:        %v\n", p.IsDirty)
	fmt.Fprintf(os.Stdout, "include-tags: %v\n", p.IncludeTags)
	if p.BackupEnabled {
		fmt.Fprintf(os.Stdout, "backup:       enabled (%s)\n", p.BackupRef)
	} else {
		fmt.Fprintf(os.Stdout, "backup:       disabled (no backup ref will be created)\n")
	}
	fmt.Fprintf(os.Stdout, "commands:\n")
	for _, cmd := range p.Commands {
		fmt.Fprintf(os.Stdout, "  %s\n", cmd)
	}
	fmt.Fprintf(os.Stdout, "---\n\n")
}
