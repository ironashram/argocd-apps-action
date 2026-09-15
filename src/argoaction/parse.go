package argoaction

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"
	"github.com/ironashram/argocd-apps-action/utils"

	"github.com/Masterminds/semver/v3"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
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

// Index urls may be relative to the repository, which is how Helm resolves them.
func tarballURL(ctx context.Context, repoURL, chart, version string, cred *models.RepoCredential, action internal.ActionInterface) (string, error) {
	username, password := "", ""
	if cred != nil {
		username, password = cred.Username, cred.Password
	}
	body, err := utils.GetHTTPResponse(ctx, strings.TrimSuffix(repoURL, "/")+"/index.yaml", username, password)
	if err != nil {
		return "", err
	}

	var index models.Index
	if err := yaml.Unmarshal(body, &index); err != nil {
		return "", err
	}
	for _, entry := range index.Entries[chart] {
		if entry.Version != version || len(entry.URLs) == 0 {
			continue
		}
		u := entry.URLs[0]
		if strings.Contains(u, "://") {
			return u, nil
		}
		return strings.TrimSuffix(repoURL, "/") + "/" + strings.TrimPrefix(u, "/"), nil
	}
	action.Debugf("No tarball url for %s %s", chart, version)
	return "", nil
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

func ociRepository(url, chart string, cred *models.RepoCredential) (*remote.Repository, error) {
	repo, err := remote.NewRepository(strings.TrimSuffix(url, "/") + "/" + chart)
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
	return repo, nil
}

const helmChartLayer = "application/vnd.cncf.helm.chart.content.v1.tar+gzip"

func pullOCIChart(ctx context.Context, url, chart, version string, cred *models.RepoCredential) ([]byte, error) {
	repo, err := ociRepository(url, chart, cred)
	if err != nil {
		return nil, err
	}

	descriptor, body, err := repo.FetchReference(ctx, strings.ReplaceAll(version, "+", "_"))
	if err != nil {
		return nil, err
	}
	defer body.Close()

	raw, err := content.ReadAll(body, descriptor)
	if err != nil {
		return nil, err
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}

	for _, layer := range manifest.Layers {
		if layer.MediaType != helmChartLayer {
			continue
		}
		blob, err := repo.Fetch(ctx, layer)
		if err != nil {
			return nil, err
		}
		defer blob.Close()
		return content.ReadAll(blob, layer)
	}
	return nil, fmt.Errorf("no chart layer in %s %s", chart, version)
}
