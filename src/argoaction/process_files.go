package argoaction

import (
	"context"
	"errors"
	"path"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"
)

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

func (u *Updater) processChartGroup(ctx context.Context, key models.ChartRef, files []models.AppFile, osw internal.OSInterface) error {
	u.Action.Debugf("Checking %s from %s (%d files)", key.Chart, key.RepoURL, len(files))

	cred := credFor(u.Config.RepoCreds, key.RepoURL)
	versions, err := listVersionsFromNative(ctx, key.RepoURL+"/index.yaml", key.Chart, cred, u.Action)
	if err != nil && !strings.Contains(err.Error(), "unsupported protocol scheme") {
		u.Action.Infof("Error getting versions for %s: %v", key.Chart, err)
		return nil
	}
	if err != nil {
		u.Action.Debugf("Not a native chart repository, trying OCI for %s", key.Chart)
		versions, err = listVersionsFromOCI(ctx, key.RepoURL, key.Chart, cred, u.Action)
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
		current, err := semver.StrictNewVersion(strings.TrimPrefix(f.CurrentVersion, "v"))
		if err != nil {
			u.Action.Infof("Skipping %s: current version %q is not a fixed semver version", f.Path, f.CurrentVersion)
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
