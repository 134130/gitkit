package gitrepo

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/134130/gitkit/gitcmd"
)

type Client struct {
	git gitcmd.Client
}

type Option func(*Client)

func New(r gitcmd.Runner, opts ...Option) Client {
	c := Client{git: gitcmd.Git(r)}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

func WithDir(dir string) Option {
	return func(c *Client) {
		c.git = c.git.InDir(dir)
	}
}

func WithStreams(stdout, stderr io.Writer) Option {
	return func(c *Client) {
		c.git = c.git.Stream(stdout, stderr)
	}
}

func (c Client) InDir(dir string) Client {
	c.git = c.git.InDir(dir)
	return c
}

func (c Client) Stream(stdout, stderr io.Writer) Client {
	c.git = c.git.Stream(stdout, stderr)
	return c
}

func (c Client) Root(ctx context.Context) (string, error) {
	return c.git.Output(ctx, "rev-parse", "--show-toplevel")
}

func (c Client) GitDir(ctx context.Context) (string, error) {
	return c.git.Output(ctx, "rev-parse", "--git-dir")
}

func (c Client) RevParse(ctx context.Context, ref string) (string, error) {
	return c.git.Output(ctx, "rev-parse", ref)
}

func (c Client) RemoteURL(ctx context.Context, name string) (string, error) {
	return c.git.Output(ctx, "remote", "get-url", name)
}

func (c Client) MergeBase(ctx context.Context, a, b string) (string, error) {
	return c.git.Output(ctx, "merge-base", a, b)
}

func (c Client) IsAncestor(ctx context.Context, ancestor, descendant string) (bool, error) {
	_, err := c.git.Run(ctx, "merge-base", "--is-ancestor", ancestor, descendant)
	if err == nil {
		return true, nil
	}
	if gitcmd.IsExitCode(err, 1) {
		return false, nil
	}
	return false, err
}

func (c Client) IsDirty(ctx context.Context) (bool, error) {
	out, err := c.git.Output(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

func (c Client) State(ctx context.Context) (State, error) {
	dirty, err := c.IsDirty(ctx)
	if err != nil {
		return State{}, err
	}

	rebasing, err := c.gitPathExists(ctx, "rebase-merge")
	if err != nil {
		return State{}, err
	}
	applyingPatch, err := c.gitPathExists(ctx, "rebase-apply")
	if err != nil {
		return State{}, err
	}
	cherryPicking, err := c.gitPathExists(ctx, "CHERRY_PICK_HEAD")
	if err != nil {
		return State{}, err
	}
	merging, err := c.gitPathExists(ctx, "MERGE_HEAD")
	if err != nil {
		return State{}, err
	}

	return State{
		Dirty:         dirty,
		Rebasing:      rebasing,
		ApplyingPatch: applyingPatch,
		CherryPicking: cherryPicking,
		Merging:       merging,
	}, nil
}

func (c Client) IsRebasing(ctx context.Context) (bool, error) {
	state, err := c.State(ctx)
	return state.Rebasing || state.ApplyingPatch, err
}

func (c Client) Fetch(ctx context.Context, remote string, refspec ...string) error {
	args := []string{"fetch", remote}
	args = append(args, refspec...)
	_, err := c.git.Run(ctx, args...)
	return err
}

func (c Client) FetchRecurseSubmodules(ctx context.Context, remote, refspec string) error {
	args := []string{"fetch", "--recurse-submodules", remote}
	if refspec != "" {
		args = append(args, refspec)
	}
	_, err := c.git.Run(ctx, args...)
	return err
}

func (c Client) Clone(ctx context.Context, remoteURL, targetDir string) error {
	_, err := c.git.Run(ctx, "clone", remoteURL, targetDir)
	return err
}

func (c Client) WorktreeAdd(ctx context.Context, path, startPoint string, args ...string) error {
	allArgs := []string{"worktree", "add"}
	allArgs = append(allArgs, args...)
	allArgs = append(allArgs, path, startPoint)
	_, err := c.git.Run(ctx, allArgs...)
	return err
}

func (c Client) WorktreeRemove(ctx context.Context, path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	_, err := c.git.Run(ctx, args...)
	return err
}

func (c Client) CheckoutNewBranch(ctx context.Context, branch, remoteStartPoint string) error {
	_, err := c.git.Run(ctx, "switch", "-c", branch, "--track", remoteStartPoint)
	return err
}

func (c Client) PushSetUpstream(ctx context.Context, remote, ref string) error {
	_, err := c.git.Run(ctx, "push", "--set-upstream", remote, ref)
	return err
}

func (c Client) PushForceWithLease(ctx context.Context, remote, branch string) error {
	_, err := c.git.Run(ctx, "push", "--force-with-lease", remote, branch)
	return err
}

type RebaseOptions struct {
	Onto     string
	Upstream string
	Branch   string
}

func (c Client) Rebase(ctx context.Context, opts RebaseOptions) error {
	args := []string{"rebase"}
	if opts.Upstream != "" {
		args = append(args, "--onto", opts.Onto, opts.Upstream, opts.Branch)
	} else {
		args = append(args, opts.Onto, opts.Branch)
	}

	_, err := c.git.Run(ctx, args...)
	if gitcmd.IsExitCode(err, 1) {
		return ErrRebaseConflict
	}
	return err
}

func (c Client) AbortRebase(ctx context.Context) error {
	_, err := c.git.Run(ctx, "rebase", "--abort")
	return err
}

func (c Client) gitPathExists(ctx context.Context, name string) (bool, error) {
	path, err := c.git.Output(ctx, "rev-parse", "--git-path", name)
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(path) == "" {
		return false, fmt.Errorf("git returned empty path for %s", name)
	}
	if _, err := os.Stat(path); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}
