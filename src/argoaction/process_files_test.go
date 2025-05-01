package argoaction

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"

	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"

	"github.com/jarcoal/httpmock"
)

func TestProcessFile(t *testing.T) {
	mockAction := &internal.MockActionInterface{
		Inputs: map[string]string{},
	}
	mockRepo := &internal.MockGitRepo{}
	mockGitHubClient := &internal.MockGithubClient{}
	mockOSInterface := &internal.MockOS{}

	cfg := &models.Config{
		CreatePr:     false,
		TargetBranch: "main",
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	entries := models.Index{
		Entries: map[string][]struct {
			Version string `yaml:"version"`
		}{
			"chart1": {{Version: "0.9.0"}, {Version: "0.8.0"}},
			"chart2": {{Version: "1.7.0"}, {Version: "0.6.0"}},
		},
	}

	responder := func(req *http.Request) (*http.Response, error) {
		yamlData, err := yaml.Marshal(entries)
		if err != nil {
			return httpmock.NewStringResponse(500, ""), err
		}

		return httpmock.NewBytesResponse(200, yamlData), nil
	}

	httpmock.RegisterResponder("GET", "https://test.local/index.yaml", responder)

	t.Run("Skip invalid application manifest", func(t *testing.T) {
		mockAction.On("Debugf", "Skipping invalid application manifest %s", mock.AnythingOfType("[]interface {}")).Once()
		fileContent := []byte(`
spec:
  source:
    fakechart: chart1
    repoURL: https://test.local
`)

		mockOSInterface.On("ReadFile", mock.Anything).Return([]byte(fileContent), nil).Once()

		err := processFile("invalid1.yaml", mockRepo, mockGitHubClient, cfg, mockAction, mockOSInterface)

		assert.NoError(t, err)
		mockAction.AssertExpectations(t)
		mockOSInterface.AssertExpectations(t)
	})

	t.Run("Skip empty chart, url, and targetRevision", func(t *testing.T) {
		mockAction.On("Debugf", "Skipping invalid application manifest %s", mock.AnythingOfType("[]interface {}")).Once()
		fileContent := []byte(`
spec:
  source:
    chart: chart1
    repoURL: https://test.local
    targetRevision:
`)
		mockOSInterface.On("ReadFile", mock.Anything).Return([]byte(fileContent), nil).Once()

		err := processFile("invalid2.yaml", mockRepo, mockGitHubClient, cfg, mockAction, mockOSInterface)

		assert.NoError(t, err)
		mockAction.AssertExpectations(t)
		mockOSInterface.AssertExpectations(t)
	})

	t.Run("Check chart, url, and targetRevision", func(t *testing.T) {
		mockAction.On("Debugf", "Checking %s from %s, current version is %s", mock.AnythingOfType("[]interface {}")).Once()
		mockAction.On("Infof", "There is a newer %s version: %s", mock.AnythingOfType("[]interface {}")).Once()
		mockAction.On("Infof", "Create PR is disabled, skipping PR creation for %s", mock.AnythingOfType("[]interface {}")).Once()
		fileContent := []byte(`
spec:
  source:
    chart: chart1
    repoURL: https://test.local
    targetRevision: 0.1.2
`)
		mockOSInterface.On("ReadFile", mock.Anything).Return([]byte(fileContent), nil).Once()

		err := processFile("valid.yaml", mockRepo, mockGitHubClient, cfg, mockAction, mockOSInterface)

		assert.NoError(t, err)
		mockAction.AssertExpectations(t)
		mockOSInterface.AssertExpectations(t)
		mockGitHubClient.AssertExpectations(t)
	})
}
