package gitrepo

import (
	"context"
	"errors"
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

type response struct {
	result gitcmd.Result
	err    error
}

type sequenceRunner struct {
	responses []response
	commands  []gitcmd.Command
}

func (r *sequenceRunner) Run(_ context.Context, cmd gitcmd.Command) (gitcmd.Result, error) {
	r.commands = append(r.commands, cmd)
	if len(r.responses) == 0 {
		return gitcmd.Result{Command: cmd}, nil
	}
	res := r.responses[0]
	r.responses = r.responses[1:]
	res.result.Command = cmd
	return res.result, res.err
}

func (r *sequenceRunner) Start(_ context.Context, cmd gitcmd.Command) (gitcmd.Process, error) {
	r.commands = append(r.commands, cmd)
	return nil, nil
}

func TestClientDefaultBranch(t *testing.T) {
	runner := &fakeRunner{result: gitcmd.Result{Stdout: []byte("origin/main\n")}}

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
	if !reflect.DeepEqual(runner.command.Args, []string{"symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD"}) {
		t.Fatalf("args mismatch: %#v", runner.command.Args)
	}
}

func TestClientDefaultBranchFallsBackToRemoteShow(t *testing.T) {
	runner := &sequenceRunner{responses: []response{
		{err: errors.New("missing local remote head")},
		{result: gitcmd.Result{Stdout: []byte(`
* remote origin
  Fetch URL: git@github.com:owner/repo.git
  Push  URL: git@github.com:owner/repo.git
  HEAD branch: main
`)}},
	}}

	branch, err := New(runner).DefaultBranch(context.Background(), "origin")
	if err != nil {
		t.Fatalf("DefaultBranch returned error: %v", err)
	}

	if branch != "main" {
		t.Fatalf("branch mismatch: want %q, got %q", "main", branch)
	}
	wantCommands := [][]string{
		{"symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD"},
		{"remote", "show", "origin"},
	}
	if len(runner.commands) != len(wantCommands) {
		t.Fatalf("command count mismatch: %#v", runner.commands)
	}
	for i, want := range wantCommands {
		if !reflect.DeepEqual(runner.commands[i].Args, want) {
			t.Fatalf("args mismatch at %d\nwant: %#v\n got: %#v", i, want, runner.commands[i].Args)
		}
	}
}

func TestClientDefaultBranchReturnsErrorWhenMissing(t *testing.T) {
	runner := &fakeRunner{result: gitcmd.Result{Stdout: []byte("no default here")}}

	_, err := New(runner).DefaultBranch(context.Background(), "upstream")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestClientCurrentBranch(t *testing.T) {
	runner := &fakeRunner{result: gitcmd.Result{Stdout: []byte("feature\n")}}

	branch, err := New(runner).CurrentBranch(context.Background())
	if err != nil {
		t.Fatalf("CurrentBranch returned error: %v", err)
	}

	if branch != "feature" {
		t.Fatalf("branch mismatch: want %q, got %q", "feature", branch)
	}
	if !reflect.DeepEqual(runner.command.Args, []string{"branch", "--show-current"}) {
		t.Fatalf("args mismatch: %#v", runner.command.Args)
	}
}

func TestClientCurrentBranchReturnsErrorWhenDetached(t *testing.T) {
	runner := &fakeRunner{result: gitcmd.Result{Stdout: []byte("\n")}}

	_, err := New(runner).CurrentBranch(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestClientStashPushReturnsNewestRef(t *testing.T) {
	runner := &sequenceRunner{responses: []response{
		{result: gitcmd.Result{Stdout: []byte("Saved working directory and index state\n")}},
		{result: gitcmd.Result{Stdout: []byte("stash@{0}\n")}},
	}}

	ref, err := New(runner).StashPush(context.Background(), "save worktree", true)
	if err != nil {
		t.Fatalf("StashPush returned error: %v", err)
	}

	if ref != "stash@{0}" {
		t.Fatalf("ref mismatch: want %q, got %q", "stash@{0}", ref)
	}
	wantCommands := [][]string{
		{"stash", "push", "-u", "-m", "save worktree"},
		{"stash", "list", "--format=%gd", "-n", "1"},
	}
	for i, want := range wantCommands {
		if !reflect.DeepEqual(runner.commands[i].Args, want) {
			t.Fatalf("command %d args mismatch: want %#v, got %#v", i, want, runner.commands[i].Args)
		}
	}
}

func TestClientStashPushReturnsEmptyRefWhenNothingChanged(t *testing.T) {
	runner := &sequenceRunner{responses: []response{
		{result: gitcmd.Result{Stdout: []byte("No local changes to save\n")}},
	}}

	ref, err := New(runner).StashPush(context.Background(), "save worktree", true)
	if err != nil {
		t.Fatalf("StashPush returned error: %v", err)
	}

	if ref != "" {
		t.Fatalf("ref mismatch: want empty, got %q", ref)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("command count mismatch: want 1, got %d", len(runner.commands))
	}
}

func TestClientAheadBehind(t *testing.T) {
	runner := &fakeRunner{result: gitcmd.Result{Stdout: []byte("2\t3\n")}}

	result, err := New(runner).AheadBehind(context.Background(), "origin/main", "HEAD")
	if err != nil {
		t.Fatalf("AheadBehind returned error: %v", err)
	}

	if result.Ahead != 3 || result.Behind != 2 {
		t.Fatalf("ahead/behind mismatch: %#v", result)
	}
	if !reflect.DeepEqual(runner.command.Args, []string{"rev-list", "--left-right", "--count", "origin/main...HEAD"}) {
		t.Fatalf("args mismatch: %#v", runner.command.Args)
	}
}

func TestClientRebaseOmitsEmptyBranch(t *testing.T) {
	runner := &fakeRunner{}

	err := New(runner).Rebase(context.Background(), RebaseOptions{
		Onto:     "origin/main",
		Upstream: "old-base",
	})
	if err != nil {
		t.Fatalf("Rebase returned error: %v", err)
	}

	if !reflect.DeepEqual(runner.command.Args, []string{"rebase", "--onto", "origin/main", "old-base"}) {
		t.Fatalf("args mismatch: %#v", runner.command.Args)
	}
}

func TestClientPushForceWithLeaseRefspec(t *testing.T) {
	runner := &fakeRunner{}

	err := New(runner).PushForceWithLeaseRefspec(context.Background(), "origin", "HEAD:feature")
	if err != nil {
		t.Fatalf("PushForceWithLeaseRefspec returned error: %v", err)
	}

	if !reflect.DeepEqual(runner.command.Args, []string{"push", "--force-with-lease", "origin", "HEAD:feature"}) {
		t.Fatalf("args mismatch: %#v", runner.command.Args)
	}
}
