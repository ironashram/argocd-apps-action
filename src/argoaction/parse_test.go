package argoaction

import (
	"errors"
	"testing"

	"github.com/Masterminds/semver/v3"
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
		name           string
		path           string
		readFileData   []byte
		readFileErr    error
		expected       *models.Application
		expectedErr    error
		skipPreRelease bool
	}{
		{
			name: "Test Case 1",
			path: "test.yaml",
			readFileData: []byte(`
spec:
  source:
    chart: chart1
    repoURL: repoURL1
    targetRevision: targetRevision1
`),
			readFileErr: nil,
			expected: &models.Application{
				Spec: models.Spec{
					Source: models.Source{
						Chart:          "chart1",
						RepoURL:        "repoURL1",
						TargetRevision: "targetRevision1",
					},
				},
			},
			expectedErr:    nil,
			skipPreRelease: true,
		},
		{
			name:           "Test Case 2",
			path:           "test.yaml",
			readFileData:   nil,
			readFileErr:    errors.New("read file error"),
			expected:       nil,
			expectedErr:    errors.New("read file error"),
			skipPreRelease: true,
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
		versions      []struct {
			Version string `yaml:"version"`
		}
		skipPreRelease bool
		expected       *semver.Version
		expectedErr    error
	}{
		{
			name:          "Test Case 1",
			targetVersion: "1.0.0",
			versions: []struct {
				Version string `yaml:"version"`
			}{
				{Version: "1.1.0"},
				{Version: "1.4.0"},
			},
			skipPreRelease: true,
			expected:       semver.MustParse("1.4.0"),
		},
		{
			name:          "Test Case 2",
			targetVersion: "1.0.0",
			versions: []struct {
				Version string `yaml:"version"`
			}{
				{Version: "0.8.0"},
				{Version: "0.9.5"},
			},
			skipPreRelease: true,
			expected:       nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseNativeNewest(tc.targetVersion, tc.versions, tc.skipPreRelease)

			assert.Equal(t, tc.expected, result)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}
