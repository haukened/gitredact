package output

import (
	"fmt"
	"os"
)

var verbose bool
var silent bool
var progressActive bool

// SetVerbose enables or disables verbose output.
func SetVerbose(v bool) {
	verbose = v
}

// IsVerbose reports whether verbose mode is enabled.
func IsVerbose() bool {
	return verbose
}

// SetSilent enables or disables silent mode. When silent, all output functions
// are no-ops; only errors (returned as Go error values) are surfaced.
func SetSilent(s bool) {
	silent = s
}

// IsSilent reports whether silent mode is enabled.
func IsSilent() bool {
	return silent
}

// clearProgress terminates an active progress line with a newline so that
// subsequent output starts on a clean line.
func clearProgress() {
	if progressActive {
		fmt.Fprintln(os.Stdout)
		progressActive = false
	}
}

// Print writes a line to stdout, clearing any active progress line first.
func Print(format string, args ...any) {
	if silent {
		return
	}
	clearProgress()
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// Verbose writes a line to stdout only when verbose mode is on.
func Verbose(format string, args ...any) {
	if silent || !verbose {
		return
	}
	clearProgress()
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// Warn writes a prominent warning line to stdout.
func Warn(format string, args ...any) {
	if silent {
		return
	}
	clearProgress()
	fmt.Fprintf(os.Stdout, "WARNING: "+format+"\n", args...)
}

// Section prints a blank-line-separated section header.
func Section(title string) {
	fmt.Fprintf(os.Stdout, "\n=== %s ===\n", title)
}

// Progress overwrites the current terminal line with a progress indicator of
// the form "  label (current/total) ZZ%". Callers must not call this when
// silent mode is active.
func Progress(label string, current, total int) {
	if silent {
		return
	}
	pct := 0
	if total > 0 {
		pct = current * 100 / total
	}
	fmt.Fprintf(os.Stdout, "\r  %s (%d/%d) %d%%", label, current, total, pct)
	progressActive = true
}

// ProgressDone finalises an active progress line by emitting a newline.
// It is a no-op when no progress line is active.
func ProgressDone() {
	clearProgress()
}
