package gitrepo

import (
	"context"
	"reflect"
	"testing"

	"github.com/134130/gitkit/gitcmd"
)

type fakeRunner struct {
	command gitcmd.Command
	result  gitcmd.Result
	err     error
}

func (r *fakeRunner) Run(_ context.Context, cmd gitcmd.Command) (gitcmd.Result, error) {
	r.command = cmd
	return r.result, r.err
}

func (r *fakeRunner) Start(_ context.Context, cmd gitcmd.Command) (gitcmd.Process, error) {
	r.command = cmd
	return nil, r.err
}

func TestClientDefaultBranch(t *testing.T) {
	runner := &fakeRunner{result: gitcmd.Result{Stdout: []byte(`
* remote origin
  Fetch URL: git@github.com:owner/repo.git
  Push  URL: git@github.com:owner/repo.git
  HEAD branch: main
`)}}

	branch, err := New(runner).DefaultBranch(context.Background(), "origin")
	if err != nil {
		t.Fatalf("DefaultBranch returned error: %v", err)
	}

	if branch != "main" {
		t.Fatalf("branch mismatch: want %q, got %q", "main", branch)
	}
	if runner.command.Program != gitcmd.ProgramGit {
		t.Fatalf("program mismatch: %q", runner.command.Program)
	}
	if !reflect.DeepEqual(runner.command.Args, []string{"remote", "show", "origin"}) {
		t.Fatalf("args mismatch: %#v", runner.command.Args)
	}
}

func TestClientDefaultBranchReturnsErrorWhenMissing(t *testing.T) {
	runner := &fakeRunner{result: gitcmd.Result{Stdout: []byte("no default here")}}

	_, err := New(runner).DefaultBranch(context.Background(), "upstream")
	if err == nil {
		t.Fatalf("expected error")
	}
}
