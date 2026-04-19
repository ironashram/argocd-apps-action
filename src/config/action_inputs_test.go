package config

import (
	"reflect"
	"testing"

	"github.com/ironashram/argocd-apps-action/internal/mocks"
	"github.com/ironashram/argocd-apps-action/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewFromInputs(t *testing.T) {
	// Test cases
	testCases := []struct {
		name        string
		action      *mocks.MockActionInterface
		expected    *models.Config
		expectedErr error
	}{
		{
			name: "Test Case 1",
			action: &mocks.MockActionInterface{
				Inputs: map[string]string{
					"skip_prerelease": "true",
					"target_branch":   "main",
					"create_pr":       "true",
					"apps_folder":     "apps",
					"labels":          "github_actions, dependencies",
					"file_extensions": "yaml,yml",
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
				FileExtensions: []string{".yaml", ".yml"},
			},
			expectedErr: nil,
		},
		{
			name: "Test Case 2",
			action: &mocks.MockActionInterface{
				Inputs: map[string]string{
					"skip_prerelease": "false",
					"target_branch":   "develop",
					"create_pr":       "false",
					"apps_folder":     "applications",
					"labels":          "github_actions, dependencies",
					"file_extensions": "yaml,yml",
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
				FileExtensions: []string{".yaml", ".yml"},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.action.On("Debugf", "skip_prerelease: %v", mock.Anything).Once()
			tc.action.On("Debugf", "target_branch: %s", mock.Anything).Once()
			tc.action.On("Debugf", "create_pr: %v", mock.Anything).Once()
			tc.action.On("Debugf", "apps_folder: %s", mock.Anything).Once()
			tc.action.On("Debugf", "file_extensions: %v", mock.Anything).Once()
			tc.action.On("Debugf", "allow_regex_fallback: %v", mock.Anything).Once()
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

func TestNewFromInputs_FileExtensions(t *testing.T) {
	baseInputs := map[string]string{
		"skip_prerelease": "true",
		"target_branch":   "main",
		"create_pr":       "true",
		"apps_folder":     "apps",
		"labels":          "deps",
	}
	baseEnv := map[string]string{
		"GITHUB_TOKEN":      "token",
		"GITHUB_REPOSITORY": "owner/repo",
		"GITHUB_WORKSPACE":  "ws",
	}

	withExt := func(ext string) *mocks.MockActionInterface {
		inputs := make(map[string]string)
		for k, v := range baseInputs {
			inputs[k] = v
		}
		inputs["file_extensions"] = ext
		return &mocks.MockActionInterface{Inputs: inputs, Env: baseEnv}
	}

	setupDebugExpectations := func(action *mocks.MockActionInterface) {
		action.On("Debugf", mock.Anything, mock.Anything).Maybe()
	}

	t.Run("empty string errors", func(t *testing.T) {
		action := withExt("")
		setupDebugExpectations(action)
		_, err := NewFromInputs(action)
		assert.EqualError(t, err, "file_extensions input is empty")
	})

	t.Run("only commas errors", func(t *testing.T) {
		action := withExt(",,,")
		setupDebugExpectations(action)
		_, err := NewFromInputs(action)
		assert.ErrorContains(t, err, "file_extensions input is invalid")
	})

	t.Run("dotted extensions preserved", func(t *testing.T) {
		action := withExt(".yaml,.yml")
		setupDebugExpectations(action)
		cfg, err := NewFromInputs(action)
		assert.NoError(t, err)
		assert.Equal(t, []string{".yaml", ".yml"}, cfg.FileExtensions)
	})

	t.Run("dots prepended when missing", func(t *testing.T) {
		action := withExt("yaml, yml")
		setupDebugExpectations(action)
		cfg, err := NewFromInputs(action)
		assert.NoError(t, err)
		assert.Equal(t, []string{".yaml", ".yml"}, cfg.FileExtensions)
	})

	t.Run("empty entries between commas errors", func(t *testing.T) {
		action := withExt("yaml,,yml,")
		setupDebugExpectations(action)
		_, err := NewFromInputs(action)
		assert.ErrorContains(t, err, "file_extensions input is invalid")
	})
}
