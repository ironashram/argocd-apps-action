package internal

import (
	"context"

	"github.com/google/go-github/v77/github"
	"github.com/stretchr/testify/mock"
)

type GitHubClient interface {
	PullRequests() PullRequestsService
	Issues() IssuesService
}

type PullRequestsService interface {
	Create(ctx context.Context, owner string, repo string, newPR *github.NewPullRequest) (*github.PullRequest, *github.Response, error)
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

type MockGithubClient struct {
	PullRequestsService PullRequestsService
	IssuesService       IssuesService
	mock.Mock
}

func (c *MockGithubClient) PullRequests() PullRequestsService {
	return c.PullRequestsService
}

func (c *MockGithubClient) Issues() IssuesService {
	return c.IssuesService
}

type MockPullRequestsService struct {
	CreateFunc func(ctx context.Context, owner string, repo string, newPR *github.NewPullRequest) (*github.PullRequest, *github.Response, error)
}

func (s *MockPullRequestsService) Create(ctx context.Context, owner string, repo string, newPR *github.NewPullRequest) (*github.PullRequest, *github.Response, error) {
	if s.CreateFunc != nil {
		return s.CreateFunc(ctx, owner, repo, newPR)
	}
	return nil, nil, nil
}

type MockIssuesService struct {
	AddLabelsToIssueFunc func(ctx context.Context, owner string, repo string, number int, labels []string) ([]*github.Label, *github.Response, error)
}

func (s *MockIssuesService) AddLabelsToIssue(ctx context.Context, owner string, repo string, number int, labels []string) ([]*github.Label, *github.Response, error) {
	if s.AddLabelsToIssueFunc != nil {
		return s.AddLabelsToIssueFunc(ctx, owner, repo, number, labels)
	}
	return nil, nil, nil
}
