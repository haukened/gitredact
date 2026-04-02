package output

import (
	"fmt"
	"os"
)

var verbose bool

// SetVerbose enables or disables verbose output.
func SetVerbose(v bool) {
	verbose = v
}

// Print writes a line to stdout.
func Print(format string, args ...any) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// Verbose writes a line to stdout only when verbose mode is on.
func Verbose(format string, args ...any) {
	if verbose {
		fmt.Fprintf(os.Stdout, format+"\n", args...)
	}
}

// Warn writes a prominent warning line to stdout.
func Warn(format string, args ...any) {
	fmt.Fprintf(os.Stdout, "WARNING: "+format+"\n", args...)
}

// Section prints a blank-line-separated section header.
func Section(title string) {
	fmt.Fprintf(os.Stdout, "\n=== %s ===\n", title)
}
