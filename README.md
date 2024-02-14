## ArgoCd Apps Action
This GitHub Action checks for updates in the specified directory of YAML files. It's written in Go and uses the `gopkg.in/yaml.v2` package to parse YAML files and the `github.com/sethvargo/go-githubactions` package for GitHub Actions specific functionalities.

## How it works

The action walks through the specified directory and its subdirectories, looking for YAML files. For each YAML file, it reads the file and unmarshals the content into an `Application` manifest.

The action then checks if the `chart`, `url`, and `targetRevision` fields are present. If they are, it sends a GET request to the URL (`RepoURL`) and unmarshals the response getting the new chart verions.

The action then checks if there is a new release and opens PR with the updates if there is.

## Usage

To use this action in your GitHub workflow you can do as follows:

```

```
