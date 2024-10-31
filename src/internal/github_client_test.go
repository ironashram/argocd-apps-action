package internal

import (
	"context"
	"testing"

	"github.com/google/go-github/v66/github"
)

func TestRealGitHubClient_PullRequests(t *testing.T) {
	client := &RealGitHubClient{
		Client: github.NewClient(nil),
	}

	pullRequestsService := client.PullRequests()

	if pullRequestsService == nil {
		t.Error("Expected PullRequestsService to be not nil")
	}
}

func TestMockClient_PullRequests(t *testing.T) {
	mockPullRequestsService := &MockPullRequestsService{}

	client := &MockGithubClient{
		PullRequestsService: mockPullRequestsService,
	}

	pullRequestsService := client.PullRequests()

	if pullRequestsService != mockPullRequestsService {
		t.Error("Expected PullRequestsService to be the same as the mock PullRequestsService")
	}
}

func TestMockPullRequestsService_Create(t *testing.T) {
	mockPullRequestsService := &MockPullRequestsService{}

	newPR := &github.NewPullRequest{
		Title: github.String("Test Pull Request"),
		Body:  github.String("This is a test pull request"),
	}

	_, _, err := mockPullRequestsService.Create(context.Background(), "owner", "repo", newPR)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}
