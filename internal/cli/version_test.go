package cli

import (
	"testing"
)

func TestNewVersionCommand_Name(t *testing.T) {
	cmd := NewVersionCommand()
	if cmd.Name != "version" {
		t.Errorf("NewVersionCommand().Name = %q, want %q", cmd.Name, "version")
	}
}

func TestNewVersionCommand_Usage(t *testing.T) {
	cmd := NewVersionCommand()
	if cmd.Usage == "" {
		t.Error("NewVersionCommand().Usage is empty")
	}
}
