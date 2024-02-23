package argoaction

import (
	"errors"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"
)

func TestProcessFile(t *testing.T) {
	mockAction := &internal.MockActionInterface{}
	mockRepo := &git.Repository{}
	mockGitHubClient := &internal.MockGithubClient{}
	mockOSInterface := &internal.MockOS{}

	cfg := &models.Config{
		CreatePr:     true,
		TargetBranch: "main",
	}

	t.Run("Skip invalid application manifest", func(t *testing.T) {
		mockAction.On("Debugf", "Skipping invalid application manifest %s\n", mock.Anything).Once()
		fileContent := []byte("spec:\n  source:\n    chart: chart1\n    repoURL: https://test.local\n    targetRevision: \n")

		mockOSInterface.On("ReadFile", mock.Anything).Return([]byte(fileContent), nil).Once()

		err := processFile("invalid.yaml", mockRepo, mockGitHubClient, cfg, mockAction, mockOSInterface)

		assert.NoError(t, err)
		mockAction.AssertExpectations(t)
		mockOSInterface.AssertExpectations(t)
	})

	t.Run("Skip empty chart, url, and targetRevision", func(t *testing.T) {
		mockAction.On("Debugf", "Skipping invalid application manifest %s\n", mock.Anything).Once()
		fileContent := []byte("spec:\n  source:\n    chart: chart1\n    repoURL: https://test.local\n    targetRevision: \n")
		mockOSInterface.On("ReadFile", mock.Anything).Return([]byte(fileContent), nil).Once()

		err := processFile("valid.yaml", mockRepo, mockGitHubClient, cfg, mockAction, mockOSInterface)

		assert.NoError(t, err)
		mockAction.AssertExpectations(t)
		mockOSInterface.AssertExpectations(t)
	})

	t.Run("Check chart, url, and targetRevision", func(t *testing.T) {
		mockAction.On("Debugf", "Checking %s from %s, current version is %s\n", mock.Anything, mock.Anything, mock.Anything).Once()
		mockOSInterface.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
		mockAction.On("Debugf", "There is a newer %s version: %s\n", mock.Anything, mock.Anything).Once()
		mockGitHubClient.On("CreatePullRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
		mockOSInterface.On("ReadFile", mock.Anything).Return([]byte("file content"), nil).Once()

		err := processFile("valid.yaml", mockRepo, mockGitHubClient, cfg, mockAction, mockOSInterface)

		assert.NoError(t, err)
		mockAction.AssertExpectations(t)
		mockOSInterface.AssertExpectations(t)
		mockGitHubClient.AssertExpectations(t)
	})

	t.Run("No newer version available", func(t *testing.T) {
		mockAction.On("Debugf", "Checking %s from %s, current version is %s\n", mock.Anything, mock.Anything, mock.Anything).Once()
		mockAction.On("Debugf", "No newer version of %s is available\n", mock.Anything).Once()
		mockOSInterface.On("ReadFile", mock.Anything).Return([]byte("file content"), nil).Once()

		err := processFile("valid.yaml", mockRepo, mockGitHubClient, cfg, mockAction, mockOSInterface)

		assert.NoError(t, err)
		mockAction.AssertExpectations(t)
		mockOSInterface.AssertExpectations(t)
	})

	t.Run("Error creating pull request", func(t *testing.T) {
		mockAction.On("Debugf", "Checking %s from %s, current version is %s\n", mock.Anything, mock.Anything, mock.Anything).Once()
		mockOSInterface.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
		mockAction.On("Debugf", "There is a newer %s version: %s\n", mock.Anything, mock.Anything).Once()
		mockGitHubClient.On("CreatePullRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("error creating pull request")).Once()
		mockOSInterface.On("ReadFile", mock.Anything).Return([]byte("file content"), nil).Once()

		err := processFile("valid.yaml", mockRepo, mockGitHubClient, cfg, mockAction, mockOSInterface)

		assert.Error(t, err)
		mockAction.AssertExpectations(t)
		mockOSInterface.AssertExpectations(t)
		mockGitHubClient.AssertExpectations(t)
	})
}
