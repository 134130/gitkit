package gitcmd

import (
	"errors"
	"fmt"
	"strings"
)

type NotInstalledError struct {
	Program string
	Err     error
}

func (e *NotInstalledError) Error() string {
	return fmt.Sprintf("unable to find %s executable in PATH; please install %s before retrying", e.Program, e.Program)
}

func (e *NotInstalledError) Unwrap() error {
	return e.Err
}

type UnsupportedProgramError struct {
	Program string
}

func (e *UnsupportedProgramError) Error() string {
	return fmt.Sprintf("unsupported command program: %s", e.Program)
}

type ExitError struct {
	Result Result
	Err    error
}

func (e *ExitError) Error() string {
	stderr := strings.TrimSpace(string(e.Result.Stderr))
	if stderr != "" {
		return fmt.Sprintf("failed to run %s: %s", e.Result.Command.Program, stderr)
	}
	return fmt.Sprintf("failed to run %s: %v", e.Result.Command.Program, e.Err)
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

func (e *ExitError) ExitCode() int {
	return e.Result.ExitCode
}

func IsExitCode(err error, code int) bool {
	var exitErr interface{ ExitCode() int }
	return errors.As(err, &exitErr) && exitErr.ExitCode() == code
}
