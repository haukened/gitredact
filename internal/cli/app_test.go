package cli

import (
	"testing"

	"github.com/urfave/cli/v3"
)

func TestGlobalFlags_Count(t *testing.T) {
	flags := GlobalFlags()
	if len(flags) == 0 {
		t.Fatal("GlobalFlags: expected non-empty slice")
	}
}

func TestGlobalFlags_ExpectedNames(t *testing.T) {
	flags := GlobalFlags()
	names := make(map[string]bool, len(flags))
	for _, f := range flags {
		for _, n := range f.Names() {
			names[n] = true
		}
	}

	want := []string{"dry-run", "yes", "include-tags", "allow-dirty", "verbose", "backup", "silent"}
	for _, name := range want {
		if !names[name] {
			t.Errorf("GlobalFlags: missing flag %q", name)
		}
	}
}

func TestNewApp_Name(t *testing.T) {
	app := NewApp()
	if app.Name != "gitredact" {
		t.Errorf("NewApp().Name = %q, want %q", app.Name, "gitredact")
	}
}

func TestNewApp_SubcommandCount(t *testing.T) {
	app := NewApp()
	if len(app.Commands) != 2 {
		t.Errorf("NewApp: expected 2 subcommands, got %d", len(app.Commands))
	}
}

func TestNewApp_SubcommandNames(t *testing.T) {
	app := NewApp()
	names := make(map[string]bool, len(app.Commands))
	for _, cmd := range app.Commands {
		names[cmd.Name] = true
	}
	for _, want := range []string{"replace", "delete-path"} {
		if !names[want] {
			t.Errorf("NewApp: missing subcommand %q", want)
		}
	}
}

func TestGlobalFlags_AllAreBoolFlags(t *testing.T) {
	for _, f := range GlobalFlags() {
		if _, ok := f.(*cli.BoolFlag); !ok {
			t.Errorf("GlobalFlags: flag %v is not a *cli.BoolFlag", f.Names())
		}
	}
}
