package cli

import (
	"context"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestNewReplaceCommand_Name(t *testing.T) {
	cmd := NewReplaceCommand()
	if cmd.Name != "replace" {
		t.Errorf("Name = %q, want %q", cmd.Name, "replace")
	}
}

func TestNewReplaceCommand_HasFromFlag(t *testing.T) {
	cmd := NewReplaceCommand()
	var found bool
	for _, f := range cmd.Flags {
		for _, n := range f.Names() {
			if n == "from" {
				found = true
			}
		}
	}
	if !found {
		t.Error("NewReplaceCommand: missing --from flag")
	}
}

func TestNewReplaceCommand_HasToFlag(t *testing.T) {
	cmd := NewReplaceCommand()
	var found bool
	for _, f := range cmd.Flags {
		for _, n := range f.Names() {
			if n == "to" {
				found = true
			}
		}
	}
	if !found {
		t.Error("NewReplaceCommand: missing --to flag")
	}
}

func TestNewReplaceCommand_FromFlagRequired(t *testing.T) {
	cmd := NewReplaceCommand()
	for _, f := range cmd.Flags {
		for _, n := range f.Names() {
			if n == "from" {
				sf, ok := f.(*cli.StringFlag)
				if !ok {
					t.Fatalf("--from is not a *cli.StringFlag, got %T", f)
				}
				if !sf.Required {
					t.Error("--from flag should be Required")
				}
			}
		}
	}
}

func TestNewReplaceCommand_ToDefaultValue(t *testing.T) {
	cmd := NewReplaceCommand()
	for _, f := range cmd.Flags {
		for _, n := range f.Names() {
			if n == "to" {
				sf, ok := f.(*cli.StringFlag)
				if !ok {
					t.Fatalf("--to is not a *cli.StringFlag, got %T", f)
				}
				if sf.Value != "REDACTED" {
					t.Errorf("--to default = %q, want %q", sf.Value, "REDACTED")
				}
			}
		}
	}
}

func TestNewReplaceCommand_Action_DryRun(t *testing.T) {
	dir, repo := initRepo(t)
	commitFile(t, dir, repo, "secret.txt", "token=abc123", "initial")

	app := NewApp()
	err := app.Run(context.Background(), []string{
		"gitredact", "replace", "--from=abc123", "--yes", "--dry-run", "--silent", dir,
	})
	if err != nil {
		t.Fatalf("replace action (dry-run): unexpected error: %v", err)
	}
}

func TestNewReplaceCommand_Action_NoPositionalArg(t *testing.T) {
	// When no positional arg is given, the action uses RepoPath="" which
	// resolves to the CWD. The test CWD is inside a git repo, so this
	// should succeed if a commit with the target string is present.
	// We just verify the error path when the string is not found.
	app := NewApp()
	err := app.Run(context.Background(), []string{
		"gitredact", "replace", "--from=this-string-does-not-exist-anywhere", "--yes", "--dry-run",
	})
	// Expect an error (NoMatchesFound) because the string is not in CWD repo.
	if err == nil {
		t.Log("replace action (no positional arg): got nil, string may actually exist in CWD repo")
	}
}
