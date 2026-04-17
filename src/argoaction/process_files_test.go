package argoaction

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/yaml"

	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/internal/mocks"
	"github.com/ironashram/argocd-apps-action/models"

	"github.com/Masterminds/semver/v3"
	"github.com/jarcoal/httpmock"
)

func TestProcessChartGroup_NewerVersion_CreatePrDisabled(t *testing.T) {
	mockAction := &mocks.MockActionInterface{Inputs: map[string]string{}}
	mockOS := &mocks.MockOS{}

	u := &Updater{
		Config: &models.Config{CreatePr: false},
		Action: mockAction,
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	entries := models.Index{
		Entries: map[string][]struct {
			Version string `yaml:"version"`
		}{
			"chart1": {{Version: "0.9.0"}, {Version: "0.8.0"}},
		},
	}
	responder := func(req *http.Request) (*http.Response, error) {
		data, _ := yaml.Marshal(entries)
		return httpmock.NewBytesResponse(200, data), nil
	}
	httpmock.RegisterResponder("GET", "https://test.local/index.yaml", responder)

	mockAction.On("Debugf", mock.Anything, mock.Anything).Maybe()
	mockAction.On("Infof", "There is a newer %s version: %s (%d file(s) to update)", mock.Anything).Once()
	mockAction.On("Infof", "Create PR is disabled, skipping PR creation for %s", mock.Anything).Once()

	key := models.ChartRef{RepoURL: "https://test.local", Chart: "chart1"}
	files := []models.AppFile{{Path: "a.yaml", CurrentVersion: "0.1.2"}}

	err := u.processChartGroup(context.Background(), key, files, mockOS)

	assert.NoError(t, err)
	mockAction.AssertExpectations(t)
}

func TestProcessChartGroup_NativeFetchError(t *testing.T) {
	mockAction := &mocks.MockActionInterface{Inputs: map[string]string{}}
	mockOS := &mocks.MockOS{}

	u := &Updater{
		Config: &models.Config{},
		Action: mockAction,
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://broken.local/index.yaml",
		httpmock.NewStringResponder(500, "server error"))

	mockAction.On("Debugf", mock.Anything, mock.Anything).Maybe()
	mockAction.On("Infof", "Error getting versions for %s: %v", mock.Anything).Once()

	key := models.ChartRef{RepoURL: "https://broken.local", Chart: "mychart"}
	files := []models.AppFile{{Path: "broken.yaml", CurrentVersion: "1.0.0"}}

	err := u.processChartGroup(context.Background(), key, files, mockOS)

	assert.NoError(t, err)
	mockAction.AssertExpectations(t)
}

func TestProcessChartGroup_NonSemverVersionsSkipped(t *testing.T) {
	mockAction := &mocks.MockActionInterface{Inputs: map[string]string{}}
	mockOS := &mocks.MockOS{}

	u := &Updater{
		Config: &models.Config{CreatePr: false},
		Action: mockAction,
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	entries := models.Index{
		Entries: map[string][]struct {
			Version string `yaml:"version"`
		}{
			"mychart": {
				{Version: "latest"},
				{Version: "stable"},
				{Version: "2.0.0"},
			},
		},
	}
	responder := func(req *http.Request) (*http.Response, error) {
		data, _ := yaml.Marshal(entries)
		return httpmock.NewBytesResponse(200, data), nil
	}
	httpmock.RegisterResponder("GET", "https://test.local/index.yaml", responder)

	mockAction.On("Debugf", mock.Anything, mock.Anything).Maybe()
	mockAction.On("Infof", "There is a newer %s version: %s (%d file(s) to update)", mock.Anything).Once()
	mockAction.On("Infof", "Create PR is disabled, skipping PR creation for %s", mock.Anything).Once()

	key := models.ChartRef{RepoURL: "https://test.local", Chart: "mychart"}
	files := []models.AppFile{{Path: "test.yaml", CurrentVersion: "1.0.0"}}

	err := u.processChartGroup(context.Background(), key, files, mockOS)

	assert.NoError(t, err)
	mockAction.AssertExpectations(t)
}

func TestProcessChartGroup_NoBumpWhenAllAhead(t *testing.T) {
	mockAction := &mocks.MockActionInterface{Inputs: map[string]string{}}
	mockOS := &mocks.MockOS{}

	u := &Updater{
		Config: &models.Config{CreatePr: true},
		Action: mockAction,
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	entries := models.Index{
		Entries: map[string][]struct {
			Version string `yaml:"version"`
		}{
			"chart1": {{Version: "1.0.0"}},
		},
	}
	responder := func(req *http.Request) (*http.Response, error) {
		data, _ := yaml.Marshal(entries)
		return httpmock.NewBytesResponse(200, data), nil
	}
	httpmock.RegisterResponder("GET", "https://test.local/index.yaml", responder)

	mockAction.On("Debugf", mock.Anything, mock.Anything).Maybe()

	key := models.ChartRef{RepoURL: "https://test.local", Chart: "chart1"}
	files := []models.AppFile{{Path: "a.yaml", CurrentVersion: "2.0.0"}}

	err := u.processChartGroup(context.Background(), key, files, mockOS)

	assert.NoError(t, err)
	mockAction.AssertExpectations(t)
}

func TestCollectCandidates_GroupsByChartAndRepo(t *testing.T) {
	dir := t.TempDir()

	writeFile := func(name, content string) {
		if err := os.WriteFile(dir+"/"+name, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	writeFile("a.yaml", `spec:
  source:
    chart: foo
    repoURL: https://charts.example.com
    targetRevision: 1.0.0
`)
	writeFile("b.yaml", `spec:
  source:
    chart: foo
    repoURL: https://charts.example.com
    targetRevision: 1.1.0
`)
	writeFile("c.yaml", `spec:
  source:
    chart: bar
    repoURL: https://charts.example.com
    targetRevision: 2.0.0
`)
	writeFile("invalid.yaml", `spec:
  source:
    chart: noversion
    repoURL: https://charts.example.com
`)

	mockAction := &mocks.MockActionInterface{Inputs: map[string]string{}}
	mockAction.On("Debugf", mock.Anything, mock.Anything).Maybe()

	u := &Updater{
		Config: &models.Config{FileExtensions: []string{".yaml"}},
		Action: mockAction,
	}

	candidates, errs := u.collectCandidates(dir, &internal.OSWrapper{})
	assert.Empty(t, errs)
	assert.Len(t, candidates, 2)

	fooKey := models.ChartRef{RepoURL: "https://charts.example.com", Chart: "foo"}
	barKey := models.ChartRef{RepoURL: "https://charts.example.com", Chart: "bar"}
	assert.Len(t, candidates[fooKey], 2)
	assert.Len(t, candidates[barKey], 1)
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
