package updater

import (
	"errors"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v33/github"
	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockActionInterface struct {
	mock.Mock
}

func (m *MockActionInterface) Debugf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockActionInterface) Fatalf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockActionInterface) GetInput(name string) string {
	args := m.Called(name)
	return args.String(0)
}

func (m *MockActionInterface) Getenv(name string) string {
	args := m.Called(name)
	return args.String(0)
}

var _ internal.ActionInterface = &MockActionInterface{}

func TestProcessFile(t *testing.T) {
	// Mock data
	path := "/path/to/file.yaml"
	repo := &git.Repository{}
	githubClient := &github.Client{}
	cfg := &internal.Config{
		CreatePr: true,
	}
	action := &MockActionInterface{}

	// Test cases
	testCases := []struct {
		name                  string
		readAndParseYAMLFunc  func(path string) (*internal.Application, error)
		getHTTPResponseFunc   func(url string) ([]byte, error)
		createNewBranchFunc   func(repo *git.Repository, branchName string) error
		commitChangesFunc     func(repo *git.Repository, path, commitMessage string) error
		pushChangesFunc       func(repo *git.Repository, branchName string, cfg *internal.Config) error
		createPullRequestFunc func(client *github.Client, baseBranch, headBranch, title, body string, action internal.ActionInterface) error
		expectedErr           error
	}{
		{
			name: "Valid application manifest",
			readAndParseYAMLFunc: func(path string) (*internal.Application, error) {
				return &internal.Application{
					Spec: internal.Spec{
						Source: internal.Source{
							Chart:          "cert-manager",
							RepoURL:        "https://charts.jetstack.io",
							TargetRevision: "1.14.2",
						},
					},
				}, nil
			},

			getHTTPResponseFunc:   getHTTPResponse,
			createNewBranchFunc:   createNewBranch,
			commitChangesFunc:     commitChanges,
			pushChangesFunc:       pushChanges,
			createPullRequestFunc: createPullRequest,
			expectedErr:           nil,
		},
		{
			name: "Invalid application manifest",
			readAndParseYAMLFunc: func(path string) (*internal.Application, error) {
				return nil, errors.New("failed to parse YAML")
			},
			expectedErr: errors.New("failed to parse YAML"),
		},
	}

	// Run tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock function calls
			readAndParseYAML = tc.readAndParseYAMLFunc
			getHTTPResponse = tc.getHTTPResponseFunc
			createNewBranch = tc.createNewBranchFunc
			commitChanges = tc.commitChangesFunc
			pushChanges = tc.pushChangesFunc
			createPullRequest = tc.createPullRequestFunc

			// Reset mock calls
			action.On("Debugf", mock.Anything, mock.Anything).Return(nil)
			//action.On("Fatalf", mock.Anything, mock.Anything).Return(nil)
			//action.On("GetInput", "inputName").Return("mockedInputValue")
			//action.On("Getenv", "envName").Return("mockedEnvValue")

			// Call the function
			err := processFile(path, repo, githubClient, cfg, action)

			// Assert the result
			assert.Equal(t, tc.expectedErr, err)

			// Verify mock calls
			action.AssertExpectations(t)
		})
	}
}
