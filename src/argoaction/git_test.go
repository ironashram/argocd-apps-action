package argoaction

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/google/go-github/v77/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ironashram/argocd-apps-action/internal/mocks"
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
	mockAction := &mocks.MockActionInterface{
		Inputs: map[string]string{},
	}

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

	mockClient := &mocks.MockGithubClient{
		PullRequestsService: &mocks.MockPullRequestsService{
			CreateFunc: func(ctx context.Context, owner string, name string, newPR *github.NewPullRequest) (*github.PullRequest, *github.Response, error) {
				assert.Equal(t, cfg.Owner, owner)
				assert.Equal(t, cfg.Name, name)
				assert.Equal(t, expectedPR, newPR)
				return nil, nil, nil
			},
		},
	}

	u := &Updater{
		GitHubClient: mockClient,
		Config:       cfg,
		Action:       mockAction,
	}

	_, err := u.createPullRequest(baseBranch, newBranch, title, body)
	assert.NoError(t, err)

	expectedError := errors.New("failed to create pull request")
	u.GitHubClient = &mocks.MockGithubClient{
		PullRequestsService: &mocks.MockPullRequestsService{
			CreateFunc: func(ctx context.Context, owner string, repo string, newPR *github.NewPullRequest) (*github.PullRequest, *github.Response, error) {
				return nil, nil, expectedError
			},
		},
	}

	_, err = u.createPullRequest(baseBranch, newBranch, title, body)
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
	mockAction := &mocks.MockActionInterface{
		Inputs: map[string]string{},
	}

	baseBranch := "main"
	newBranch := "feature-branch"
	title := "Test Pull Request"
	body := "This is a test pull request"

	expectedPR := &github.NewPullRequest{
		Title:               github.Ptr(title),
		Head:                github.Ptr(newBranch),
		Base:                github.Ptr(baseBranch),
		Body:                github.Ptr(body),
		MaintainerCanModify: github.Bool(true),
	}

	expectedError := errors.New("failed to create pull request")
	mockClient := &mocks.MockGithubClient{
		PullRequestsService: &mocks.MockPullRequestsService{
			CreateFunc: func(ctx context.Context, owner string, name string, newPR *github.NewPullRequest) (*github.PullRequest, *github.Response, error) {
				assert.Equal(t, cfg.Owner, owner)
				assert.Equal(t, cfg.Name, name)
				assert.Equal(t, expectedPR, newPR)
				return nil, nil, expectedError
			},
		},
	}

	u := &Updater{
		GitHubClient: mockClient,
		Config:       cfg,
		Action:       mockAction,
	}

	_, err := u.createPullRequest(baseBranch, newBranch, title, body)
	assert.EqualError(t, err, expectedError.Error())
}

func TestCreateNewBranch(t *testing.T) {
	gitOps := new(mocks.MockGitRepo)
	worktree := new(mocks.MockWorktree)

	headRef := plumbing.NewHashReference(plumbing.HEAD, plumbing.ZeroHash)
	gitOps.On("Worktree").Return(worktree, nil)
	gitOps.On("Head").Return(headRef, nil)
	gitOps.On("SetReference", mock.Anything, mock.Anything).Return(nil)

	worktree.On("Checkout", mock.Anything).Return(nil)

	u := &Updater{GitOps: gitOps}

	err := u.createNewBranch("base", "new-branch")

	gitOps.AssertExpectations(t)
	worktree.AssertExpectations(t)
	assert.NoError(t, err)
}

func TestCommitChanges(t *testing.T) {
	mockRepo := new(mocks.MockGitRepo)
	worktree := new(mocks.MockWorktree)

	mockRepo.On("Worktree").Return(worktree, nil)
	hash := plumbing.NewHash("0000000000000000000000000000000000000000")
	worktree.On("Add", ".").Return(hash, nil)
	commitHash := plumbing.NewHash("0000000000000000000000000000000000000001")
	worktree.On("Commit", "Test commit", mock.Anything).Return(commitHash, nil)
	worktree.On("Root").Return("/valid/path", nil)

	u := &Updater{GitOps: mockRepo}

	err := u.commitChanges("/valid/path", "Test commit")

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	worktree.AssertExpectations(t)
}

func TestPushChanges(t *testing.T) {
	mockRepo := &mocks.MockGitRepo{}
	mockRepo.On("Push", mock.Anything).Return(nil)

	u := &Updater{
		GitOps: mockRepo,
		Config: &models.Config{Token: "test-token"},
	}

	err := u.pushChanges("test-branch")
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCreateNewBranch_Error(t *testing.T) {
	expectedError := fmt.Errorf("failed to create new branch: %w", errors.New("some error"))
	mockRepo := new(mocks.MockGitRepo)
	worktree := new(mocks.MockWorktree)

	mockRepo.On("Worktree").Return(worktree, nil)
	worktree.On("Checkout", mock.Anything).Return(expectedError)

	u := &Updater{GitOps: mockRepo}

	err := u.createNewBranch("main", "test-branch")

	assert.EqualError(t, err, expectedError.Error())
	mockRepo.AssertExpectations(t)
	worktree.AssertExpectations(t)
}

func TestCommitChanges_Error(t *testing.T) {
	expectedError := fmt.Errorf("failed to commit changes: %w", errors.New("some error"))
	mockRepo := new(mocks.MockGitRepo)
	mockWorktree := new(mocks.MockWorktree)
	mockRepo.On("Worktree").Return(mockWorktree, expectedError)

	u := &Updater{GitOps: mockRepo}

	err := u.commitChanges(".", "Test commit")
	assert.EqualError(t, err, fmt.Errorf("failed to commit changes: %w", expectedError).Error())
	mockRepo.AssertExpectations(t)
	mockWorktree.AssertExpectations(t)
}

func TestPushChanges_Error(t *testing.T) {
	expectedError := fmt.Errorf("failed to push changes: %w", errors.New("some error"))
	mockRepo := &mocks.MockGitRepo{}
	mockRepo.On("Push", mock.Anything).Return(expectedError)

	u := &Updater{
		GitOps: mockRepo,
		Config: &models.Config{Token: "test-token"},
	}

	err := u.pushChanges("test-branch")
	assert.EqualError(t, err, fmt.Errorf("failed to push changes: %w", expectedError).Error())
	mockRepo.AssertExpectations(t)
}
