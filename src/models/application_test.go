package models

import (
	"testing"
)

func TestSource(t *testing.T) {
	testCases := []struct {
		name     string
		source   Source
		expected string
	}{
		{
			name: "Test Case 1",
			source: Source{
				Chart:          "chart1",
				RepoURL:        "repoURL1",
				TargetRevision: "targetRevision1",
			},
			expected: "chart1",
		},
		{
			name: "Test Case 2",
			source: Source{
				Chart:          "chart2",
				RepoURL:        "repoURL2",
				TargetRevision: "targetRevision2",
			},
			expected: "chart2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.source.Chart
			if result != tc.expected {
				t.Errorf("Expected result: %s, got: %s", tc.expected, result)
			}
		})
	}
}
