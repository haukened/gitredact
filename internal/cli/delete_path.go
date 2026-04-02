package cli

import (
	"context"

	"github.com/urfave/cli/v3"

	"gitredact/internal/app"
)

// NewDeletePathCommand returns the "delete-path" subcommand.
func NewDeletePathCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete-path",
		Usage: "delete an exact repo-relative path from all reachable history",
		Flags: append(GlobalFlags(), []cli.Flag{
			&cli.StringFlag{
				Name:     "path",
				Usage:    "repo-relative path to remove (required)",
				Required: true,
			},
		}...),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			repoPath := ""
			if cmd.Args().Len() > 0 {
				repoPath = cmd.Args().First()
			}
			return app.RunDeletePath(app.DeletePathRequest{
				Path:        cmd.String("path"),
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
