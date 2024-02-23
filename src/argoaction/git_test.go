package argoaction

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v59/github"
	"github.com/stretchr/testify/assert"

	"github.com/ironashram/argocd-apps-action/internal"
)

func TestCreatePullRequest(t *testing.T) {
	mockAction := &internal.MockActionInterface{
		Inputs: map[string]string{
			"token": "your-github-token",
			"owner": "your-github-owner",
			"repo":  "your-github-repo",
		},
	}

	var mockClient internal.GitHubClient

	baseBranch := "main"
	newBranch := "feature-branch"
	title := "Test Pull Request"
	body := "This is a test pull request"

	expectedPR := &github.NewPullRequest{
		Title:               github.String(title),
		Head:                github.String(newBranch),
		Base:                github.String(baseBranch),
		Body:                github.String(body),
		MaintainerCanModify: github.Bool(true),
	}

	mockClient = &internal.MockGithubClient{
		PullRequestsService: &internal.MockPullRequestsService{
			CreateFunc: func(ctx context.Context, owner string, repo string, newPR *github.NewPullRequest) (*github.PullRequest, *github.Response, error) {
				assert.Equal(t, mockAction.GetInput("owner"), owner)
				assert.Equal(t, mockAction.GetInput("repo"), repo)
				assert.Equal(t, expectedPR, newPR)
				return nil, nil, nil
			},
		},
	}

	err := createPullRequest(mockClient, baseBranch, newBranch, title, body, mockAction)

	assert.NoError(t, err)

	expectedError := errors.New("failed to create pull request")
	mockClient = &internal.MockGithubClient{
		PullRequestsService: &internal.MockPullRequestsService{
			CreateFunc: func(ctx context.Context, owner string, repo string, newPR *github.NewPullRequest) (*github.PullRequest, *github.Response, error) {
				return nil, nil, expectedError
			},
		},
	}

	err = createPullRequest(mockClient, baseBranch, newBranch, title, body, mockAction)

	assert.EqualError(t, err, expectedError.Error())
}
