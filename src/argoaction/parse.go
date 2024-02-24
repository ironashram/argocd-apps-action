package argoaction

import (
	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"

	"github.com/Masterminds/semver"

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

var getNewestVersion = func(targetVersion string, entries map[string][]struct {
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
