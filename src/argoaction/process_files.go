package argoaction

import (
	"context"
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

func (u *Updater) CheckForUpdates(ctx context.Context) error {
	dir := path.Join(u.Config.Workspace, u.Config.AppsFolder)

	osw := &internal.OSWrapper{}
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
			err := u.processFile(ctx, path, osw)
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

func (u *Updater) processFile(ctx context.Context, path string, osw internal.OSInterface) error {
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

	newest, err := getNewestVersionFromNative(ctx, url+"/index.yaml", chart, targetRevision, u.Action, u.Config.SkipPreRelease)
	if err != nil && !strings.Contains(err.Error(), "unsupported protocol scheme") {
		u.Action.Infof("Error getting newest version for %s: %v", chart, err)
		return nil
	}
	if err != nil {
		u.Action.Debugf("Not a native chart repository, trying OCI for %s", chart)
		newest, err = getNewestVersionFromOCI(ctx, url, chart, targetRevision, u.Action, u.Config.SkipPreRelease)
		if err != nil {
			u.Action.Infof("Error getting newest version for %s: %v", chart, err)
			return nil
		}
	}

	if newest != nil {
		u.Action.Infof("There is a newer %s version: %s", chart, newest)

		if u.Config.CreatePr {
			err = u.handleNewVersion(ctx, chart, newest, path, osw)
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
