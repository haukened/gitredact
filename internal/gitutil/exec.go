package gitutil

// ExecError is the unified error type returned by all gitutil and app operations.
// Code maps to one of the exitcodes constants.
type ExecError struct {
	Code    int
	Message string
}

func (e *ExecError) Error() string { return e.Message }
