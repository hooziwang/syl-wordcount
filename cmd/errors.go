package cmd

import "fmt"

const (
	ExitOK        = 0
	ExitViolation = 1
	ExitArg       = 2
	ExitInput     = 3
	ExitConfig    = 4
	ExitInternal  = 5
)

type ExitError struct {
	Code int
	Msg  string
}

func (e *ExitError) Error() string {
	if e.Msg == "" {
		return fmt.Sprintf("exit code %d", e.Code)
	}
	return e.Msg
}
