package argoaction

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/ironashram/argocd-apps-action/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
