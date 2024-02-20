package config

import (
	"reflect"
	"testing"

	"github.com/ironashram/argocd-apps-action/internal"
)

func TestConfig(t *testing.T) {
	// Test cases
	testCases := []struct {
		name         string
		targetBranch string
		createPr     bool
		appsFolder   string
		token        string
		repo         string
		workspace    string
	}{
		{
			name:         "Test Case 1",
			targetBranch: "main",
			createPr:     true,
			appsFolder:   "apps",
			token:        "abc123",
			repo:         "my-repo",
			workspace:    "my-workspace",
		},
		{
			name:         "Test Case 2",
			targetBranch: "develop",
			createPr:     false,
			appsFolder:   "applications",
			token:        "xyz789",
			repo:         "another-repo",
			workspace:    "another-workspace",
		},
	}

	// Run tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := internal.Config{
				TargetBranch: tc.targetBranch,
				CreatePr:     tc.createPr,
				AppsFolder:   tc.appsFolder,
				Token:        tc.token,
				Repo:         tc.repo,
				Workspace:    tc.workspace,
			}

			if config.TargetBranch != tc.targetBranch {
				t.Errorf("Expected TargetBranch to be %s, got %s", tc.targetBranch, config.TargetBranch)
			}
			if config.CreatePr != tc.createPr {
				t.Errorf("Expected CreatePr to be %v, got %v", tc.createPr, config.CreatePr)
			}
			if config.AppsFolder != tc.appsFolder {
				t.Errorf("Expected AppsFolder to be %s, got %s", tc.appsFolder, config.AppsFolder)
			}
			if config.Token != tc.token {
				t.Errorf("Expected Token to be %s, got %s", tc.token, config.Token)
			}
			if config.Repo != tc.repo {
				t.Errorf("Expected Repo to be %s, got %s", tc.repo, config.Repo)
			}
			if config.Workspace != tc.workspace {
				t.Errorf("Expected Workspace to be %s, got %s", tc.workspace, config.Workspace)
			}
		})
	}
}

type MockActionInterface struct {
	Inputs map[string]string
	Env    map[string]string
}

func (m MockActionInterface) GetInput(name string) string {
	return m.Inputs[name]
}

func (m MockActionInterface) Getenv(name string) string {
	return m.Env[name]
}

func (m MockActionInterface) Debugf(format string, args ...interface{}) {
}

func (m MockActionInterface) Fatalf(format string, args ...interface{}) {
}

func TestNewFromInputs(t *testing.T) {
	// Test cases
	testCases := []struct {
		name        string
		action      MockActionInterface
		expected    *internal.Config
		expectedErr error
	}{
		{
			name: "Test Case 1",
			action: MockActionInterface{
				Inputs: map[string]string{
					"target_branch": "main",
					"create_pr":     "true",
					"apps_folder":   "apps",
				},
				Env: map[string]string{
					"GITHUB_TOKEN":      "abc123",
					"GITHUB_REPOSITORY": "my-repo",
					"GITHUB_WORKSPACE":  "my-workspace",
				},
			},
			expected: &internal.Config{
				TargetBranch: "main",
				CreatePr:     true,
				AppsFolder:   "apps",
				Token:        "abc123",
				Repo:         "my-repo",
				Workspace:    "my-workspace",
			},
			expectedErr: nil,
		},
		{
			name: "Test Case 2",
			action: MockActionInterface{
				Inputs: map[string]string{
					"target_branch": "develop",
					"create_pr":     "false",
					"apps_folder":   "applications",
				},
				Env: map[string]string{
					"GITHUB_TOKEN":      "xyz789",
					"GITHUB_REPOSITORY": "another-repo",
					"GITHUB_WORKSPACE":  "another-workspace",
				},
			},
			expected: &internal.Config{
				TargetBranch: "develop",
				CreatePr:     false,
				AppsFolder:   "applications",
				Token:        "xyz789",
				Repo:         "another-repo",
				Workspace:    "another-workspace",
			},
			expectedErr: nil,
		},
	}

	// Run tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := NewFromInputs(tc.action)

			if err != tc.expectedErr {
				t.Errorf("Expected error: %v, got: %v", tc.expectedErr, err)
			}

			if !reflect.DeepEqual(config, tc.expected) {
				t.Errorf("Expected config: %+v, got: %+v", tc.expected, config)
			}
		})
	}
}
