package internal

import (
	"context"

	"github.com/google/go-github/v77/github"
)

type GitHubClient interface {
	PullRequests() PullRequestsService
	Issues() IssuesService
}

type PullRequestsService interface {
	Create(ctx context.Context, owner string, repo string, newPR *github.NewPullRequest) (*github.PullRequest, *github.Response, error)
	List(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
}

type IssuesService interface {
	AddLabelsToIssue(ctx context.Context, owner string, repo string, number int, labels []string) ([]*github.Label, *github.Response, error)
}

type RealGitHubClient struct {
	Client *github.Client
}

func (c *RealGitHubClient) PullRequests() PullRequestsService {
	return c.Client.PullRequests
}

func (c *RealGitHubClient) Issues() IssuesService {
	return c.Client.Issues
}
