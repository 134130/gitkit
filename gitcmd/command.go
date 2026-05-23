package gitcmd

import (
	"io"
	"slices"
	"strings"
)

const (
	ProgramGit = "git"
	ProgramGH  = "gh"
)

type Command struct {
	Program string
	Args    []string
	Dir     string
	Env     []string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func NewCommand(program string, args ...string) Command {
	return Command{
		Program: program,
		Args:    slices.Clone(args),
	}
}

func GitCommand(args ...string) Command {
	return NewCommand(ProgramGit, args...)
}

func GHCommand(args ...string) Command {
	return NewCommand(ProgramGH, args...)
}

func (c Command) InDir(dir string) Command {
	c = c.clone()
	c.Dir = dir
	return c
}

func (c Command) WithEnv(env ...string) Command {
	c = c.clone()
	c.Env = append(c.Env, env...)
	return c
}

func (c Command) WithStdin(stdin io.Reader) Command {
	c = c.clone()
	c.Stdin = stdin
	return c
}

func (c Command) WithStdout(stdout io.Writer) Command {
	c = c.clone()
	c.Stdout = stdout
	return c
}

func (c Command) WithStderr(stderr io.Writer) Command {
	c = c.clone()
	c.Stderr = stderr
	return c
}

func (c Command) Stream(stdout, stderr io.Writer) Command {
	c = c.clone()
	c.Stdout = stdout
	c.Stderr = stderr
	return c
}

func (c Command) String() string {
	if c.Program == "" {
		return strings.Join(c.Args, " ")
	}
	if len(c.Args) == 0 {
		return c.Program
	}
	return c.Program + " " + strings.Join(c.Args, " ")
}

func (c Command) clone() Command {
	c.Args = slices.Clone(c.Args)
	c.Env = slices.Clone(c.Env)
	return c
}
