package cli

import (
	"github.com/urfave/cli/v3"
)

func GlobalFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "dry-run", Usage: "print plan and exit; perform no mutations"},
		&cli.BoolFlag{Name: "yes", Usage: "skip interactive confirmation"},
		&cli.BoolFlag{Name: "include-tags", Usage: "rewrite tags in addition to branches"},
		&cli.BoolFlag{Name: "allow-dirty", Usage: "allow running on a dirty worktree"},
		&cli.BoolFlag{Name: "verbose", Usage: "verbose output"},
		&cli.BoolFlag{Name: "backup", Usage: "create a backup ref before rewrite (opt-in; skipped in dry-run)"},
	}
}

func NewApp() *cli.Command {
	return &cli.Command{
		Name:  "gitredact",
		Usage: "rewrite Git history to remove sensitive data",
		Commands: []*cli.Command{
			NewReplaceCommand(),
			NewDeletePathCommand(),
		},
	}
}
