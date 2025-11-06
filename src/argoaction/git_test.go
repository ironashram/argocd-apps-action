package argoaction

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/google/go-github/v74/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"
)

func TestCreatePullRequest(t *testing.T) {
	cfg := &models.Config{
		CreatePr:     true,
		TargetBranch: "main",
		Name:         "your-github-repo",
		Owner:        "your-github-owner",
		Token:        "your-github-token",
	}
	mockAction := &internal.MockActionInterface{
		Inputs: map[string]string{},
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
			CreateFunc: func(ctx context.Context, owner string, name string, newPR *github.NewPullRequest) (*github.PullRequest, *github.Response, error) {
				assert.Equal(t, cfg.Owner, owner)
				assert.Equal(t, cfg.Name, name)
				assert.Equal(t, expectedPR, newPR)
				return nil, nil, nil
			},
		},
	}

	_, err := createPullRequest(mockClient, baseBranch, newBranch, title, body, mockAction, cfg)

	assert.NoError(t, err)

	expectedError := errors.New("failed to create pull request")
	mockClient = &internal.MockGithubClient{
		PullRequestsService: &internal.MockPullRequestsService{
			CreateFunc: func(ctx context.Context, owner string, repo string, newPR *github.NewPullRequest) (*github.PullRequest, *github.Response, error) {
				return nil, nil, expectedError
			},
		},
	}

	_, err = createPullRequest(mockClient, baseBranch, newBranch, title, body, mockAction, cfg)

	assert.EqualError(t, err, expectedError.Error())
}

func TestCreatePullRequest_Error(t *testing.T) {
	cfg := &models.Config{
		CreatePr:     true,
		TargetBranch: "main",
		Name:         "your-github-repo",
		Owner:        "your-github-owner",
		Token:        "your-github-token",
	}
	mockAction := &internal.MockActionInterface{
		Inputs: map[string]string{},
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

	expectedError := errors.New("failed to create pull request")
	mockClient = &internal.MockGithubClient{
		PullRequestsService: &internal.MockPullRequestsService{
			CreateFunc: func(ctx context.Context, owner string, name string, newPR *github.NewPullRequest) (*github.PullRequest, *github.Response, error) {
				assert.Equal(t, cfg.Owner, owner)
				assert.Equal(t, cfg.Name, name)
				assert.Equal(t, expectedPR, newPR)
				return nil, nil, expectedError
			},
		},
	}

	_, err := createPullRequest(mockClient, baseBranch, newBranch, title, body, mockAction, cfg)

	assert.EqualError(t, err, expectedError.Error())
}

func TestCreateNewBranch(t *testing.T) {
	gitOps := new(internal.MockGitRepo)
	worktree := new(internal.MockWorktree)

	headRef := plumbing.NewHashReference(plumbing.HEAD, plumbing.ZeroHash)
	gitOps.On("Worktree").Return(worktree, nil)
	gitOps.On("Head").Return(headRef, nil)
	gitOps.On("SetReference", mock.Anything, mock.Anything).Return(nil)

	worktree.On("Checkout", mock.Anything).Return(nil)

	err := createNewBranch(gitOps, "base", "new-branch")

	gitOps.AssertExpectations(t)
	worktree.AssertExpectations(t)
	assert.NoError(t, err)
}

func TestCommitChanges(t *testing.T) {
	mockRepo := new(internal.MockGitRepo)
	worktree := new(internal.MockWorktree)

	mockRepo.On("Worktree").Return(worktree, nil)
	hash := plumbing.NewHash("0000000000000000000000000000000000000000")
	worktree.On("Add", ".").Return(hash, nil)
	commitHash := plumbing.NewHash("0000000000000000000000000000000000000001")
	worktree.On("Commit", "Test commit", mock.Anything).Return(commitHash, nil)
	worktree.On("Root").Return("/valid/path", nil)

	err := commitChanges(mockRepo, "/valid/path", "Test commit")

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	worktree.AssertExpectations(t)
}

func TestPushChanges(t *testing.T) {
	mockRepo := &internal.MockGitRepo{}
	mockRepo.On("Push", mock.Anything).Return(nil)
	err := pushChanges(mockRepo, "test-branch", &models.Config{Token: "test-token"})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCreateNewBranch_Error(t *testing.T) {
	expectedError := fmt.Errorf("failed to create new branch: %w", errors.New("some error"))
	mockRepo := new(internal.MockGitRepo)
	worktree := new(internal.MockWorktree)

	mockRepo.On("Worktree").Return(worktree, nil)

	worktree.On("Checkout", mock.Anything).Return(expectedError)

	err := createNewBranch(mockRepo, "main", "test-branch")

	assert.EqualError(t, err, expectedError.Error())
	mockRepo.AssertExpectations(t)
	worktree.AssertExpectations(t)
}

func TestCommitChanges_Error(t *testing.T) {
	expectedError := fmt.Errorf("failed to commit changes: %w", errors.New("some error"))
	mockRepo := new(internal.MockGitRepo)
	mockWorktree := new(internal.MockWorktree)
	mockRepo.On("Worktree").Return(mockWorktree, expectedError)
	err := commitChanges(mockRepo, ".", "Test commit")
	assert.EqualError(t, err, fmt.Errorf("failed to commit changes: %w", expectedError).Error())
	mockRepo.AssertExpectations(t)
	mockWorktree.AssertExpectations(t)
}

func TestPushChanges_Error(t *testing.T) {
	expectedError := fmt.Errorf("failed to push changes: %w", errors.New("some error"))
	mockRepo := &internal.MockGitRepo{}
	mockRepo.On("Push", mock.Anything).Return(expectedError)
	err := pushChanges(mockRepo, "test-branch", &models.Config{Token: "test-token"})
	assert.EqualError(t, err, fmt.Errorf("failed to push changes: %w", expectedError).Error())
	mockRepo.AssertExpectations(t)
}
