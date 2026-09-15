package argoaction

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/ironashram/argocd-apps-action/models"
)

func decodeDoc(t *testing.T, doc string) map[string]any {
	t.Helper()
	var m map[string]any
	assert.NoError(t, yaml.Unmarshal([]byte(doc), &m))
	return m
}

func pinPaths(pins []models.Pin) map[string]string {
	out := map[string]string{}
	for _, p := range pins {
		out[p.Path] = p.Value
	}
	return out
}

func TestCollectPins_ValuesTree(t *testing.T) {
	doc := decodeDoc(t, `
spec:
  values:
    image:
      tag: "1.0.0"
    cache:
      image:
        repository: cache
        tag: 1.6.45-alpine
    exporter:
      image:
        digest: sha256:abc
    nested:
      deeper:
        image:
          tag: v2.3.4
    unrelated:
      tag: not-a-pin
`)
	rule := models.ChartRule{ValuesPath: "spec.values"}

	got := pinPaths(collectPins(doc, rule))

	assert.Equal(t, map[string]string{
		"image.tag":               "1.0.0",
		"cache.image.tag":         "1.6.45-alpine",
		"exporter.image.digest":   "sha256:abc",
		"nested.deeper.image.tag": "v2.3.4",
	}, got)
}

func TestCollectPins_ImageAsString(t *testing.T) {
	doc := decodeDoc(t, `
spec:
  values:
    a:
      image: "registry.example.com:5000/team/app:3.1.0"
    b:
      image: "app-with-no-tag"
`)
	rule := models.ChartRule{ValuesPath: "spec.values"}

	assert.Equal(t, map[string]string{"a.image": "3.1.0"}, pinPaths(collectPins(doc, rule)))
}

func TestCollectPins_AllArgoShapes(t *testing.T) {
	doc := decodeDoc(t, `
spec:
  source:
    helm:
      valuesObject:
        one:
          image:
            tag: "1.1.1"
      values: |
        two:
          image:
            tag: "2.2.2"
      parameters:
        - name: three.image.tag
          value: "3.3.3"
        - name: replicas
          value: "2"
      fileParameters:
        - name: four.image.tag
          path: values/tag.txt
`)
	rule := models.ChartRule{
		ValuesPath:         "spec.source.helm.valuesObject",
		ValuesStringPath:   "spec.source.helm.values",
		ParametersPath:     "spec.source.helm.parameters",
		FileParametersPath: "spec.source.helm.fileParameters",
	}

	pins := collectPins(doc, rule)

	assert.Equal(t, map[string]string{
		"one.image.tag":   "1.1.1",
		"two.image.tag":   "2.2.2",
		"three.image.tag": "3.3.3",
		"four.image.tag":  "",
	}, pinPaths(pins))

	for _, p := range pins {
		if p.Path == "four.image.tag" {
			assert.True(t, p.Opaque)
		}
	}
}

func TestCompareTags(t *testing.T) {
	cases := []struct {
		pinned, chartDefault string
		want                 PinState
	}{
		{"1.6.45-alpine", "1.6.50-alpine", PinBehind},
		{"1.6.50-alpine", "1.6.45-alpine", PinAhead},
		{"1.6.45-alpine", "1.6.45-alpine", PinRedundant},
		{"1.6.45-alpine", "1.6.45-debian", PinDiffers},
		{"1.6.45-alpine", "1.6.45", PinDiffers},
		{"v0.17.0", "v0.15.4", PinAhead},
		{"v0.17.0", "v0.17.0", PinRedundant},
		{"1.2", "1.2.0", PinRedundant},
		{"1.10.0", "1.9.0", PinAhead},
		{"20260915", "20260801", PinAhead},
		{"latest", "1.0.0", PinDiffers},
		{"latest", "latest", PinDiffers},
		{"5.15.0-1-ce", "5.13.0-1-ce@sha256:aa", PinAhead},
		{"0.74.0@sha256:aa", "0.74.0", PinDiffers},
		{"0.74.0@sha256:aa", "0.74.0@sha256:aa", PinRedundant},
		{"0.74.0@sha256:aa", "0.74.0@sha256:bb", PinDiffers},
		{"1.0.0@sha256:aa", "2.0.0@sha256:bb", PinBehind},
	}

	for _, c := range cases {
		assert.Equal(t, c.want, compareTags(c.pinned, c.chartDefault), "%s vs %s", c.pinned, c.chartDefault)
	}
}

func TestComparePin(t *testing.T) {
	sameDigest := comparePin(models.Pin{Path: "a.image.digest", Value: "sha256:aa", Digest: true}, "sha256:aa")
	assert.Equal(t, PinRedundant, sameDigest.State)

	otherDigest := comparePin(models.Pin{Path: "a.image.digest", Value: "sha256:aa", Digest: true}, "sha256:bb")
	assert.Equal(t, PinDiffers, otherDigest.State)

	behind := comparePin(models.Pin{Path: "a.image.tag", Value: "1.0.0"}, "1.1.0")
	assert.Equal(t, PinBehind, behind.State)
}

func TestPinScanOutput(t *testing.T) {
	scan := PinScan{Reports: []PinReport{
		{Pin: models.Pin{Path: "memcached.image.tag", Value: "1.6.45-alpine"}, Default: "1.6.50-alpine", State: PinBehind},
		{Pin: models.Pin{Path: "memcachedExporter.image.tag", Value: "v0.17.0"}, Default: "v0.17.0", State: PinRedundant},
		{Pin: models.Pin{Path: "foo.image.tag", Value: "2.0.1"}, Default: "2.0.0", State: PinAhead},
	}}

	assert.Equal(t, " [1 pin behind, 1 redundant]", scan.titleSuffix())

	body := scan.bodySection("loki", "7.4.0")
	assert.Contains(t, body, "Image pins checked against loki 7.4.0 defaults:")
	assert.Contains(t, body, "BEHIND    memcached.image.tag = 1.6.45-alpine (chart default 1.6.50-alpine)")
	assert.Contains(t, body, "REDUNDANT memcachedExporter.image.tag = v0.17.0 (matches chart default)")
	assert.Contains(t, body, "AHEAD     foo.image.tag = 2.0.1 (chart default 2.0.0)")
}

func TestPinScanOutput_QuietWhenNothingActionable(t *testing.T) {
	scan := PinScan{Reports: []PinReport{
		{Pin: models.Pin{Path: "a.image.tag", Value: "2.0.1"}, Default: "2.0.0", State: PinAhead},
	}}
	assert.Equal(t, "", scan.titleSuffix())
	assert.Contains(t, scan.bodySection("c", "1.0.0"), "AHEAD")

	assert.Equal(t, "", PinScan{}.titleSuffix())
	assert.Equal(t, "", PinScan{}.bodySection("c", "1.0.0"))
}

func TestScanPins_OnlyReportsPathsTheChartDefines(t *testing.T) {
	defaults := map[string]string{
		"cache.image.tag": "1.6.50-alpine",
		"app.image":       "registry.example.com:5000/team/app:4.0.0",
	}
	lookup := func(path string) (string, bool) {
		v, ok := defaults[path]
		return v, ok
	}

	pins := []models.Pin{
		{Path: "cache.image.tag", Value: "1.6.45-alpine"},
		{Path: "app.image", Value: "3.1.0", Ref: true},
		{Path: "ghost.image.tag", Value: "9.9.9"},
		{Path: "opaque.image.tag", Opaque: true},
	}

	scan := scanPins(pins, lookup)

	assert.Equal(t, []string{"ghost.image.tag", "opaque.image.tag"}, scan.Skipped)
	assert.Len(t, scan.Reports, 2)
	assert.Equal(t, "cache.image.tag", scan.Reports[0].Pin.Path)
	assert.Equal(t, PinBehind, scan.Reports[0].State)
	assert.Equal(t, "app.image", scan.Reports[1].Pin.Path)
	assert.Equal(t, "4.0.0", scan.Reports[1].Default)
	assert.Equal(t, PinBehind, scan.Reports[1].State)
}
