package argoaction

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ironashram/argocd-apps-action/models"
	"github.com/ironashram/argocd-apps-action/utils"
)

func (u *Updater) scanChartPins(ctx context.Context, key models.ChartRef, version string, files []models.AppFile) PinScan {
	if !u.Config.CheckImagePins {
		return PinScan{}
	}

	seen := map[string]bool{}
	var pins []models.Pin
	for _, f := range files {
		for _, p := range f.Pins {
			if seen[p.Path] {
				continue
			}
			seen[p.Path] = true
			pins = append(pins, p)
		}
	}
	if len(pins) == 0 {
		return PinScan{}
	}

	defaults, ok := u.chartDefaults(ctx, key, version)
	if !ok {
		return PinScan{}
	}

	scan := scanPinsWithSource(pins, defaults.lookup)
	if len(scan.Skipped) > 0 {
		u.Action.Infof("Image pins without a %s %s default, not reported: %s",
			key.Chart, version, strings.Join(scan.Skipped, ", "))
	}
	return scan
}

func (u *Updater) chartDefaults(ctx context.Context, key models.ChartRef, version string) (*chartDefaults, bool) {
	cacheKey := key.RepoURL + "/" + key.Chart + "@" + version
	if cached, ok := u.defaults[cacheKey]; ok {
		return cached, cached != nil
	}

	defaults, err := u.fetchChartDefaults(ctx, key, version)
	if err != nil {
		u.Action.Infof("Image pins not checked for %s %s: %v", key.Chart, version, err)
		defaults = nil
	}
	if u.defaults == nil {
		u.defaults = map[string]*chartDefaults{}
	}
	u.defaults[cacheKey] = defaults
	return defaults, defaults != nil
}

func (u *Updater) fetchChartDefaults(ctx context.Context, key models.ChartRef, version string) (*chartDefaults, error) {
	cred := credFor(u.Config.RepoCreds, key.RepoURL)

	url, err := tarballURL(ctx, key.RepoURL, key.Chart, version, cred, u.Action)
	if err == nil && url != "" {
		username, password := "", ""
		if cred != nil {
			username, password = cred.Username, cred.Password
		}
		archive, ferr := utils.GetHTTPResponse(ctx, url, username, password)
		if ferr != nil {
			return nil, ferr
		}
		return readChartDefaults(archive)
	}

	archive, err := pullOCIChart(ctx, key.RepoURL, key.Chart, version, cred)
	if err != nil {
		return nil, err
	}
	return readChartDefaults(archive)
}

type chartDefaults struct {
	values             map[string]any
	appVersion         string
	depAppVersions     map[string]string
	appVersionPaths    map[string]bool
	depAppVersionPaths map[string]map[string]bool
	deps               map[string]map[string]any
}

type chartMeta struct {
	AppVersion   string `yaml:"appVersion"`
	Dependencies []struct {
		Name  string `yaml:"name"`
		Alias string `yaml:"alias"`
	} `yaml:"dependencies"`
}

// An empty value counts as the appVersion only where a template defaults it so.
func (c *chartDefaults) lookup(dotted string) (string, bool, bool) {
	if v := scalarString(getPath(c.values, dotted)); v != "" {
		return v, false, true
	}
	if c.appVersionPaths[dotted] {
		return c.appVersion, true, c.appVersion != ""
	}

	key, rest, found := strings.Cut(dotted, ".")
	if !found {
		return "", false, false
	}
	dep, ok := c.deps[key]
	if !ok {
		return "", false, false
	}
	if v := scalarString(getPath(dep, rest)); v != "" {
		return v, false, true
	}
	if c.depAppVersionPaths[key][rest] {
		return c.depAppVersions[key], true, c.depAppVersions[key] != ""
	}
	return "", false, false
}

func readChartDefaults(archive []byte) (*chartDefaults, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	out := &chartDefaults{
		deps:            map[string]map[string]any{},
		appVersionPaths: map[string]bool{},
	}
	depValues := map[string]map[string]any{}
	depMeta := map[string]chartMeta{}
	depTemplatePaths := map[string]map[string]bool{}

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		name := strings.TrimPrefix(path.Clean(hdr.Name), "./")
		parts := strings.Split(name, "/")
		if len(parts) < 2 {
			continue
		}
		root := parts[0]
		rest := strings.Join(parts[1:], "/")

		switch {
		case rest == "values.yaml":
			out.values, err = decodeTree(tr)
		case rest == "Chart.yaml":
			var meta chartMeta
			if err = decodeInto(tr, &meta); err == nil {
				depMeta[root] = meta
			}
		case strings.HasPrefix(rest, "charts/") && strings.HasSuffix(rest, "/values.yaml"):
			dep := strings.Split(strings.TrimPrefix(rest, "charts/"), "/")[0]
			depValues[dep], err = decodeTree(tr)
		case strings.HasPrefix(rest, "charts/") && strings.HasSuffix(rest, "/Chart.yaml"):
			dep := strings.Split(strings.TrimPrefix(rest, "charts/"), "/")[0]
			var meta chartMeta
			if err = decodeInto(tr, &meta); err == nil {
				depMeta["charts/"+dep] = meta
			}
		case strings.HasPrefix(rest, "charts/") && strings.Contains(rest, "/templates/"):
			dep := strings.Split(strings.TrimPrefix(rest, "charts/"), "/")[0]
			if depTemplatePaths[dep] == nil {
				depTemplatePaths[dep] = map[string]bool{}
			}
			err = collectAppVersionPaths(tr, depTemplatePaths[dep])
		case strings.HasPrefix(rest, "templates/"):
			err = collectAppVersionPaths(tr, out.appVersionPaths)
		}
		if err != nil {
			return nil, err
		}
	}

	if out.values == nil {
		return nil, fmt.Errorf("values.yaml not found in chart archive")
	}

	aliases := map[string]string{}
	for root, meta := range depMeta {
		if !strings.HasPrefix(root, "charts/") {
			out.appVersion = meta.AppVersion
		}
		for _, d := range meta.Dependencies {
			if d.Alias != "" {
				aliases[d.Name] = d.Alias
			}
		}
	}

	out.depAppVersions = map[string]string{}
	for root, meta := range depMeta {
		dep, ok := strings.CutPrefix(root, "charts/")
		if !ok {
			continue
		}
		out.depAppVersions[depKey(dep, aliases)] = meta.AppVersion
	}

	out.depAppVersionPaths = map[string]map[string]bool{}
	for dep, paths := range depTemplatePaths {
		out.depAppVersionPaths[depKey(dep, aliases)] = paths
	}
	for name, values := range depValues {
		key := name
		if alias, ok := aliases[name]; ok {
			key = alias
		}
		out.deps[key] = values
	}

	return out, nil
}

func depKey(dep string, aliases map[string]string) string {
	if alias, ok := aliases[dep]; ok {
		return alias
	}
	return dep
}

var appVersionDefaultRe = regexp.MustCompile(`\.Values\.([A-Za-z0-9_.\-]+)\s*\|\s*default\s+\.Chart\.AppVersion`)

func collectAppVersionPaths(r io.Reader, into map[string]bool) error {
	data, err := io.ReadAll(io.LimitReader(r, 4<<20))
	if err != nil {
		return err
	}
	for _, m := range appVersionDefaultRe.FindAllSubmatch(data, -1) {
		into[string(m[1])] = true
	}
	return nil
}

func decodeTree(r io.Reader) (map[string]any, error) {
	var tree map[string]any
	if err := decodeInto(r, &tree); err != nil {
		return nil, err
	}
	return tree, nil
}

func decodeInto(r io.Reader, target any) error {
	data, err := io.ReadAll(io.LimitReader(r, 8<<20))
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, target)
}
