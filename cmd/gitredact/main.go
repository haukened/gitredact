package main

import (
	"context"
	"fmt"
	"os"

	gitredactcli "gitredact/internal/cli"
	"gitredact/internal/exitcodes"
	"gitredact/internal/gitutil"
)

var Version = "dev"

func main() {
	app := gitredactcli.NewApp(Version)
	if err := app.Run(context.Background(), os.Args); err != nil {
		code := exitcodes.InvalidUsage
		if ee, ok := err.(*gitutil.ExecError); ok {
			code = ee.Code
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(code)
	}
}
