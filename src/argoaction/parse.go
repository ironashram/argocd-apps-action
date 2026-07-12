package argoaction

import (
	"context"
	"strings"

	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"
	"github.com/ironashram/argocd-apps-action/utils"

	"github.com/Masterminds/semver/v3"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"

	"sigs.k8s.io/yaml"
)

func stripScheme(u string) string {
	for _, p := range []string{"oci://", "https://", "http://"} {
		u = strings.TrimPrefix(u, p)
	}
	return u
}

func credFor(creds []models.RepoCredential, url string) *models.RepoCredential {
	target := stripScheme(url)
	var best *models.RepoCredential
	bestLen := -1
	for i, c := range creds {
		prefix := stripScheme(c.URLPrefix)
		if strings.HasPrefix(target, prefix) && len(prefix) > bestLen {
			best = &creds[i]
			bestLen = len(prefix)
		}
	}
	return best
}

func pickNewest(candidates []string, skipPreRelease bool, action internal.ActionInterface) *semver.Version {
	var newest *semver.Version
	for _, candidate := range candidates {
		v, err := semver.NewVersion(candidate)
		if err != nil {
			action.Debugf("Skipping non-semver version %q: %v", candidate, err)
			continue
		}
		if skipPreRelease && v.Prerelease() != "" {
			continue
		}
		if newest == nil || newest.LessThan(v) {
			newest = v
		}
	}
	return newest
}

func listVersionsFromNative(ctx context.Context, url string, chart string, cred *models.RepoCredential, action internal.ActionInterface) ([]string, error) {
	var index models.Index

	username, password := "", ""
	if cred != nil {
		username, password = cred.Username, cred.Password
	}
	body, err := utils.GetHTTPResponse(ctx, url, username, password)
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

	entry, ok := index.Entries[chart]
	if !ok || len(entry) == 0 {
		action.Debugf("Chart entry %s does not exist or is empty at %s", chart, url)
		return nil, nil
	}

	versions := make([]string, 0, len(entry))
	for _, v := range entry {
		versions = append(versions, v.Version)
	}
	return versions, nil
}

func listVersionsFromOCI(ctx context.Context, url string, chart string, cred *models.RepoCredential, action internal.ActionInterface) ([]string, error) {
	url = strings.TrimSuffix(url, "/") + "/" + chart
	repo, err := remote.NewRepository(url)
	if err != nil {
		return nil, err
	}

	if cred != nil {
		repo.Client = &auth.Client{
			Client: retry.DefaultClient,
			Cache:  auth.NewCache(),
			Credential: auth.StaticCredential(repo.Reference.Registry, auth.Credential{
				Username: cred.Username,
				Password: cred.Password,
			}),
		}
	}

	var versions []string
	err = repo.Tags(ctx, "", func(tagsResult []string) error {
		for _, tag := range tagsResult {
			versions = append(versions, strings.ReplaceAll(tag, "_", "+"))
		}
		return nil
	})
	if err != nil {
		action.Debugf("Error getting tags: %v", err)
		return nil, err
	}

	return versions, nil
}
