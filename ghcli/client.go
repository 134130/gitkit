package ghcli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/134130/gitkit/gitcmd"
)

type Client struct {
	gh gitcmd.Client
}

type Option func(*Client)

func New(r gitcmd.Runner, opts ...Option) Client {
	c := Client{gh: gitcmd.GH(r)}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

func WithDir(dir string) Option {
	return func(c *Client) {
		c.gh = c.gh.InDir(dir)
	}
}

func WithStreams(stdout, stderr io.Writer) Option {
	return func(c *Client) {
		c.gh = c.gh.Stream(stdout, stderr)
	}
}

func (c Client) InDir(dir string) Client {
	c.gh = c.gh.InDir(dir)
	return c
}

func (c Client) Stream(stdout, stderr io.Writer) Client {
	c.gh = c.gh.Stream(stdout, stderr)
	return c
}

func (c Client) Run(ctx context.Context, args ...string) (gitcmd.Result, error) {
	return c.gh.Run(ctx, args...)
}

func (c Client) OutputBytes(ctx context.Context, args ...string) ([]byte, error) {
	return c.gh.OutputBytes(ctx, args...)
}

func (c Client) Output(ctx context.Context, args ...string) (string, error) {
	return c.gh.Output(ctx, args...)
}

func (c Client) Interactive(ctx context.Context, args ...string) error {
	return c.gh.Interactive(ctx, args...)
}

type APIOptions struct {
	Hostname string
	Headers  map[string]string
	JQ       string
}

func (c Client) API(ctx context.Context, endpoint string, opts APIOptions) (string, error) {
	out, err := c.APIBytes(ctx, endpoint, opts)
	return strings.TrimSpace(string(out)), err
}

func (c Client) APIBytes(ctx context.Context, endpoint string, opts APIOptions) ([]byte, error) {
	args := []string{"api"}
	if opts.Hostname != "" {
		args = append(args, "--hostname", opts.Hostname)
	}

	if len(opts.Headers) > 0 {
		keys := make([]string, 0, len(opts.Headers))
		for k := range opts.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			args = append(args, "-H", fmt.Sprintf("%s: %s", k, opts.Headers[k]))
		}
	}

	args = append(args, endpoint)
	if opts.JQ != "" {
		args = append(args, "--jq", opts.JQ)
	}
	return c.gh.OutputBytes(ctx, args...)
}

func (c Client) PRDiff(ctx context.Context, number int) ([]byte, error) {
	return c.gh.OutputBytes(ctx, "pr", "diff", fmt.Sprint(number), "--patch")
}
