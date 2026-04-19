package internal_test

import (
	"context"
	"testing"

	"github.com/google/go-github/v77/github"
	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/internal/mocks"
)

func TestRealGitHubClient_PullRequests(t *testing.T) {
	client := &internal.RealGitHubClient{
		Client: github.NewClient(nil),
	}

	pullRequestsService := client.PullRequests()

	if pullRequestsService == nil {
		t.Error("Expected PullRequestsService to be not nil")
	}
}

func TestMockClient_PullRequests(t *testing.T) {
	mockPullRequestsService := &mocks.MockPullRequestsService{}

	client := &mocks.MockGithubClient{
		PullRequestsService: mockPullRequestsService,
	}

	pullRequestsService := client.PullRequests()

	if pullRequestsService != mockPullRequestsService {
		t.Error("Expected PullRequestsService to be the same as the mock PullRequestsService")
	}
}

func TestMockPullRequestsService_Create(t *testing.T) {
	mockPullRequestsService := &mocks.MockPullRequestsService{}

	newPR := &github.NewPullRequest{
		Title: github.Ptr("Test Pull Request"),
		Body:  github.Ptr("This is a test pull request"),
	}

	_, _, err := mockPullRequestsService.Create(context.Background(), "owner", "repo", newPR)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}
