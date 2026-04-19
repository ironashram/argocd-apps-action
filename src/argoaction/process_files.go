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
	"github.com/ironashram/argocd-apps-action/models"
)

var targetRevisionRe = regexp.MustCompile(`(.*targetRevision: ).*`)

func (u *Updater) CheckForUpdates(ctx context.Context) error {
	dir := path.Join(u.Config.Workspace, u.Config.AppsFolder)

	osw := &internal.OSWrapper{}
	var errs []error

	candidates, walkErrs := u.collectCandidates(dir, osw)
	errs = append(errs, walkErrs...)

	for key, files := range candidates {
		if err := u.processChartGroup(ctx, key, files, osw); err != nil {
			u.Action.Debugf("Error processing chart group %s (%s): %v", key.Chart, key.RepoURL, err)
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (u *Updater) collectCandidates(dir string, osw internal.OSInterface) (map[models.ChartRef][]models.AppFile, []error) {
	candidates := map[models.ChartRef][]models.AppFile{}
	var errs []error

	walkErr := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			u.Action.Debugf("Error walking path: %v", err)
			errs = append(errs, err)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !u.matchesExtension(filepath.Ext(p)) {
			return nil
		}

		app, err := readAndParseYAML(osw, p, u.Config.AllowRegexFallback, u.Action)
		if err != nil {
			u.Action.Debugf("Error reading and parsing YAML %s: %v", p, err)
			errs = append(errs, err)
			return nil
		}

		chart := app.Spec.Source.Chart
		url := app.Spec.Source.RepoURL
		rev := app.Spec.Source.TargetRevision
		if chart == "" || url == "" || rev == "" {
			u.Action.Debugf("Skipping invalid application manifest %s", p)
			return nil
		}

		key := models.ChartRef{RepoURL: url, Chart: chart}
		candidates[key] = append(candidates[key], models.AppFile{Path: p, CurrentVersion: rev})
		return nil
	})
	if walkErr != nil {
		errs = append(errs, walkErr)
	}

	return candidates, errs
}

func (u *Updater) processChartGroup(ctx context.Context, key models.ChartRef, files []models.AppFile, osw internal.OSInterface) error {
	u.Action.Debugf("Checking %s from %s (%d files)", key.Chart, key.RepoURL, len(files))

	versions, err := listVersionsFromNative(ctx, key.RepoURL+"/index.yaml", key.Chart, u.Action)
	if err != nil && !strings.Contains(err.Error(), "unsupported protocol scheme") {
		u.Action.Infof("Error getting versions for %s: %v", key.Chart, err)
		return nil
	}
	if err != nil {
		u.Action.Debugf("Not a native chart repository, trying OCI for %s", key.Chart)
		versions, err = listVersionsFromOCI(ctx, key.RepoURL, key.Chart, u.Action)
		if err != nil {
			u.Action.Infof("Error getting versions for %s: %v", key.Chart, err)
			return nil
		}
	}

	newest := pickNewest(versions, u.Config.SkipPreRelease, u.Action)
	if newest == nil {
		u.Action.Debugf("No newer version of %s is available", key.Chart)
		return nil
	}

	var toBump []models.AppFile
	for _, f := range files {
		current, err := semver.NewVersion(f.CurrentVersion)
		if err != nil {
			u.Action.Infof("Skipping %s: non-semver current version %q", f.Path, f.CurrentVersion)
			continue
		}
		if current.LessThan(newest) {
			toBump = append(toBump, f)
		}
	}

	if len(toBump) == 0 {
		u.Action.Debugf("No files need bumping for %s", key.Chart)
		return nil
	}

	u.Action.Infof("There is a newer %s version: %s (%d file(s) to update)", key.Chart, newest, len(toBump))

	if !u.Config.CreatePr {
		u.Action.Infof("Create PR is disabled, skipping PR creation for %s", key.Chart)
		return nil
	}

	return u.handleChartGroup(ctx, key.Chart, newest, toBump, osw)
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
