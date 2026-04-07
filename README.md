[![Go unit tests](https://github.com/ironashram/argocd-apps-action/actions/workflows/go-unit-tests.yaml/badge.svg)](https://github.com/ironashram/argocd-apps-action/actions/workflows/go-unit-tests.yaml)
[![go: build binaries](https://github.com/ironashram/argocd-apps-action/actions/workflows/go-binary.yaml/badge.svg)](https://github.com/ironashram/argocd-apps-action/actions/workflows/go-binary.yaml)

## ArgoCD Apps Action
This GitHub Action checks for updates in the specified directory of YAML files. It's written in Go and uses the `"sigs.k8s.io/yaml"` package to parse YAML files and the `github.com/sethvargo/go-githubactions` package for GitHub Actions specific functionalities.

## How it works

The action walks through the specified directory and its subdirectories, looking for files matching the configured extensions (default: `yaml`, `yml`). For each YAML file, it reads the file and unmarshals the content into an `Application` manifest.

The action then checks if the `chart`, `url`, and `targetRevision` fields are present. If they are, it sends a GET request to the URL (`RepoURL`) and unmarshals the response getting the new chart versions.

The action supports both Helm chart repositories and OCI registries. For OCI registries, it uses the `oras.land/oras-go` package to interact with the registry. Please note that currently only public repositories are supported.


The action then checks if there is a new release and opens PR with the updates if there is.

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
        uses: ironashram/argocd-apps-action@v1.8.0
        with:
          skip_prerelease: true
          target_branch: main
          create_pr: true
          apps_folder: apps/manifests
          file_extensions: yaml,yml
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Immutable Releases

Since v1.6.0, each release ships a pre-built Go binary attached to an immutable GitHub Release. By pinning the action to a commit SHA (e.g. `ironashram/argocd-apps-action@56274b82d5397c88b2f0e84ef480b3ef71d1fe68 # v1.7.1`), there is no supply-chain risk since the referenced code and binary cannot be altered after release.

## Important Note about Pull Requests

Please ensure that you have allowed GitHub Actions to create and approve pull requests. This is necessary for the correct operation of the action.

You can enable this setting in your repository's settings under the Actions tab. If your repository is part of an organization, you might need to check the organization's settings or contact your organization's owner for help.
