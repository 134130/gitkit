package gitcmd

import (
	"context"
	"io"
	"os"
	"strings"
)

type Client struct {
	runner  Runner
	program string
	dir     string
	env     []string
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
}

func Git(r Runner) Client {
	return NewClient(r, ProgramGit)
}

func GH(r Runner) Client {
	return NewClient(r, ProgramGH)
}

func NewClient(r Runner, program string) Client {
	if r == nil {
		r = NewRunner()
	}
	return Client{
		runner:  r,
		program: program,
	}
}

func (c Client) InDir(dir string) Client {
	c.dir = dir
	return c
}

func (c Client) WithEnv(env ...string) Client {
	c.env = append(append([]string{}, c.env...), env...)
	return c
}

func (c Client) WithStdin(stdin io.Reader) Client {
	c.stdin = stdin
	return c
}

func (c Client) Stream(stdout, stderr io.Writer) Client {
	c.stdout = stdout
	c.stderr = stderr
	return c
}

func (c Client) Run(ctx context.Context, args ...string) (Result, error) {
	return c.runner.Run(ctx, c.command(args...))
}

func (c Client) Start(ctx context.Context, args ...string) (Process, error) {
	return c.runner.Start(ctx, c.command(args...))
}

func (c Client) OutputBytes(ctx context.Context, args ...string) ([]byte, error) {
	result, err := c.Run(ctx, args...)
	return result.Stdout, err
}

func (c Client) Output(ctx context.Context, args ...string) (string, error) {
	out, err := c.OutputBytes(ctx, args...)
	return strings.TrimSpace(string(out)), err
}

func (c Client) Interactive(ctx context.Context, args ...string) error {
	cmd := c.command(args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_, err := c.runner.Run(ctx, cmd)
	return err
}

func (c Client) command(args ...string) Command {
	cmd := NewCommand(c.program, args...)
	cmd.Dir = c.dir
	cmd.Env = append([]string{}, c.env...)
	cmd.Stdin = c.stdin
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr
	return cmd
}
