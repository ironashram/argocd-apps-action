package main

import (
	"io"

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

func checkForUpdates(dir string) error {
	var walkErr error
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) == ".yaml" {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			var app Application
			err = yaml.Unmarshal(data, &app)
			if err != nil {
				return err
			}

			chart := app.Spec.Source.Chart
			url := app.Spec.Source.RepoURL + "/index.yaml"
			targetRevision := app.Spec.Source.TargetRevision

			if chart == "" || url == "" || targetRevision == "" {
				fmt.Printf("Missing required fields in the application manifest %s\n", path)
				return nil
			}

			fmt.Printf("Checking %s from %s, current version is %s\n", chart, url, targetRevision)

			resp, err := http.Get(url)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
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
			fmt.Println(index.Entries[chart])
		}

		return nil
	})

	return walkErr
}

func main() {
	err := checkForUpdates("/home/m1k/Documents/github/kub1k/apps/templates")
	if err != nil {
		githubactions.Fatalf("error: %v", err)
	}
}
