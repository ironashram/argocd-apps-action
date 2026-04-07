package argoaction

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"

	"github.com/ironashram/argocd-apps-action/internal/mocks"
	"github.com/ironashram/argocd-apps-action/models"

	"github.com/Masterminds/semver/v3"
	"github.com/jarcoal/httpmock"
)

func TestProcessFile(t *testing.T) {
	mockAction := &mocks.MockActionInterface{
		Inputs: map[string]string{},
	}
	mockOSInterface := &mocks.MockOS{}

	cfg := &models.Config{
		CreatePr:       false,
		TargetBranch:   "main",
		FileExtensions: []string{".yaml", ".yml"},
	}

	u := &Updater{
		GitOps:       &mocks.MockGitRepo{},
		GitHubClient: &mocks.MockGithubClient{},
		Config:       cfg,
		Action:       mockAction,
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

		err := u.processFile("invalid1.yaml", mockOSInterface)

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

		err := u.processFile("invalid2.yaml", mockOSInterface)

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

		err := u.processFile("valid.yaml", mockOSInterface)

		assert.NoError(t, err)
		mockAction.AssertExpectations(t)
		mockOSInterface.AssertExpectations(t)
	})
}

func TestUpdateTargetRevision(t *testing.T) {
	mockAction := &mocks.MockActionInterface{}
	mockOSInterface := &mocks.MockOS{}

	fileContent := `spec:
  source:
    chart: chart1
    repoURL: https://test.local
    targetRevision: 0.1.2
`
	expectedContent := `spec:
  source:
    chart: chart1
    repoURL: https://test.local
    targetRevision: 0.2.0
`
	mockOSInterface.On("ReadFile", "test.yaml").Return([]byte(fileContent), nil)
	mockOSInterface.On("WriteFile", "test.yaml", []byte(expectedContent), os.FileMode(0644)).Return(nil)

	newest, _ := semver.NewVersion("0.2.0")

	err := updateTargetRevision(newest, "test.yaml", mockAction, mockOSInterface)

	assert.NoError(t, err)
	mockOSInterface.AssertExpectations(t)

	writeFileCall := mockOSInterface.Calls[1]
	assert.Equal(t, "WriteFile", writeFileCall.Method)

	writtenContent := string(writeFileCall.Arguments[1].([]byte))
	assert.Equal(t, expectedContent, writtenContent)
	assert.Equal(t, 5, strings.Count(writtenContent, "\n"))
}
