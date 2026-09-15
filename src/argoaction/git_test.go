package argoaction

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ironashram/argocd-apps-action/internal"
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

	u := &Updater{
		Provider: &mocks.MockGitProvider{
			CreatePRFunc: func(ctx context.Context, p internal.NewPR) (*internal.PR, error) {
				assert.Equal(t, internal.NewPR{Title: title, Head: newBranch, Base: baseBranch, Body: body}, p)
				return &internal.PR{Number: 1, HeadRef: newBranch}, nil
			},
		},
		Config: cfg,
		Action: mockAction,
	}

	pr, err := u.createPullRequest(context.Background(), baseBranch, newBranch, title, body)
	assert.NoError(t, err)
	assert.Equal(t, 1, pr.Number)

	expectedError := errors.New("failed to create pull request")
	u.Provider = &mocks.MockGitProvider{
		CreatePRFunc: func(ctx context.Context, p internal.NewPR) (*internal.PR, error) {
			return nil, expectedError
		},
	}

	_, err = u.createPullRequest(context.Background(), baseBranch, newBranch, title, body)
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

	expectedError := errors.New("failed to create pull request")
	u := &Updater{
		Provider: &mocks.MockGitProvider{
			CreatePRFunc: func(ctx context.Context, p internal.NewPR) (*internal.PR, error) {
				assert.Equal(t, internal.NewPR{Title: title, Head: newBranch, Base: baseBranch, Body: body}, p)
				return nil, expectedError
			},
		},
		Config: cfg,
		Action: mockAction,
	}

	_, err := u.createPullRequest(context.Background(), baseBranch, newBranch, title, body)
	assert.EqualError(t, err, expectedError.Error())
}

func TestHandleChartGroup_Scope(t *testing.T) {
	run := func(scope, wantBranch, wantSummary string) {
		gitOps := new(mocks.MockGitRepo)
		worktree := new(mocks.MockWorktree)
		osw := new(mocks.MockOS)

		headRef := plumbing.NewHashReference(plumbing.HEAD, plumbing.ZeroHash)
		gitOps.On("Worktree").Return(worktree, nil)
		gitOps.On("Head").Return(headRef, nil)
		gitOps.On("SetReference", mock.Anything, mock.Anything).Return(nil)
		gitOps.On("Push", mock.Anything).Return(nil)
		worktree.On("Checkout", mock.Anything).Return(nil)
		worktree.On("Root").Return("/repo", nil)
		worktree.On("Add", mock.Anything).Return(plumbing.ZeroHash, nil)
		worktree.On("Commit", wantSummary, mock.Anything).Return(plumbing.ZeroHash, nil)

		manifest := []byte("spec:\n  chart:\n    spec:\n      version: \"1.0.0\"\n")
		osw.On("ReadFile", "/repo/app.yaml").Return(manifest, nil)
		osw.On("WriteFile", "/repo/app.yaml", mock.Anything, mock.Anything).Return(nil)

		mockAction := &mocks.MockActionInterface{Inputs: map[string]string{}}
		mockAction.On("Debugf", mock.Anything, mock.Anything).Maybe()
		mockAction.On("Infof", mock.Anything, mock.Anything).Maybe()

		var gotBranch, gotTitle string
		u := &Updater{
			GitOps: gitOps,
			Provider: &mocks.MockGitProvider{
				CreatePRFunc: func(ctx context.Context, p internal.NewPR) (*internal.PR, error) {
					gotBranch, gotTitle = p.Head, p.Title
					return &internal.PR{Number: 7, HeadRef: p.Head}, nil
				},
				AddLabelsFunc: func(ctx context.Context, number int, labels []string) error { return nil },
			},
			Config: &models.Config{TargetBranch: "main", Scope: scope, Workspace: "/repo"},
			Action: mockAction,
		}

		files := []models.AppFile{{Path: "/repo/app.yaml", CurrentVersion: "1.0.0", VersionPath: "spec.chart.spec.version"}}
		err := u.handleChartGroup(context.Background(), models.ChartRef{Chart: "mychart"}, semver.MustParse("2.0.0"), files, osw)

		assert.NoError(t, err)
		assert.Equal(t, wantBranch, gotBranch)
		assert.Equal(t, wantSummary, gotTitle)
		worktree.AssertExpectations(t)
	}

	run("", "update-mychart-2.0.0", "chore: bump mychart to version 2.0.0")
	run("staging", "update-staging-mychart-2.0.0", "chore: bump mychart to version 2.0.0 (staging)")
	run("production", "update-production-mychart-2.0.0", "chore: bump mychart to version 2.0.0 (production)")
}

func TestHandleChartGroup_DeletesSupersededBranchesOnlyWhenEnabled(t *testing.T) {
	run := func(deleteBranch bool) []string {
		gitOps := new(mocks.MockGitRepo)
		worktree := new(mocks.MockWorktree)
		osw := new(mocks.MockOS)

		headRef := plumbing.NewHashReference(plumbing.HEAD, plumbing.ZeroHash)
		gitOps.On("Worktree").Return(worktree, nil)
		gitOps.On("Head").Return(headRef, nil)
		gitOps.On("SetReference", mock.Anything, mock.Anything).Return(nil)
		gitOps.On("Push", mock.Anything).Return(nil)
		worktree.On("Checkout", mock.Anything).Return(nil)
		worktree.On("Root").Return("/repo", nil)
		worktree.On("Add", mock.Anything).Return(plumbing.ZeroHash, nil)
		worktree.On("Commit", mock.Anything, mock.Anything).Return(plumbing.ZeroHash, nil)

		manifest := []byte("spec:\n  chart:\n    spec:\n      version: \"1.0.0\"\n")
		osw.On("ReadFile", "/repo/app.yaml").Return(manifest, nil)
		osw.On("WriteFile", "/repo/app.yaml", mock.Anything, mock.Anything).Return(nil)

		mockAction := &mocks.MockActionInterface{Inputs: map[string]string{}}
		mockAction.On("Debugf", mock.Anything, mock.Anything).Maybe()
		mockAction.On("Infof", mock.Anything, mock.Anything).Maybe()

		var deleted []string
		u := &Updater{
			GitOps: gitOps,
			Provider: &mocks.MockGitProvider{
				CreatePRFunc: func(ctx context.Context, p internal.NewPR) (*internal.PR, error) {
					return &internal.PR{Number: 50, HeadRef: p.Head}, nil
				},
				DeleteBranchFunc: func(ctx context.Context, branch string) error {
					deleted = append(deleted, branch)
					return nil
				},
			},
			Config: &models.Config{TargetBranch: "main", Workspace: "/repo", DeleteBranch: deleteBranch},
			Action: mockAction,
			openPRs: []internal.PR{
				{Number: 41, HeadRef: "update-netbox-8.3.62"},
				{Number: 44, HeadRef: "update-netbox-operator-8.3.62"},
			},
		}

		files := []models.AppFile{{Path: "/repo/app.yaml", CurrentVersion: "1.0.0", VersionPath: "spec.chart.spec.version"}}
		err := u.handleChartGroup(context.Background(), models.ChartRef{Chart: "netbox"}, semver.MustParse("8.3.63"), files, osw)
		assert.NoError(t, err)
		return deleted
	}

	assert.Empty(t, run(false))
	assert.Equal(t, []string{"update-netbox-8.3.62"}, run(true))
}

func TestHandleChartGroup_ClosesSupersededPRs(t *testing.T) {
	gitOps := new(mocks.MockGitRepo)
	worktree := new(mocks.MockWorktree)
	osw := new(mocks.MockOS)

	headRef := plumbing.NewHashReference(plumbing.HEAD, plumbing.ZeroHash)
	gitOps.On("Worktree").Return(worktree, nil)
	gitOps.On("Head").Return(headRef, nil)
	gitOps.On("SetReference", mock.Anything, mock.Anything).Return(nil)
	gitOps.On("Push", mock.Anything).Return(nil)
	worktree.On("Checkout", mock.Anything).Return(nil)
	worktree.On("Root").Return("/repo", nil)
	worktree.On("Add", mock.Anything).Return(plumbing.ZeroHash, nil)
	worktree.On("Commit", mock.Anything, mock.Anything).Return(plumbing.ZeroHash, nil)

	manifest := []byte("spec:\n  chart:\n    spec:\n      version: \"1.0.0\"\n")
	osw.On("ReadFile", "/repo/app.yaml").Return(manifest, nil)
	osw.On("WriteFile", "/repo/app.yaml", mock.Anything, mock.Anything).Return(nil)

	mockAction := &mocks.MockActionInterface{Inputs: map[string]string{}}
	mockAction.On("Debugf", mock.Anything, mock.Anything).Maybe()
	mockAction.On("Infof", mock.Anything, mock.Anything).Maybe()

	closed := map[int]string{}
	u := &Updater{
		GitOps: gitOps,
		Provider: &mocks.MockGitProvider{
			CreatePRFunc: func(ctx context.Context, p internal.NewPR) (*internal.PR, error) {
				return &internal.PR{Number: 50, HeadRef: p.Head}, nil
			},
			ClosePRFunc: func(ctx context.Context, number int, comment string) error {
				closed[number] = comment
				return nil
			},
		},
		Config: &models.Config{TargetBranch: "main", Scope: "staging", Workspace: "/repo"},
		Action: mockAction,
		openPRs: []internal.PR{
			{Number: 41, HeadRef: "update-staging-netbox-8.3.62"},
			{Number: 42, HeadRef: "update-staging-netbox-8.3.60"},
			{Number: 43, HeadRef: "update-production-netbox-8.3.62"},
			{Number: 44, HeadRef: "update-staging-netbox-operator-8.3.62"},
			{Number: 45, HeadRef: "update-staging-netbox-9.0.0"},
			{Number: 46, HeadRef: "feature/netbox-rework"},
		},
	}

	files := []models.AppFile{{Path: "/repo/app.yaml", CurrentVersion: "1.0.0", VersionPath: "spec.chart.spec.version"}}
	err := u.handleChartGroup(context.Background(), models.ChartRef{Chart: "netbox"}, semver.MustParse("8.3.63"), files, osw)
	assert.NoError(t, err)

	assert.Equal(t, []int{41, 42}, slices.Sorted(maps.Keys(closed)))
	assert.Contains(t, closed[41], "Superseded by #50")
}

func TestHandleChartGroup_RefreshRewritesTitleAndBody(t *testing.T) {
	mockAction := &mocks.MockActionInterface{Inputs: map[string]string{}}
	mockAction.On("Debugf", mock.Anything, mock.Anything).Maybe()
	mockAction.On("Infof", mock.Anything, mock.Anything).Maybe()

	var gotNumber int
	var gotTitle, gotBody string
	u := &Updater{
		Provider: &mocks.MockGitProvider{
			UpdatePRFunc: func(ctx context.Context, number int, title, body string) error {
				gotNumber, gotTitle, gotBody = number, title, body
				return nil
			},
		},
		Config:  &models.Config{TargetBranch: "main", Workspace: "/repo"},
		Action:  mockAction,
		openPRs: []internal.PR{{Number: 12, HeadRef: "update-netbox-8.3.63", Title: "stale title"}},
	}

	files := []models.AppFile{{Path: "/repo/app.yaml", CurrentVersion: "8.3.62"}}
	err := u.handleChartGroup(context.Background(), models.ChartRef{Chart: "netbox"}, semver.MustParse("8.3.63"), files, new(mocks.MockOS))

	assert.NoError(t, err)
	assert.Equal(t, 12, gotNumber)
	assert.Equal(t, "chore: bump netbox to version 8.3.63", gotTitle)
	assert.Contains(t, gotBody, "8.3.62")
}

func TestHandleChartGroup_CreatedBodyCarriesThePinSection(t *testing.T) {
	gitOps := new(mocks.MockGitRepo)
	worktree := new(mocks.MockWorktree)
	osw := new(mocks.MockOS)

	headRef := plumbing.NewHashReference(plumbing.HEAD, plumbing.ZeroHash)
	gitOps.On("Worktree").Return(worktree, nil)
	gitOps.On("Head").Return(headRef, nil)
	gitOps.On("SetReference", mock.Anything, mock.Anything).Return(nil)
	gitOps.On("Push", mock.Anything).Return(nil)
	worktree.On("Checkout", mock.Anything).Return(nil)
	worktree.On("Root").Return("/repo", nil)
	worktree.On("Add", mock.Anything).Return(plumbing.ZeroHash, nil)
	worktree.On("Commit", mock.Anything, mock.Anything).Return(plumbing.ZeroHash, nil)

	manifest := []byte("spec:\n  chart:\n    spec:\n      version: \"1.0.0\"\n")
	osw.On("ReadFile", "/repo/app.yaml").Return(manifest, nil)
	osw.On("WriteFile", "/repo/app.yaml", mock.Anything, mock.Anything).Return(nil)

	mockAction := &mocks.MockActionInterface{Inputs: map[string]string{}}
	mockAction.On("Debugf", mock.Anything, mock.Anything).Maybe()
	mockAction.On("Infof", mock.Anything, mock.Anything).Maybe()

	key := models.ChartRef{RepoURL: "https://charts.example.com", Chart: "app"}
	var gotTitle, gotBody string
	u := &Updater{
		GitOps: gitOps,
		Provider: &mocks.MockGitProvider{
			CreatePRFunc: func(ctx context.Context, p internal.NewPR) (*internal.PR, error) {
				gotTitle, gotBody = p.Title, p.Body
				return &internal.PR{Number: 3, HeadRef: p.Head}, nil
			},
		},
		Config: &models.Config{TargetBranch: "main", Workspace: "/repo", CheckImagePins: true},
		Action: mockAction,
		defaults: map[string]*chartDefaults{
			"https://charts.example.com/app@2.0.0": {
				values: map[string]any{"cache": map[string]any{"image": map[string]any{"tag": "1.6.50"}}},
			},
		},
	}

	files := []models.AppFile{{
		Path:           "/repo/app.yaml",
		CurrentVersion: "1.0.0",
		VersionPath:    "spec.chart.spec.version",
		Pins:           []models.Pin{{Path: "cache.image.tag", Value: "1.6.45"}},
	}}

	err := u.handleChartGroup(context.Background(), key, semver.MustParse("2.0.0"), files, osw)
	assert.NoError(t, err)

	assert.Contains(t, gotTitle, "[1 pin behind]")
	assert.Contains(t, gotBody, "BEHIND cache.image.tag = 1.6.45 (chart default 1.6.50)")
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

	err := u.commitChanges([]string{"/valid/path"}, "Test commit")

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

	err := u.commitChanges([]string{"."}, "Test commit")
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
