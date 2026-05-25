package gitrepo

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
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

func (c Client) CurrentBranch(ctx context.Context) (string, error) {
	branch, err := c.git.Output(ctx, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	if branch == "" {
		return "", fmt.Errorf("current HEAD is detached")
	}
	return branch, nil
}

func (c Client) DefaultBranch(ctx context.Context, remote string) (string, error) {
	branch, err := c.defaultBranchFromRemoteHead(ctx, remote)
	if err == nil {
		return branch, nil
	}
	return c.defaultBranchFromRemoteShow(ctx, remote)
}

func (c Client) defaultBranchFromRemoteHead(ctx context.Context, remote string) (string, error) {
	out, err := c.git.Output(ctx, "symbolic-ref", "--quiet", "--short", "refs/remotes/"+remote+"/HEAD")
	if err != nil {
		return "", err
	}
	if branch, ok := defaultBranchFromSymbolicRef(remote, out); ok {
		return branch, nil
	}
	return "", fmt.Errorf("could not determine default branch from refs/remotes/%s/HEAD", remote)
}

func (c Client) defaultBranchFromRemoteShow(ctx context.Context, remote string) (string, error) {
	out, err := c.git.Output(ctx, "remote", "show", remote)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		if before, after, ok := strings.Cut(line, ":"); ok && strings.Contains(before, "HEAD branch") {
			branch := strings.TrimSpace(after)
			if branch != "" {
				return branch, nil
			}
		}
	}
	return "", fmt.Errorf("could not determine default branch for remote %s", remote)
}

func defaultBranchFromSymbolicRef(remote, ref string) (string, bool) {
	ref = strings.TrimSpace(ref)
	branch, ok := strings.CutPrefix(ref, remote+"/")
	return branch, ok && branch != ""
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

func (c Client) Switch(ctx context.Context, branch string) error {
	_, err := c.git.Run(ctx, "switch", branch)
	return err
}

func (c Client) SwitchCreateOrReset(ctx context.Context, branch, startPoint string) error {
	_, err := c.git.Run(ctx, "switch", "-C", branch, startPoint)
	return err
}

func (c Client) PullRebase(ctx context.Context, remote, branch string) error {
	_, err := c.git.Run(ctx, "pull", "--rebase", remote, branch)
	return err
}

func (c Client) StashPush(ctx context.Context, message string, includeUntracked bool) (string, error) {
	args := []string{"stash", "push"}
	if includeUntracked {
		args = append(args, "-u")
	}
	if message != "" {
		args = append(args, "-m", message)
	}
	result, err := c.git.Run(ctx, args...)
	if err != nil {
		return "", err
	}
	if strings.Contains(result.StdoutString(), "No local changes to save") {
		return "", nil
	}
	ref, err := c.git.Output(ctx, "stash", "list", "--format=%gd", "-n", "1")
	if err != nil {
		return "", err
	}
	if ref == "" {
		return "", fmt.Errorf("stash push succeeded but no stash ref was found")
	}
	return ref, nil
}

func (c Client) StashApply(ctx context.Context, ref string) error {
	_, err := c.git.Run(ctx, "stash", "apply", ref)
	return err
}

func (c Client) StashDrop(ctx context.Context, ref string) error {
	_, err := c.git.Run(ctx, "stash", "drop", ref)
	return err
}

type AheadBehind struct {
	Ahead  int
	Behind int
}

func (c Client) AheadBehind(ctx context.Context, left, right string) (AheadBehind, error) {
	out, err := c.git.Output(ctx, "rev-list", "--left-right", "--count", left+"..."+right)
	if err != nil {
		return AheadBehind{}, err
	}
	fields := strings.Fields(out)
	if len(fields) != 2 {
		return AheadBehind{}, fmt.Errorf("unexpected ahead/behind output %q", out)
	}
	leftCount, err := strconv.Atoi(fields[0])
	if err != nil {
		return AheadBehind{}, fmt.Errorf("parse left count %q: %w", fields[0], err)
	}
	rightCount, err := strconv.Atoi(fields[1])
	if err != nil {
		return AheadBehind{}, fmt.Errorf("parse right count %q: %w", fields[1], err)
	}
	return AheadBehind{Ahead: rightCount, Behind: leftCount}, nil
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

func (c Client) PushForceWithLeaseRefspec(ctx context.Context, remote, refspec string) error {
	_, err := c.git.Run(ctx, "push", "--force-with-lease", remote, refspec)
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
		args = append(args, "--onto", opts.Onto, opts.Upstream)
		if opts.Branch != "" {
			args = append(args, opts.Branch)
		}
	} else {
		args = append(args, opts.Onto)
		if opts.Branch != "" {
			args = append(args, opts.Branch)
		}
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
