package ghpr

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/134130/gitkit/ghcli"
	"github.com/134130/gitkit/gitrepo"
)

type MergeStrategy string

const (
	MergeStrategyRebase      MergeStrategy = "rebase"
	MergeStrategySquash      MergeStrategy = "squash"
	MergeStrategyMergeCommit MergeStrategy = "merge_commit"
)

type Client struct {
	gh   ghcli.Client
	repo gitrepo.Client
}

func New(gh ghcli.Client, repo gitrepo.Client) Client {
	return Client{gh: gh, repo: repo}
}

func (c Client) MergeStrategy(ctx context.Context, prNumber int, mergeCommitSHA string) (MergeStrategy, error) {
	if mergeCommitSHA == "" {
		return "", fmt.Errorf("failed to get merge commit SHA for PR #%d: PR not merged", prNumber)
	}

	nameWithOwner, err := c.gh.Output(ctx, "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	if err != nil {
		return "", fmt.Errorf("failed to get repository name with owner: %w", err)
	}

	hostname, err := c.hostname(ctx)
	if err != nil {
		return "", err
	}

	commitEndpoint := fmt.Sprintf("repos/%s/commits/%s", nameWithOwner, mergeCommitSHA)
	parentCountStr, err := c.gh.API(ctx, commitEndpoint, ghcli.APIOptions{
		Hostname: hostname,
		JQ:       ".parents | length",
	})
	if err != nil {
		return "", fmt.Errorf("failed to get parent count for merge commit %s: %w", mergeCommitSHA, err)
	}

	parentCount, err := strconv.Atoi(strings.TrimSpace(parentCountStr))
	if err != nil {
		return "", fmt.Errorf("failed to parse parent count %q: %w", parentCountStr, err)
	}
	if parentCount > 1 {
		return MergeStrategyMergeCommit, nil
	}

	prevCommitSHA, err := c.gh.API(ctx, commitEndpoint, ghcli.APIOptions{
		Hostname: hostname,
		JQ:       ".parents[0].sha",
	})
	if err != nil {
		return "", fmt.Errorf("failed to get previous commit SHA for merge commit %s: %w", mergeCommitSHA, err)
	}

	prNumbersStr, err := c.gh.API(ctx, fmt.Sprintf("repos/%s/commits/%s/pulls", nameWithOwner, prevCommitSHA), ghcli.APIOptions{
		Hostname: hostname,
		Headers:  map[string]string{"Accept": "application/vnd.github+json"},
		JQ:       ".[].number",
	})
	if err != nil {
		return "", fmt.Errorf("failed to get related PR numbers for commit %s: %w", prevCommitSHA, err)
	}

	targetPR := strconv.Itoa(prNumber)
	for line := range strings.SplitSeq(prNumbersStr, "\n") {
		if strings.TrimSpace(line) == targetPR {
			return MergeStrategyRebase, nil
		}
	}
	return MergeStrategySquash, nil
}

func (c Client) hostname(ctx context.Context) (string, error) {
	remoteURL, err := c.repo.RemoteURL(ctx, "origin")
	if err != nil {
		return "", fmt.Errorf("failed to get origin remote URL: %w", err)
	}
	hostname, err := HostFromRemoteURL(remoteURL)
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub hostname: %w", err)
	}
	return hostname, nil
}

func HostFromRemoteURL(remoteURL string) (string, error) {
	if strings.Contains(remoteURL, "://") {
		u, err := url.Parse(remoteURL)
		if err != nil {
			return "", fmt.Errorf("cannot parse remote URL %q: %w", remoteURL, err)
		}
		if u.Host == "" {
			return "", fmt.Errorf("cannot parse remote URL %q: missing host", remoteURL)
		}
		return u.Host, nil
	}

	if _, after, ok := strings.Cut(remoteURL, "@"); ok {
		hostAndPath := after
		if before, _, ok := strings.Cut(hostAndPath, ":"); ok {
			return before, nil
		}
	}

	return "", fmt.Errorf("cannot parse remote URL %q: unsupported format", remoteURL)
}
