package config

import (
	"reflect"
	"testing"

	"github.com/ironashram/argocd-apps-action/models"
	"github.com/ironashram/argocd-apps-action/internal"
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
			expected: &models.Config{
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
			action: &internal.MockActionInterface{
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
			expected: &models.Config{
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
