package internal

import (
	"context"

	"github.com/google/go-github/v59/github"
	"github.com/stretchr/testify/mock"
)

type GitHubClient interface {
	PullRequests() PullRequestsService
}

type PullRequestsService interface {
	Create(ctx context.Context, owner string, repo string, newPR *github.NewPullRequest) (*github.PullRequest, *github.Response, error)
}

type RealGitHubClient struct {
	Client *github.Client
}

func (c *RealGitHubClient) PullRequests() PullRequestsService {
	return c.Client.PullRequests
}

type MockGithubClient struct {
	PullRequestsService PullRequestsService
	mock.Mock
}

func (c *MockGithubClient) PullRequests() PullRequestsService {
	return c.PullRequestsService
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