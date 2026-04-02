package cli

import (
	"context"

	"github.com/urfave/cli/v3"
)

func NewVersionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "print the version",
		Action: func(_ context.Context, cmd *cli.Command) error {
			cli.ShowVersion(cmd.Root())
			return nil
		},
	}
}
