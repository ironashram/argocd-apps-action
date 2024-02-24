package argoaction

import (
	"errors"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockOS struct {
	mock.Mock
}

func (m *MockOS) ReadFile(path string) ([]byte, error) {
	args := m.Called(path)
	return args.Get(0).([]byte), args.Error(1)
}

func TestReadAndParseYAML(t *testing.T) {
	testCases := []struct {
		name         string
		path         string
		readFileData []byte
		readFileErr  error
		expected     *models.Application
		expectedErr  error
	}{
		{
			name:         "Test Case 1",
			path:         "test.yaml",
			readFileData: []byte("spec:\n  source:\n    chart: chart1\n    repoURL: repoURL1\n    targetRevision: targetRevision1\n"),
			readFileErr:  nil,
			expected: &models.Application{
				Spec: models.Spec{
					Source: models.Source{
						Chart:          "chart1",
						RepoURL:        "repoURL1",
						TargetRevision: "targetRevision1",
					},
				},
			},
			expectedErr: nil,
		},
		{
			name:         "Test Case 2",
			path:         "test.yaml",
			readFileData: nil,
			readFileErr:  errors.New("read file error"),
			expected:     nil,
			expectedErr:  errors.New("read file error"),
		},
	}

	t.Run("TestReadAndParseYAML", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockOS := &internal.MockOS{}
				mockOS.On("ReadFile", tc.path).Return(tc.readFileData, tc.readFileErr)

				result, err := readAndParseYAML(mockOS, tc.path)

				assert.Equal(t, tc.expected, result)
				assert.Equal(t, tc.expectedErr, err)
			})
		}
	})
}

func TestGetNewestVersion(t *testing.T) {
	testCases := []struct {
		name          string
		targetVersion string
		entries       map[string][]struct {
			Version string `yaml:"version"`
		}
		expected    *semver.Version
		expectedErr error
	}{
		{
			name:          "Test Case 1",
			targetVersion: "1.0.0",
			entries: map[string][]struct {
				Version string `yaml:"version"`
			}{
				"entry1": {{Version: "1.1.0"}, {Version: "1.2.0"}},
				"entry2": {{Version: "1.3.0"}, {Version: "1.4.0"}},
			},
			expected: semver.MustParse("1.4.0"),
		},
		{
			name:          "Test Case 2",
			targetVersion: "1.0.0",
			entries: map[string][]struct {
				Version string `yaml:"version"`
			}{
				"entry1": {{Version: "0.9.0"}, {Version: "0.8.0"}},
				"entry2": {{Version: "0.7.0"}, {Version: "0.6.0"}},
			},
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := getNewestVersion(tc.targetVersion, tc.entries)

			assert.Equal(t, tc.expected, result)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}
