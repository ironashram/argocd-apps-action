package argoaction

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"
	"github.com/ironashram/argocd-apps-action/utils"

	"gopkg.in/yaml.v2"
)

var checkForUpdates = func(gitOps internal.GitOperations, githubClient internal.GitHubClient, cfg *models.Config, action internal.ActionInterface) error {
	dir := path.Join(cfg.Workspace, cfg.AppsFolder)

	var walkErr error
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) == ".yaml" {
			osWrapper := &internal.OSWrapper{}
			err := processFile(path, gitOps, githubClient, cfg, action, osWrapper)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return walkErr
}

var processFile = func(path string, gitOps internal.GitOperations, githubClient internal.GitHubClient, cfg *models.Config, action internal.ActionInterface, osw internal.OSInterface) error {
	app, err := readAndParseYAML(osw, path)
	if err != nil {
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
		return err
	}

	var index models.Index
	err = yaml.Unmarshal(body, &index)
	if err != nil {
		return fmt.Errorf("failed to unmarshal YAML body: %w", err)
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
			branchName := "update-" + chart
			err = createNewBranch(gitOps, branchName)
			if err != nil {
				return err
			}

			app.Spec.Source.TargetRevision = newest.String()
			newData, err := yaml.Marshal(app)
			if err != nil {
				return err
			}

			err = os.WriteFile(path, newData, 0644)
			if err != nil {
				return err
			}

			commitMessage := "Update " + chart + " to version " + newest.String()
			err = commitChanges(gitOps, path, commitMessage)
			if err != nil {
				return err
			}

			err = pushChanges(gitOps, branchName, cfg)
			if err != nil {
				return err
			}

			prTitle := "Update " + chart + " to version " + newest.String()
			prBody := "This PR updates " + chart + " to version " + newest.String()
			err = createPullRequest(githubClient, cfg.TargetBranch, branchName, prTitle, prBody, action)
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
