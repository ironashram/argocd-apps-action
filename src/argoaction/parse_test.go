package argoaction

import (
	"errors"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/ironashram/argocd-apps-action/internal/mocks"
	"github.com/ironashram/argocd-apps-action/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
				mockOS := &mocks.MockOS{}
				mockOS.On("ReadFile", tc.path).Return(tc.readFileData, tc.readFileErr)
				mockAction := &mocks.MockActionInterface{}
				mockAction.On("Infof", mock.Anything, mock.Anything).Maybe()

				result, err := readAndParseYAML(mockOS, tc.path, false, mockAction)

				assert.Equal(t, tc.expected, result)
				assert.Equal(t, tc.expectedErr, err)
			})
		}
	})
}

func TestReadAndParseYAML_RegexFallback(t *testing.T) {
	helmTemplated := []byte(`spec:
  source:
    chart: {{ .Values.chart }}
    repoURL: "https://charts.example.com"
    targetRevision: '1.2.3'
`)

	t.Run("fallback disabled surfaces yaml error", func(t *testing.T) {
		mockOS := &mocks.MockOS{}
		mockOS.On("ReadFile", "test.yaml").Return(helmTemplated, nil)
		mockAction := &mocks.MockActionInterface{}

		result, err := readAndParseYAML(mockOS, "test.yaml", false, mockAction)

		assert.Nil(t, result)
		assert.Error(t, err)
	})

	t.Run("fallback enabled extracts fields", func(t *testing.T) {
		mockOS := &mocks.MockOS{}
		mockOS.On("ReadFile", "test.yaml").Return(helmTemplated, nil)
		mockAction := &mocks.MockActionInterface{}
		mockAction.On("Infof", mock.Anything, mock.Anything).Maybe()

		result, err := readAndParseYAML(mockOS, "test.yaml", true, mockAction)

		assert.NoError(t, err)
		assert.Equal(t, "{{ .Values.chart }}", result.Spec.Source.Chart)
		assert.Equal(t, "https://charts.example.com", result.Spec.Source.RepoURL)
		assert.Equal(t, "1.2.3", result.Spec.Source.TargetRevision)
	})

	t.Run("multi-source spec logs warning", func(t *testing.T) {
		multiSource := []byte(`spec:
  sources:
    - chart: {{ .Values.chart }}
      repoURL: "https://charts.example.com"
      targetRevision: 1.0.0
    - chart: other
      repoURL: "https://other.example.com"
      targetRevision: 2.0.0
`)
		mockOS := &mocks.MockOS{}
		mockOS.On("ReadFile", "test.yaml").Return(multiSource, nil)
		mockAction := &mocks.MockActionInterface{}
		mockAction.On("Infof", mock.MatchedBy(func(s string) bool {
			return true
		}), mock.Anything).Return()

		_, err := readAndParseYAML(mockOS, "test.yaml", true, mockAction)

		assert.NoError(t, err)
		mockAction.AssertCalled(t, "Infof", mock.Anything, mock.Anything)
	})
}

func TestPickNewest(t *testing.T) {
	testCases := []struct {
		name           string
		candidates     []string
		skipPreRelease bool
		expected       *semver.Version
	}{
		{
			name:           "Finds newest",
			candidates:     []string{"1.1.0", "1.4.0", "0.9.0"},
			skipPreRelease: true,
			expected:       semver.MustParse("1.4.0"),
		},
		{
			name:           "Empty input",
			candidates:     nil,
			skipPreRelease: true,
			expected:       nil,
		},
		{
			name:           "Skips prereleases when enabled",
			candidates:     []string{"1.0.0", "1.1.0-rc.1"},
			skipPreRelease: true,
			expected:       semver.MustParse("1.0.0"),
		},
		{
			name:           "Includes prereleases when allowed",
			candidates:     []string{"1.0.0", "1.1.0-rc.1"},
			skipPreRelease: false,
			expected:       semver.MustParse("1.1.0-rc.1"),
		},
		{
			name:           "Non-semver entries skipped",
			candidates:     []string{"latest", "stable", "2.0.0", "1.5.0"},
			skipPreRelease: true,
			expected:       semver.MustParse("2.0.0"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAction := &mocks.MockActionInterface{}
			mockAction.On("Debugf", mock.Anything, mock.Anything).Maybe()
			result := pickNewest(tc.candidates, tc.skipPreRelease, mockAction)
			assert.Equal(t, tc.expected, result)
		})
	}
}
