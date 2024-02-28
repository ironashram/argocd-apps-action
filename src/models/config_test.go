package models

import (
	"testing"
)

func TestConfig(t *testing.T) {
	testCases := []struct {
		name         string
		targetBranch string
		createPr     bool
		appsFolder   string
		token        string
		repo         string
		workspace    string
		labels       []string
	}{
		{
			name:         "Test Case 1",
			targetBranch: "main",
			createPr:     true,
			appsFolder:   "apps",
			token:        "abc123",
			repo:         "my-repo",
			workspace:    "my-workspace",
			labels:       []string{"label1", "label2"},
		},
		{
			name:         "Test Case 2",
			targetBranch: "develop",
			createPr:     false,
			appsFolder:   "applications",
			token:        "xyz789",
			repo:         "another-repo",
			workspace:    "another-workspace",
			labels:       []string{"label3", "label4"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := Config{
				TargetBranch: tc.targetBranch,
				CreatePr:     tc.createPr,
				AppsFolder:   tc.appsFolder,
				Token:        tc.token,
				Repo:         tc.repo,
				Workspace:    tc.workspace,
				Labels:       tc.labels,
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
			if len(config.Labels) != len(tc.labels) {
				t.Errorf("Expected Labels to be %v, got %v", tc.labels, config.Labels)
			}
		})
	}
}
