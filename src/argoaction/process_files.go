package argoaction

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"
	"github.com/ironashram/argocd-apps-action/utils"

	"sigs.k8s.io/yaml"
)

var checkForUpdates = func(gitOps internal.GitOperations, githubClient internal.GitHubClient, cfg *models.Config, action internal.ActionInterface) error {
	dir := path.Join(cfg.Workspace, cfg.AppsFolder)

	var walkErr error
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			action.Debugf("Error walking path: %v\n", err)
			walkErr = err
			return nil
		}

		if filepath.Ext(path) == ".yaml" {
			osw := &internal.OSWrapper{}
			err := processFile(path, gitOps, githubClient, cfg, action, osw)
			if err != nil {
				action.Debugf("Error processing file: %v\n", err)
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
		action.Debugf("Error reading file: %v\n", err)
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
		action.Debugf("Error writing file: %v\n", err)
		return err
	}

	return nil
}

var processFile = func(path string, gitOps internal.GitOperations, githubClient internal.GitHubClient, cfg *models.Config, action internal.ActionInterface, osw internal.OSInterface) error {
	app, err := readAndParseYAML(osw, path)
	if err != nil {
		action.Debugf("Error reading and parsing YAML: %v\n", err)
		return err
	}

	chart := app.Spec.Source.Chart
	url := app.Spec.Source.RepoURL + "/index.yaml"
	targetRevision := app.Spec.Source.TargetRevision

	if chart == "" || url == "" || targetRevision == "" {
		action.Debugf("Skipping invalid application manifest %s\n", path)
		return nil
	}

	action.Debugf("Checking %s from %s, current version is %s\n", chart, url, targetRevision)

	body, err := utils.GetHTTPResponse(url)
	if err != nil {
		action.Debugf("failed to get HTTP response: %v\n", err)
		return err
	}

	var index models.Index
	err = yaml.Unmarshal(body, &index)
	if err != nil {
		action.Debugf("failed to unmarshal YAML body: %v\n", err)
		return err
	}

	if index.Entries == nil {
		action.Debugf("No entries found in index at %s\n", url)
		return nil
	}

	if _, ok := index.Entries[chart]; !ok || len(index.Entries[chart]) == 0 {
		action.Debugf("Chart entry %s does not exist or is empty at %s\n", chart, url)
		return nil
	}

	newest, err := getNewestVersion(targetRevision, index.Entries)
	if err != nil {
		action.Debugf("Error comparing versions: %v\n", err)
		return err
	}

	if newest != nil {
		action.Infof("There is a newer %s version: %s\n", chart, newest)

		if cfg.CreatePr {
			err = handleNewVersion(chart, newest, path, gitOps, cfg, action, osw, githubClient)
			if err != nil {
				return err
			}
		} else {
			action.Infof("Create PR is disabled, skipping PR creation for %s\n", chart)
		}
	} else {
		action.Debugf("No newer version of %s is available\n", chart)
	}
	return nil
}
