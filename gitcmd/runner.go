package gitcmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"slices"

	"github.com/cli/safeexec"
)

type Runner interface {
	Run(ctx context.Context, cmd Command) (Result, error)
	Start(ctx context.Context, cmd Command) (Process, error)
}

type Process interface {
	Wait() (Result, error)
}

type DefaultRunner struct {
	GitPath string
	GHPath  string
	Env     []string
}

func NewRunner() *DefaultRunner {
	return &DefaultRunner{}
}

func (r *DefaultRunner) Run(ctx context.Context, cmd Command) (Result, error) {
	p, err := r.Start(ctx, cmd)
	if err != nil {
		return Result{Command: cmd}, err
	}
	return p.Wait()
}

func (r *DefaultRunner) Start(ctx context.Context, command Command) (Process, error) {
	exe, err := r.path(command.Program)
	if err != nil {
		return nil, err
	}

	execCmd := exec.CommandContext(ctx, exe, command.Args...)
	execCmd.Dir = command.Dir
	execCmd.Stdin = command.Stdin
	if len(r.Env) > 0 || len(command.Env) > 0 {
		execCmd.Env = append(os.Environ(), r.Env...)
		execCmd.Env = append(execCmd.Env, command.Env...)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	execCmd.Stdout = stdout
	execCmd.Stderr = stderr
	if command.Stdout != nil {
		execCmd.Stdout = io.MultiWriter(stdout, command.Stdout)
	}
	if command.Stderr != nil {
		execCmd.Stderr = io.MultiWriter(stderr, command.Stderr)
	}

	if err := execCmd.Start(); err != nil {
		return nil, err
	}

	return &execProcess{
		cmd:     execCmd,
		command: command.clone(),
		stdout:  stdout,
		stderr:  stderr,
	}, nil
}

func (r *DefaultRunner) path(program string) (string, error) {
	switch program {
	case ProgramGit:
		if r.GitPath != "" {
			return r.GitPath, nil
		}
		path, err := safeexec.LookPath(ProgramGit)
		if err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				return "", &NotInstalledError{Program: ProgramGit, Err: err}
			}
			return "", err
		}
		return path, nil
	case ProgramGH:
		if r.GHPath != "" {
			return r.GHPath, nil
		}
		if ghPath := os.Getenv("GH_PATH"); ghPath != "" {
			return ghPath, nil
		}
		path, err := safeexec.LookPath(ProgramGH)
		if err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				return "", &NotInstalledError{Program: ProgramGH, Err: err}
			}
			return "", err
		}
		return path, nil
	default:
		return "", &UnsupportedProgramError{Program: program}
	}
}

type execProcess struct {
	cmd     *exec.Cmd
	command Command
	stdout  *bytes.Buffer
	stderr  *bytes.Buffer
}

func (p *execProcess) Wait() (Result, error) {
	err := p.cmd.Wait()
	result := Result{
		Command: p.command.clone(),
		Stdout:  slices.Clone(p.stdout.Bytes()),
		Stderr:  slices.Clone(p.stderr.Bytes()),
	}
	if err == nil {
		return result, nil
	}

	result.ExitCode = 1
	if exitError, ok := errors.AsType[*exec.ExitError](err); ok {
		result.ExitCode = exitError.ExitCode()
	}
	return result, &ExitError{Result: result, Err: err}
}
