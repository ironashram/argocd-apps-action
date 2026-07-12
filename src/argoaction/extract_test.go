package argoaction

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/internal/mocks"
	"github.com/ironashram/argocd-apps-action/models"
)

func TestCollectCandidates_FluxPreset(t *testing.T) {
	dir := t.TempDir()

	write := func(name, content string) {
		if err := os.WriteFile(dir+"/"+name, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	write("metallb-source.yaml", `apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmRepository
metadata:
  name: metallb
  namespace: flux-system
spec:
  url: https://metallb.github.io/metallb
`)
	write("metallb-release.yaml", `apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: metallb
  namespace: metallb-system
spec:
  chart:
    spec:
      chart: metallb
      version: "0.15.2"
      sourceRef:
        kind: HelmRepository
        name: metallb
        namespace: flux-system
`)
	write("private-source.yaml", `apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmRepository
metadata:
  name: private
  namespace: flux-system
spec:
  url: https://example.com/private
  secretRef:
    name: token
`)
	write("private-release.yaml", `apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: private
  namespace: default
spec:
  chart:
    spec:
      chart: privatechart
      version: "1.0.0"
      sourceRef:
        kind: HelmRepository
        name: private
        namespace: flux-system
`)
	write("netbox-oci.yaml", `apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: OCIRepository
metadata:
  name: netbox
  namespace: netbox
spec:
  url: oci://ghcr.io/netbox-community/netbox-chart/netbox
  ref:
    semver: "5.0.78"
`)

	mockAction := &mocks.MockActionInterface{Inputs: map[string]string{}}
	mockAction.On("Debugf", mock.Anything, mock.Anything).Maybe()

	u := &Updater{
		Config:  &models.Config{FileExtensions: []string{".yaml"}},
		Action:  mockAction,
		Sources: fluxPreset(),
	}

	candidates, errs := u.collectCandidates(dir, &internal.OSWrapper{})
	assert.Empty(t, errs)

	metallb := models.ChartRef{RepoURL: "https://metallb.github.io/metallb", Chart: "metallb"}
	netbox := models.ChartRef{RepoURL: "ghcr.io/netbox-community/netbox-chart", Chart: "netbox"}

	assert.Len(t, candidates[metallb], 1)
	assert.Equal(t, "0.15.2", candidates[metallb][0].CurrentVersion)
	assert.Equal(t, "spec.chart.spec.version", candidates[metallb][0].VersionPath)

	assert.Len(t, candidates[netbox], 1)
	assert.Equal(t, "5.0.78", candidates[netbox][0].CurrentVersion)
	assert.Equal(t, "spec.ref.semver", candidates[netbox][0].VersionPath)

	private := models.ChartRef{RepoURL: "https://example.com/private", Chart: "privatechart"}
	assert.Empty(t, candidates[private])
}

func TestWriteVersion_NodePrecise(t *testing.T) {
	content := `apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: metallb
spec:
  chart:
    spec:
      chart: metallb
      version: "0.15.2" # pinned
  interval: 5m
`
	expected := `apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: metallb
spec:
  chart:
    spec:
      chart: metallb
      version: "0.16.1" # pinned
  interval: 5m
`
	f := models.AppFile{VersionPath: "spec.chart.spec.version", CurrentVersion: "0.15.2", DocIndex: 0}
	out := writeVersion([]byte(content), f, "0.16.1")
	assert.Equal(t, expected, string(out))
}

func TestWriteVersion_ArgoTargetRevision(t *testing.T) {
	content := `spec:
  source:
    chart: chart1
    repoURL: https://test.local
    targetRevision: 0.1.2
`
	expected := `spec:
  source:
    chart: chart1
    repoURL: https://test.local
    targetRevision: 0.2.0
`
	f := models.AppFile{VersionPath: "spec.source.targetRevision", CurrentVersion: "0.1.2", DocIndex: 0}
	out := writeVersion([]byte(content), f, "0.2.0")
	assert.Equal(t, expected, string(out))
}

func TestWriteVersion_LeafFallbackOnTemplated(t *testing.T) {
	content := `spec:
  source:
    chart: {{ .Values.chart }}
    repoURL: https://test.local
    targetRevision: 1.2.3
`
	f := models.AppFile{VersionPath: "spec.source.targetRevision", CurrentVersion: "1.2.3", DocIndex: 0}
	out := writeVersion([]byte(content), f, "1.3.0")
	assert.Contains(t, string(out), "targetRevision: 1.3.0")
	assert.NotContains(t, string(out), "targetRevision: 1.2.3")
}

func TestSourcesFor(t *testing.T) {
	argo, err := SourcesFor(&models.Config{Preset: "argocd"}, nil)
	assert.NoError(t, err)
	assert.Len(t, argo.Charts, 1)
	assert.Equal(t, "spec.source.targetRevision", argo.Charts[0].VersionPath)

	flux, err := SourcesFor(&models.Config{Preset: "flux"}, nil)
	assert.NoError(t, err)
	assert.Len(t, flux.Repositories, 1)
	assert.Len(t, flux.Charts, 2)

	empty, err := SourcesFor(&models.Config{Preset: ""}, nil)
	assert.NoError(t, err)
	assert.Equal(t, "spec.source.chart", empty.Charts[0].ChartPath)

	_, err = SourcesFor(&models.Config{Preset: "nonsense"}, nil)
	assert.Error(t, err)
}

func TestSourcesFor_CustomFile(t *testing.T) {
	cfgYAML := `charts:
  - files: ["*.yaml"]
    chartPath: spec.chart
    versionPath: spec.version
    urlPath: spec.repo
`
	mockOS := &mocks.MockOS{}
	mockOS.On("ReadFile", mock.Anything).Return([]byte(cfgYAML), nil)

	sc, err := SourcesFor(&models.Config{SourcesFile: "custom.yaml", Workspace: "/ws"}, mockOS)
	assert.NoError(t, err)
	assert.Len(t, sc.Charts, 1)
	assert.Equal(t, "spec.version", sc.Charts[0].VersionPath)
	assert.Equal(t, "spec.repo", sc.Charts[0].URLPath)
}
