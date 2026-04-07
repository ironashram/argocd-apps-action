package argoaction

import (
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/ironashram/argocd-apps-action/internal"
)

var targetRevisionRe = regexp.MustCompile(`(.*targetRevision: ).*`)

func (u *Updater) CheckForUpdates() error {
	dir := path.Join(u.Config.Workspace, u.Config.AppsFolder)

	var errs []error
	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			u.Action.Debugf("Error walking path: %v", err)
			errs = append(errs, err)
			return nil
		}

		if d.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if u.matchesExtension(ext) {
			osw := &internal.OSWrapper{}
			err := u.processFile(path, osw)
			if err != nil {
				u.Action.Debugf("Error processing file: %v", err)
				errs = append(errs, err)
			}
		}

		return nil
	})
	if walkErr != nil {
		errs = append(errs, walkErr)
	}

	return errors.Join(errs...)
}

func (u *Updater) matchesExtension(ext string) bool {
	return slices.Contains(u.Config.FileExtensions, ext)
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
			lines[i] = targetRevisionRe.ReplaceAllString(line, "${1}"+newest.String())
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

func (u *Updater) processFile(path string, osw internal.OSInterface) error {
	app, err := readAndParseYAML(osw, path)
	if err != nil {
		u.Action.Debugf("Error reading and parsing YAML: %v", err)
		return err
	}

	chart := app.Spec.Source.Chart
	url := app.Spec.Source.RepoURL
	targetRevision := app.Spec.Source.TargetRevision

	if chart == "" || url == "" || targetRevision == "" {
		u.Action.Debugf("Skipping invalid application manifest %s", path)
		return nil
	}

	u.Action.Debugf("Checking %s from %s, current version is %s", chart, url, targetRevision)

	newest, err := getNewestVersionFromNative(url+"/index.yaml", chart, targetRevision, u.Action, u.Config.SkipPreRelease)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported protocol scheme") {
			u.Action.Debugf("Does not look like a native chart repository, trying OCI\n")
			newest, err = getNewestVersionFromOCI(url, chart, targetRevision, u.Action, u.Config.SkipPreRelease)
			if err != nil {
				u.Action.Infof("Error getting newest version: %v", err)
				return nil
			}
		}
	}

	if newest != nil {
		u.Action.Infof("There is a newer %s version: %s", chart, newest)

		if u.Config.CreatePr {
			err = u.handleNewVersion(chart, newest, path, osw)
			if err != nil {
				return err
			}
		} else {
			u.Action.Infof("Create PR is disabled, skipping PR creation for %s", chart)
		}
	} else {
		u.Action.Debugf("No newer version of %s is available", chart)
	}
	return nil
}
