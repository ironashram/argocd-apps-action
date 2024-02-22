package internal

import (
	"context"
	"testing"

	"github.com/google/go-github/v59/github"
)

func TestRealGitHubClient_PullRequests(t *testing.T) {
	// Create a real GitHub client
	client := &RealGitHubClient{
		Client: github.NewClient(nil),
	}

	// Get the PullRequestsService
	pullRequestsService := client.PullRequests()

	// Check if the returned PullRequestsService is not nil
	if pullRequestsService == nil {
		t.Error("Expected PullRequestsService to be not nil")
	}
}

func TestMockClient_PullRequests(t *testing.T) {
	// Create a mock PullRequestsService
	mockPullRequestsService := &MockPullRequestsService{}

	// Create a mock client with the mock PullRequestsService
	client := &MockClient{
		PullRequestsService: mockPullRequestsService,
	}

	// Get the PullRequestsService
	pullRequestsService := client.PullRequests()

	// Check if the returned PullRequestsService is the same as the mock PullRequestsService
	if pullRequestsService != mockPullRequestsService {
		t.Error("Expected PullRequestsService to be the same as the mock PullRequestsService")
	}
}

func TestMockPullRequestsService_Create(t *testing.T) {
	// Create a mock PullRequestsService
	mockPullRequestsService := &MockPullRequestsService{}

	// Create a new pull request
	newPR := &github.NewPullRequest{
		Title: github.String("Test Pull Request"),
		Body:  github.String("This is a test pull request"),
	}

	// Call the Create function on the mock PullRequestsService
	_, _, err := mockPullRequestsService.Create(context.Background(), "owner", "repo", newPR)

	// Check if there was an error
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}
