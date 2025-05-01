package argoaction

import (
	"context"
	"strings"

	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"
	"github.com/ironashram/argocd-apps-action/utils"

	"github.com/Masterminds/semver/v3"
	"oras.land/oras-go/v2/registry/remote"

	"sigs.k8s.io/yaml"
)

var readAndParseYAML = func(osi internal.OSInterface, path string) (*models.Application, error) {
	data, err := osi.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var app models.Application
	err = yaml.Unmarshal(data, &app)
	if err != nil {
		return nil, err
	}

	return &app, nil
}

var parseNativeNewest = func(targetVersion string, versions []struct {
	Version string `yaml:"version"`
}, skipPreRelease bool) (*semver.Version, error) {
	target, err := semver.NewVersion(targetVersion)
	if err != nil {
		return nil, err
	}

	var newest *semver.Version
	for _, version := range versions {
		upstream, err := semver.NewVersion(version.Version)
		if err != nil {
			return nil, err
		}

		if skipPreRelease && upstream.Prerelease() != "" {
			continue
		}

		if target.LessThan(upstream) {
			if newest == nil || newest.LessThan(upstream) {
				newest = upstream
			}
		}
	}

	return newest, nil
}

func getNewestVersionFromNative(url string, chart string, targetRevision string, action internal.ActionInterface, skipPreRelease bool) (*semver.Version, error) {
	var index models.Index

	body, err := utils.GetHTTPResponse(url)
	if err != nil {
		action.Debugf("failed to get HTTP response: %v", err)
		return nil, err
	}

	err = yaml.Unmarshal(body, &index)
	if err != nil {
		action.Debugf("failed to unmarshal YAML body: %v", err)
		return nil, err
	}

	if index.Entries == nil {
		action.Debugf("No entries found in index at %s", url)
		return nil, nil
	}

	if _, ok := index.Entries[chart]; !ok || len(index.Entries[chart]) == 0 {
		action.Debugf("Chart entry %s does not exist or is empty at %s", chart, url)
		return nil, nil
	}

	newest, err := parseNativeNewest(targetRevision, index.Entries[chart], skipPreRelease)
	if err != nil {
		action.Debugf("Error comparing versions: %v", err)
		return nil, err
	}

	return newest, nil
}

var parseOCINewest = func(tags *models.TagsList, targetVersion string, action internal.ActionInterface, skipPreRelease bool) (*semver.Version, error) {
	target, err := semver.NewVersion(targetVersion)
	if err != nil {
		action.Debugf("Error parsing target version: %v", err)
		return nil, err
	}

	var newest *semver.Version
	for _, tag := range tags.Tags {
		upstream, err := semver.NewVersion(tag)
		if err != nil {
			action.Debugf("Error parsing tag version: %v", err)
			return nil, err
		}

		if skipPreRelease && upstream.Prerelease() != "" {
			continue
		}

		if target.LessThan(upstream) {
			if newest == nil || newest.LessThan(upstream) {
				newest = upstream
			}
		}
	}

	return newest, nil
}

func getNewestVersionFromOCI(url string, chart string, targetRevision string, action internal.ActionInterface, skipPreRelease bool) (*semver.Version, error) {
	tags := &models.TagsList{}

	url = strings.TrimSuffix(url, "/")
	url = strings.Replace(url, "https://", "", 1)
	url = strings.Replace(url, "http://", "", 1)
	url = strings.Replace(url, "oci://", "", 1)
	url = url + "/" + chart
	repo, err := remote.NewRepository(url)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	err = repo.Tags(ctx, "", func(tagsResult []string) error {
		for _, tag := range tagsResult {
			convertedTag := strings.ReplaceAll(tag, "_", "+")
			tags.Tags = append(tags.Tags, convertedTag)
		}

		return err
	})

	newest, err := parseOCINewest(tags, targetRevision, action, skipPreRelease)
	if err != nil {
		action.Debugf("Error comparing versions: %v", err)
		return nil, err
	}

	if err != nil {
		action.Debugf("Error getting tags: %v", err)
		return nil, err
	}

	return newest, nil

}
