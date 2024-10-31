package config

import (
	"reflect"
	"testing"

	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"
	"github.com/stretchr/testify/mock"
)

func TestNewFromInputs(t *testing.T) {
	// Test cases
	testCases := []struct {
		name        string
		action      *internal.MockActionInterface
		expected    *models.Config
		expectedErr error
	}{
		{
			name: "Test Case 1",
			action: &internal.MockActionInterface{
				Inputs: map[string]string{
					"skip_prerelease": "true",
					"target_branch":   "main",
					"create_pr":       "true",
					"apps_folder":     "apps",
					"labels":          "github_actions, dependencies",
				},
				Env: map[string]string{
					"GITHUB_TOKEN":      "abc123",
					"GITHUB_REPOSITORY": "githubuser/my-repo",
					"GITHUB_WORKSPACE":  "my-workspace",
				},
			},
			expected: &models.Config{
				SkipPreRelease: true,
				TargetBranch:   "main",
				CreatePr:       true,
				AppsFolder:     "apps",
				Token:          "abc123",
				Repo:           "githubuser/my-repo",
				Workspace:      "my-workspace",
				Owner:          "githubuser",
				Name:           "my-repo",
				Labels:         []string{"github_actions", "dependencies"},
			},
			expectedErr: nil,
		},
		{
			name: "Test Case 2",
			action: &internal.MockActionInterface{
				Inputs: map[string]string{
					"skip_prerelease": "false",
					"target_branch":   "develop",
					"create_pr":       "false",
					"apps_folder":     "applications",
					"labels":          "github_actions, dependencies",
				},
				Env: map[string]string{
					"GITHUB_TOKEN":      "xyz789",
					"GITHUB_REPOSITORY": "githubuser/another-repo",
					"GITHUB_WORKSPACE":  "another-workspace",
				},
			},
			expected: &models.Config{
				SkipPreRelease: false,
				TargetBranch:   "develop",
				CreatePr:       false,
				AppsFolder:     "applications",
				Token:          "xyz789",
				Repo:           "githubuser/another-repo",
				Workspace:      "another-workspace",
				Owner:          "githubuser",
				Name:           "another-repo",
				Labels:         []string{"github_actions", "dependencies"},
			},
			expectedErr: nil,
		},
	}

	// Run tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.action.On("Debugf", "skip_prerelease: %v", mock.Anything).Once()
			tc.action.On("Debugf", "target_branch: %s", mock.Anything).Once()
			tc.action.On("Debugf", "create_pr: %v", mock.Anything).Once()
			tc.action.On("Debugf", "apps_folder: %s", mock.Anything).Once()
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
