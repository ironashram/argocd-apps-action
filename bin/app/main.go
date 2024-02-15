package main

import (
	"io"

	"github.com/Masterminds/semver"
	"github.com/sethvargo/go-githubactions"

	"fmt"
	"net/http"
	"path/filepath"

	"os"

	"gopkg.in/yaml.v2"
)

type Application struct {
	Spec struct {
		Source struct {
			Chart          string `yaml:"chart"`
			RepoURL        string `yaml:"repoURL"`
			TargetRevision string `yaml:"targetRevision"`
		} `yaml:"source"`
	} `yaml:"spec"`
}

type Index struct {
	Entries map[string][]struct {
		Version string `yaml:"version"`
	} `yaml:"entries"`
}

func readAndParseYAML(path string) (Application, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Application{}, err
	}

	var app Application
	err = yaml.Unmarshal(data, &app)
	if err != nil {
		return Application{}, err
	}

	return app, nil
}

func getHTTPResponse(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func processFile(path string) error {
	app, err := readAndParseYAML(path)
	if err != nil {
		return err
	}

	chart := app.Spec.Source.Chart
	url := app.Spec.Source.RepoURL + "/index.yaml"
	targetRevision := app.Spec.Source.TargetRevision

	if chart == "" || url == "" || targetRevision == "" {
		fmt.Printf("Skipping invalid application manifest %s\n", path)
		return nil
	}

	fmt.Printf("Checking %s from %s, current version is %s\n", chart, url, targetRevision)

	body, err := getHTTPResponse(url)
	if err != nil {
		return err
	}

	var index Index
	err = yaml.Unmarshal(body, &index)
	if err != nil {
		return err
	}

	if _, ok := index.Entries[chart]; !ok || len(index.Entries[chart]) == 0 {
		fmt.Printf("Chart entry %s does not exist or is empty at %s\n", chart, url)
		return nil
	}

	newest, err := getNewestVersion(targetRevision, index.Entries)
	if err != nil {
		fmt.Printf("Error comparing versions: %v\n", err)
		return err
	}

	if newest != nil {
		fmt.Printf("There is a newer %s version: %s\n", chart, newest)
	} else {
		fmt.Printf("No newer version of %s is available\n", chart)
	}

	return nil
}

func checkForUpdates(dir string) error {
	dir = filepath.Clean(dir)

	var walkErr error
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) == ".yaml" {
			err := processFile(path)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return walkErr
}

func getNewestVersion(targetVersion string, entries map[string][]struct {
	Version string `yaml:"version"`
}) (*semver.Version, error) {
	target, err := semver.NewVersion(targetVersion)
	if err != nil {
		return nil, err
	}

	var newest *semver.Version
	for _, entry := range entries {
		for _, version := range entry {
			upstream, err := semver.NewVersion(version.Version)
			if err != nil {
				return nil, err
			}

			if target.LessThan(upstream) {
				if newest == nil || newest.LessThan(upstream) {
					newest = upstream
				}
			}
		}
	}

	return newest, nil
}

func main() {
	err := checkForUpdates("/home/m1k/Documents/github/kub1k/apps/templates")
	if err != nil {
		githubactions.Fatalf("error: %v", err)
	}
}
