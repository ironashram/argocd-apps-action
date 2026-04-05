package mocks

import (
	"context"

	"github.com/google/go-github/v77/github"
	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/stretchr/testify/mock"
)

type MockGithubClient struct {
	PullRequestsService internal.PullRequestsService
	IssuesService       internal.IssuesService
	mock.Mock
}

func (c *MockGithubClient) PullRequests() internal.PullRequestsService {
	return c.PullRequestsService
}

func (c *MockGithubClient) Issues() internal.IssuesService {
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
