package gitcmd

import (
	"bytes"
	"context"
	"reflect"
	"testing"
)

type fakeRunner struct {
	command Command
	result  Result
	err     error
}

func (r *fakeRunner) Run(_ context.Context, cmd Command) (Result, error) {
	r.command = cmd
	return r.result, r.err
}

func (r *fakeRunner) Start(_ context.Context, cmd Command) (Process, error) {
	r.command = cmd
	return nil, r.err
}

func TestClientBuildsCommand(t *testing.T) {
	r := &fakeRunner{}
	stdin := bytes.NewBufferString("input")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	_, _ = Git(r).
		InDir("/repo").
		WithEnv("A=B").
		WithStdin(stdin).
		Stream(stdout, stderr).
		Run(context.Background(), "status", "--porcelain")

	if r.command.Program != ProgramGit {
		t.Fatalf("Program = %q, want %q", r.command.Program, ProgramGit)
	}
	if !reflect.DeepEqual(r.command.Args, []string{"status", "--porcelain"}) {
		t.Fatalf("Args = %#v", r.command.Args)
	}
	if r.command.Dir != "/repo" {
		t.Fatalf("Dir = %q", r.command.Dir)
	}
	if !reflect.DeepEqual(r.command.Env, []string{"A=B"}) {
		t.Fatalf("Env = %#v", r.command.Env)
	}
	if r.command.Stdin != stdin || r.command.Stdout != stdout || r.command.Stderr != stderr {
		t.Fatalf("stdio writers were not propagated")
	}
}
