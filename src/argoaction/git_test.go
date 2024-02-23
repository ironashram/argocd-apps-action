package argoaction

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/google/go-github/v59/github"
	"github.com/stretchr/testify/assert"

	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"
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

func setupRepoAndInitialCommit() (*git.Repository, error) {
	repo, err := git.Init(memory.NewStorage(), memfs.New())
	if err != nil {
		return nil, err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	file, err := wt.Filesystem.Create("test.txt")
	if err != nil {
		return nil, err
	}

	_, err = file.Write([]byte("This is a test file"))
	if err != nil {
		return nil, err
	}

	file.Close()

	_, err = wt.Add("test.txt")
	if err != nil {
		return nil, err
	}

	_, err = wt.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})

	if err != nil {
		return nil, err
	}

	return repo, nil
}

func TestCreateNewBranch(t *testing.T) {
	repo, err := setupRepoAndInitialCommit()
	assert.NoError(t, err)
	gitOps := &internal.GitRepo{Repo: repo}

	err = createNewBranch(gitOps, "test-branch")
	assert.NoError(t, err)

	ref, err := repo.Reference(plumbing.NewBranchReferenceName("test-branch"), true)
	assert.NoError(t, err)
	assert.NotNil(t, ref)
}

func TestCommitChanges(t *testing.T) {
	repo, err := setupRepoAndInitialCommit()
	assert.NoError(t, err)
	gitOps := &internal.GitRepo{Repo: repo}

	err = commitChanges(gitOps, ".", "Test commit")
	assert.NoError(t, err)

	ref, _ := repo.Head()
	commit, _ := repo.CommitObject(ref.Hash())
	assert.Equal(t, "Test commit", commit.Message)
}

func TestPushChanges(t *testing.T) {
	mockRepo := &internal.MockGitRepo{
		MockPush: func(*git.PushOptions) error {
			return nil
		},
	}

	err := pushChanges(mockRepo, "test-branch", &models.Config{Token: "test-token"})
	assert.NoError(t, err)
}

func TestCreatePullRequest_Error(t *testing.T) {
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

	expectedError := errors.New("failed to create pull request")
	mockClient = &internal.MockGithubClient{
		PullRequestsService: &internal.MockPullRequestsService{
			CreateFunc: func(ctx context.Context, owner string, repo string, newPR *github.NewPullRequest) (*github.PullRequest, *github.Response, error) {
				assert.Equal(t, mockAction.GetInput("owner"), owner)
				assert.Equal(t, mockAction.GetInput("repo"), repo)
				assert.Equal(t, expectedPR, newPR)
				return nil, nil, expectedError
			},
		},
	}

	err := createPullRequest(mockClient, baseBranch, newBranch, title, body, mockAction)

	assert.EqualError(t, err, expectedError.Error())
}

func TestCreateNewBranch_Error(t *testing.T) {
	expectedError := fmt.Errorf("failed to create new branch: %w", errors.New("some error"))
	mockRepo := &internal.MockGitRepo{
		MockSetReference: func(name string, ref *plumbing.Reference) error {
			return errors.New("some error")
		},
		MockHead: func() (*plumbing.Reference, error) {
			return &plumbing.Reference{}, nil
		},
	}
	err := createNewBranch(mockRepo, "test-branch")
	assert.EqualError(t, err, expectedError.Error())
}

func TestCommitChanges_Error(t *testing.T) {
	expectedError := fmt.Errorf("failed to commit changes: %w", errors.New("some error"))
	mockRepo := &internal.MockGitRepo{
		MockWorktree: func() (*git.Worktree, error) {
			return nil, errors.New("some error")
		},
	}
	err := commitChanges(mockRepo, ".", "Test commit")
	assert.EqualError(t, err, expectedError.Error())
}

func TestPushChanges_Error(t *testing.T) {
	expectedError := fmt.Errorf("failed to push changes: %w", errors.New("some error"))
	mockRepo := &internal.MockGitRepo{
		MockPush: func(*git.PushOptions) error {
			return errors.New("some error")
		},
	}
	err := pushChanges(mockRepo, "test-branch", &models.Config{Token: "test-token"})
	assert.EqualError(t, err, expectedError.Error())
}
