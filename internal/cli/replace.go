package cli

import (
	"context"

	"github.com/urfave/cli/v3"

	"gitredact/internal/app"
	"gitredact/internal/output"
)

// NewReplaceCommand returns the "replace" subcommand.
func NewReplaceCommand() *cli.Command {
	return &cli.Command{
		Name:  "replace",
		Usage: "replace a literal string across all reachable history",
		Flags: append(GlobalFlags(), []cli.Flag{
			&cli.StringFlag{
				Name:     "from",
				Usage:    "literal string to find (required)",
				Required: true,
			},
			&cli.StringFlag{
				Name:        "to",
				Usage:       "literal string to substitute",
				DefaultText: "REDACTED",
				Value:       "REDACTED",
			},
		}...),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			output.SetVerbose(cmd.Bool("verbose"))
			repoPath := ""
			if cmd.Args().Len() > 0 {
				repoPath = cmd.Args().First()
			}
			return app.RunReplace(app.ReplaceRequest{
				From:        cmd.String("from"),
				To:          cmd.String("to"),
				RepoPath:    repoPath,
				DryRun:      cmd.Bool("dry-run"),
				Yes:         cmd.Bool("yes"),
				IncludeTags: cmd.Bool("include-tags"),
				AllowDirty:  cmd.Bool("allow-dirty"),
				Backup:      cmd.Bool("backup"),
				Silent:      cmd.Bool("silent"),
			})
		},
	}
}
