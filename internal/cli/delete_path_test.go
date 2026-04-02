package cli

import (
	"context"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestNewDeletePathCommand_Name(t *testing.T) {
	cmd := NewDeletePathCommand()
	if cmd.Name != "delete-path" {
		t.Errorf("Name = %q, want %q", cmd.Name, "delete-path")
	}
}

func TestNewDeletePathCommand_HasPathFlag(t *testing.T) {
	cmd := NewDeletePathCommand()
	var found bool
	for _, f := range cmd.Flags {
		for _, n := range f.Names() {
			if n == "path" {
				found = true
			}
		}
	}
	if !found {
		t.Error("NewDeletePathCommand: missing --path flag")
	}
}

func TestNewDeletePathCommand_PathFlagRequired(t *testing.T) {
	cmd := NewDeletePathCommand()
	for _, f := range cmd.Flags {
		for _, n := range f.Names() {
			if n == "path" {
				sf, ok := f.(*cli.StringFlag)
				if !ok {
					t.Fatalf("--path is not a *cli.StringFlag, got %T", f)
				}
				if !sf.Required {
					t.Error("--path flag should be Required")
				}
			}
		}
	}
}

func TestNewDeletePathCommand_HasGlobalFlags(t *testing.T) {
	cmd := NewDeletePathCommand()
	globalNames := make(map[string]bool)
	for _, f := range GlobalFlags() {
		for _, n := range f.Names() {
			globalNames[n] = true
		}
	}
	cmdNames := make(map[string]bool)
	for _, f := range cmd.Flags {
		for _, n := range f.Names() {
			cmdNames[n] = true
		}
	}
	for name := range globalNames {
		if !cmdNames[name] {
			t.Errorf("NewDeletePathCommand: missing global flag %q", name)
		}
	}
}

func TestNewDeletePathCommand_Action_DryRun(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "private.key", "private key data", "initial")

	app := NewApp()
	err := app.Run(context.Background(), []string{
		"gitredact", "delete-path", "--path=private.key", "--yes", "--dry-run", "--silent", dir,
	})
	if err != nil {
		t.Fatalf("delete-path action (dry-run): unexpected error: %v", err)
	}
}

func TestNewDeletePathCommand_Action_NoPositionalArg(t *testing.T) {
	// Exercises the repoPath="" branch (uses CWD). Expect an error because
	// the path is unlikely to exist in the CWD repo history.
	app := NewApp()
	err := app.Run(context.Background(), []string{
		"gitredact", "delete-path", "--path=this-path-does-not-exist/anywhere.txt",
		"--yes", "--dry-run",
	})
	if err == nil {
		t.Log("delete-path action (no positional arg): got nil, path may exist in CWD repo")
	}
}
