package argoaction

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"
)

var checkForUpdates = func(gitOps internal.GitOperations, githubClient internal.GitHubClient, cfg *models.Config, action internal.ActionInterface) error {
	dir := path.Join(cfg.Workspace, cfg.AppsFolder)

	var walkErr error
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			action.Debugf("Error walking path: %v", err)
			walkErr = err
			return nil
		}

		if filepath.Ext(path) == ".yaml" {
			osw := &internal.OSWrapper{}
			err := processFile(path, gitOps, githubClient, cfg, action, osw)
			if err != nil {
				action.Debugf("Error processing file: %v", err)
				walkErr = err
			}
		}

		return nil
	})

	return walkErr
}

func updateTargetRevision(newest *semver.Version, path string, action internal.ActionInterface, osw internal.OSInterface) error {
	oldData, err := osw.ReadFile(path)
	if err != nil {
		action.Debugf("Error reading file: %v", err)
		return err
	}

	lines := strings.Split(string(oldData), "\n")

	for i, line := range lines {
		if strings.Contains(line, "targetRevision:") {
			re := regexp.MustCompile(`(.*targetRevision: ).*`)
			lines[i] = re.ReplaceAllString(line, "${1}"+newest.String())
			break
		}
	}

	newData := strings.Join(lines, "\n")

	err = osw.WriteFile(path, []byte(newData), 0644)
	if err != nil {
		action.Debugf("Error writing file: %v", err)
		return err
	}

	return nil
}

var processFile = func(path string, gitOps internal.GitOperations, githubClient internal.GitHubClient, cfg *models.Config, action internal.ActionInterface, osw internal.OSInterface) error {
	app, err := readAndParseYAML(osw, path)
	if err != nil {
		action.Debugf("Error reading and parsing YAML: %v", err)
		return err
	}

	chart := app.Spec.Source.Chart
	url := app.Spec.Source.RepoURL
	targetRevision := app.Spec.Source.TargetRevision

	if chart == "" || url == "" || targetRevision == "" {
		action.Debugf("Skipping invalid application manifest %s", path)
		return nil
	}

	action.Debugf("Checking %s from %s, current version is %s", chart, url, targetRevision)

	newest, err := getNewestVersionFromNative(url+"/index.yaml", chart, targetRevision, action, cfg.SkipPreRelease)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported protocol scheme") {
			action.Debugf("Does not look like a native chart repository, trying OCI\n")
			newest, err = getNewestVersionFromOCI(url, chart, targetRevision, action, cfg.SkipPreRelease)
			if err != nil {
				action.Infof("Error getting newest version: %v", err)
				return nil
			}
		}
	}

	if newest != nil {
		action.Infof("There is a newer %s version: %s", chart, newest)

		if cfg.CreatePr {
			err = handleNewVersion(chart, newest, path, gitOps, cfg, action, osw, githubClient)
			if err != nil {
				return err
			}
		} else {
			action.Infof("Create PR is disabled, skipping PR creation for %s", chart)
		}
	} else {
		action.Debugf("No newer version of %s is available", chart)
	}
	return nil
}
