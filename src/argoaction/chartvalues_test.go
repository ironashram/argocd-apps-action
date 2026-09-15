package argoaction

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/stretchr/testify/assert"
)

func buildArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		assert.NoError(t, tw.WriteHeader(&tar.Header{
			Name:     name,
			Mode:     0o644,
			Size:     int64(len(body)),
			Typeflag: tar.TypeReg,
		}))
		_, err := tw.Write([]byte(body))
		assert.NoError(t, err)
	}
	assert.NoError(t, tw.Close())
	assert.NoError(t, gz.Close())
	return buf.Bytes()
}

func TestReadChartDefaults(t *testing.T) {
	archive := buildArchive(t, map[string]string{
		"app/Chart.yaml":               "name: app\ndependencies:\n  - name: cache\n    alias: sidecache\n",
		"app/values.yaml":              "image:\n  tag: 1.0.0\nsidecache:\n  enabled: true\n",
		"app/charts/cache/values.yaml": "image:\n  tag: 2.0.0\n",
		"app/charts/other/values.yaml": "image:\n  tag: 3.0.0\n",
	})

	defaults, err := readChartDefaults(archive)
	assert.NoError(t, err)

	top, _, ok := defaults.lookup("image.tag")
	assert.True(t, ok)
	assert.Equal(t, "1.0.0", top)

	aliased, _, ok := defaults.lookup("sidecache.image.tag")
	assert.True(t, ok)
	assert.Equal(t, "2.0.0", aliased)

	unaliased, _, ok := defaults.lookup("other.image.tag")
	assert.True(t, ok)
	assert.Equal(t, "3.0.0", unaliased)

	_, _, ok = defaults.lookup("cache.image.tag")
	assert.False(t, ok)

	_, _, ok = defaults.lookup("nothing.image.tag")
	assert.False(t, ok)
}

func TestReadChartDefaults_ParentWinsOverDependency(t *testing.T) {
	archive := buildArchive(t, map[string]string{
		"app/Chart.yaml":               "name: app\n",
		"app/values.yaml":              "cache:\n  image:\n    tag: parent-1.0\n",
		"app/charts/cache/values.yaml": "image:\n  tag: dep-9.9\n",
	})

	defaults, err := readChartDefaults(archive)
	assert.NoError(t, err)

	got, _, ok := defaults.lookup("cache.image.tag")
	assert.True(t, ok)
	assert.Equal(t, "parent-1.0", got)
}

func TestReadChartDefaults_EmptyTagUsesAppVersionOnlyWhenTemplatesDo(t *testing.T) {
	archive := buildArchive(t, map[string]string{
		"app/Chart.yaml":  "name: app\nappVersion: v2.5.0\n",
		"app/values.yaml": "image:\n  tag:\n  repository: app\nrequired:\n  image:\n    tag:\n",
		"app/templates/deploy.yaml": "image: {{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}\n" +
			"other: {{ .Values.required.image.tag }}\n",
		"app/charts/dep/Chart.yaml":        "name: dep\nappVersion: v0.0.25\n",
		"app/charts/dep/values.yaml":       "image:\n  tag:\n",
		"app/charts/dep/templates/ds.yaml": "image: {{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}\n",
	})

	defaults, err := readChartDefaults(archive)
	assert.NoError(t, err)

	got, fromAppVersion, ok := defaults.lookup("image.tag")
	assert.True(t, ok)
	assert.True(t, fromAppVersion)
	assert.Equal(t, "v2.5.0", got)

	depGot, depFromAppVersion, ok := defaults.lookup("dep.image.tag")
	assert.True(t, ok)
	assert.True(t, depFromAppVersion)
	assert.Equal(t, "v0.0.25", depGot)

	_, _, ok = defaults.lookup("required.image.tag")
	assert.False(t, ok)
}

func TestReadChartDefaults_NoValues(t *testing.T) {
	archive := buildArchive(t, map[string]string{"app/Chart.yaml": "name: app\n"})
	_, err := readChartDefaults(archive)
	assert.Error(t, err)
}
