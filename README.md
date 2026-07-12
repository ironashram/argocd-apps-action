[![Go unit tests](https://github.com/ironashram/argocd-apps-action/actions/workflows/go-unit-tests.yaml/badge.svg)](https://github.com/ironashram/argocd-apps-action/actions/workflows/go-unit-tests.yaml)
[![go: build binaries](https://github.com/ironashram/argocd-apps-action/actions/workflows/go-binary.yaml/badge.svg)](https://github.com/ironashram/argocd-apps-action/actions/workflows/go-binary.yaml)

## ArgoCD Apps Action
This action bumps Helm chart versions pinned in GitOps manifests and opens a pull request when a newer chart release is available. It works with both ArgoCD `Application` manifests and Flux `HelmRelease`/`OCIRepository` manifests, and against both GitHub and Forgejo/Gitea (auto-detected). It is written in Go and uses `github.com/sethvargo/go-githubactions` for the Actions runtime.

## How it works

The action walks the configured directory and its subdirectories, looking for files matching the configured extensions (default: `yaml`, `yml`), and extracts each pinned chart's name, repository URL and current version according to the selected `preset`:

- `argocd` (default): reads `spec.source.{chart,repoURL,targetRevision}` from `Application` manifests.
- `flux`: reads chart + version from `HelmRelease` (`spec.chart.spec.{chart,version}`), resolving the repository URL from the referenced `HelmRepository` via `sourceRef`; and reads `OCIRepository` charts directly (`spec.url` + `spec.ref.semver`). Repositories with a `secretRef` (private) are skipped unless a matching entry exists in `repo_credentials`.

For each chart it fetches the available versions (Helm `index.yaml` for HTTP repos, or the registry tags via `oras.land/oras-go` for OCI repos) and, if a newer version exists, edits the exact version field in place and opens a pull request. Private repositories are supported through the `repo_credentials` input.

Only fixed pins (`X.Y.Z`, optionally `v`-prefixed) are ever bumped. Semver ranges and partial versions (`1.x`, `2.*`, `~1.2.0`, `6.5`) are left untouched - resolving those is the GitOps tool's job. The pull request is created through the git provider's REST API selected by `provider`/`GITHUB_API_URL`, so the same action works on GitHub and Forgejo/Gitea.

For layouts not covered by the presets, set `sources_file` to a custom extraction config (see `preset` definitions in `src/argoaction/extract.go` for the schema).

## Usage

Example GitHub workflow:

```yaml
name: "ArgoCD App Updates"

on:
  schedule:
    - cron:  '0 7 * * MON'
  workflow_dispatch:

jobs:

  update:
    runs-on: ubuntu-latest
    permissions:
        contents: write
        pull-requests: write
    steps:

      - name: Check out
        uses: actions/checkout@v4
        with:
          fetch-depth: '0'

      - name: Check updates for ArgoCD Apps
        uses: ironashram/argocd-apps-action@v2.2.0
        with:
          skip_prerelease: true
          target_branch: main
          create_pr: true
          apps_folder: apps/manifests
          file_extensions: yaml,yml
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Presets and custom layouts

Two built-in presets cover the common cases:

- `preset: argocd` (default) - ArgoCD `Application` manifests (`spec.source.*`).
- `preset: flux` - Flux `HelmRelease` + `HelmRepository`/`OCIRepository` manifests.

For any other layout, set `sources_file` to a YAML file in your repo describing where the chart, version and repository live. It overrides `preset` and is run by the same engine. For example, this reproduces the Flux preset:

```yaml
# .github/chart-sources.yaml
repositories:            # build a name/namespace -> url index for by-reference repos
  - files: ["*"]         # basename globs; "*" matches all scanned files
    namePath: metadata.name
    namespacePath: metadata.namespace
    urlPath: spec.url
    skipIfSet: spec.secretRef   # skip private repos
charts:
  - files: ["*"]
    chartPath: spec.chart.spec.chart
    versionPath: spec.chart.spec.version      # the field that gets bumped
    repoRef:                                  # resolve repo url via the index above
      namePath: spec.chart.spec.sourceRef.name
      namespacePath: spec.chart.spec.sourceRef.namespace
  - files: ["*"]                              # OCIRepository: chart is the url basename
    urlPath: spec.url
    versionPath: spec.ref.semver
```

```yaml
      - uses: ironashram/argocd-apps-action@v3.0.0
        with:
          sources_file: .github/chart-sources.yaml
          apps_folder: clusters
```

## Inputs

| Input | Default | Description |
| --- | --- | --- |
| `target_branch` | `main` | Branch the pull request targets. |
| `create_pr` | `true` | Open a pull request when updates are found. |
| `labels` | `github_actions, dependencies` | Labels to add to the pull request (must already exist in the repo). |
| `apps_folder` | `apps/manifests` | Folder (relative to the repo) to scan. |
| `file_extensions` | `yaml,yml` | Comma-separated file extensions to scan. |
| `skip_prerelease` | `true` | Skip semver prerelease versions. |
| `allow_regex_fallback` | `false` | When a manifest fails YAML parse (e.g. Helm templating), fall back to regex extraction. |
| `token` | `${{ github.token }}` | Token used to push branches and open pull requests. |
| `provider` | `auto` | Git provider: `auto`, `github`, or `gitea`/`forgejo`/`codeberg`. |
| `preset` | `argocd` | Manifest layout: `argocd` or `flux`. |
| `sources_file` | `""` | Path to a custom extraction config; overrides `preset` when set. |
| `repo_credentials` | `""` | Credentials for private chart repositories, one per line: `url-prefix\|username\|password`. Longest matching prefix wins. Works for both HTTP repos (basic auth) and OCI registries. |

## Immutable Releases

Since v1.6.0, each release ships a pre-built Go binary attached to an immutable GitHub Release. By pinning the action to a commit SHA (e.g. `ironashram/argocd-apps-action@56274b82d5397c88b2f0e84ef480b3ef71d1fe68 # v1.7.1`), there is no supply-chain risk since the referenced code and binary cannot be altered after release.

## Important Note about Pull Requests

Please ensure that you have allowed GitHub Actions to create and approve pull requests. This is necessary for the correct operation of the action.

You can enable this setting in your repository's settings under the Actions tab. If your repository is part of an organization, you might need to check the organization's settings or contact your organization's owner for help.
