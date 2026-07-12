package argoaction

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"

	"gopkg.in/yaml.v3"
)

var multiSourceRe = regexp.MustCompile(`(?m)^\s+sources:\s*$`)

func argocdPreset(regexFallback bool) *models.SourcesConfig {
	return &models.SourcesConfig{
		Charts: []models.ChartRule{{
			Files:         []string{"*"},
			ChartPath:     "spec.source.chart",
			VersionPath:   "spec.source.targetRevision",
			URLPath:       "spec.source.repoURL",
			RegexFallback: regexFallback,
		}},
	}
}

func fluxPreset() *models.SourcesConfig {
	return &models.SourcesConfig{
		Repositories: []models.RepoRule{{
			Files:         []string{"*"},
			NamePath:      "metadata.name",
			NamespacePath: "metadata.namespace",
			URLPath:       "spec.url",
			SkipIfSet:     "spec.secretRef",
		}},
		Charts: []models.ChartRule{
			{
				Files:       []string{"*"},
				ChartPath:   "spec.chart.spec.chart",
				VersionPath: "spec.chart.spec.version",
				RepoRef: &models.RepoRef{
					NamePath:      "spec.chart.spec.sourceRef.name",
					NamespacePath: "spec.chart.spec.sourceRef.namespace",
				},
			},
			{
				Files:       []string{"*"},
				URLPath:     "spec.url",
				VersionPath: "spec.ref.semver",
			},
		},
	}
}

func SourcesFor(cfg *models.Config, osi internal.OSInterface) (*models.SourcesConfig, error) {
	if cfg.SourcesFile != "" {
		data, err := osi.ReadFile(filepath.Join(cfg.Workspace, cfg.SourcesFile))
		if err != nil {
			return nil, err
		}
		var sc models.SourcesConfig
		if err := yaml.Unmarshal(data, &sc); err != nil {
			return nil, err
		}
		return &sc, nil
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Preset)) {
	case "flux":
		return fluxPreset(), nil
	case "argocd", "":
		return argocdPreset(cfg.AllowRegexFallback), nil
	default:
		return nil, fmt.Errorf("unknown preset: %s", cfg.Preset)
	}
}

type parsedFile struct {
	path    string
	raw     []byte
	docs    []map[string]any
	decErr  error
}

func (u *Updater) collectCandidates(dir string, osw internal.OSInterface) (map[models.ChartRef][]models.AppFile, []error) {
	sc := u.Sources
	if sc == nil {
		sc = argocdPreset(u.Config.AllowRegexFallback)
	}

	candidates := map[models.ChartRef][]models.AppFile{}
	var errs []error
	var files []parsedFile

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
		data, rerr := osw.ReadFile(p)
		if rerr != nil {
			u.Action.Debugf("Error reading %s: %v", p, rerr)
			errs = append(errs, rerr)
			return nil
		}
		docs, derr := decodeDocs(data)
		files = append(files, parsedFile{path: p, raw: data, docs: docs, decErr: derr})
		return nil
	})
	if walkErr != nil {
		errs = append(errs, walkErr)
	}

	index := map[string]string{}
	for _, f := range files {
		for _, r := range sc.Repositories {
			if !matchFiles(r.Files, f.path) {
				continue
			}
			for _, doc := range f.docs {
				name := getString(doc, r.NamePath)
				url := getString(doc, r.URLPath)
				if name == "" || url == "" {
					continue
				}
				if r.SkipIfSet != "" && hasPath(doc, r.SkipIfSet) && credFor(u.Config.RepoCreds, url) == nil {
					continue
				}
				ns := ""
				if r.NamespacePath != "" {
					ns = getString(doc, r.NamespacePath)
				}
				index[ns+"/"+name] = url
			}
		}
	}

	for _, f := range files {
		matched := false

		if f.decErr != nil {
			for _, c := range sc.Charts {
				if !matchFiles(c.Files, f.path) || !c.RegexFallback {
					continue
				}
				ref, af, ok := regexExtract(f.raw, c, u.Action, f.path)
				if ok {
					candidates[ref] = append(candidates[ref], af)
					matched = true
				}
			}
			if !matched {
				u.Action.Debugf("Error reading and parsing YAML %s: %v", f.path, f.decErr)
				errs = append(errs, f.decErr)
			}
			continue
		}

		for di, doc := range f.docs {
			for _, c := range sc.Charts {
				if !matchFiles(c.Files, f.path) {
					continue
				}
				ref, ver, ok := extractChart(doc, c, index)
				if !ok {
					continue
				}
				candidates[ref] = append(candidates[ref], models.AppFile{
					Path:           f.path,
					CurrentVersion: ver,
					VersionPath:    c.VersionPath,
					DocIndex:       di,
				})
				matched = true
			}
		}
		if !matched {
			u.Action.Debugf("Skipping invalid application manifest %s", f.path)
		}
	}

	return candidates, errs
}

func extractChart(doc map[string]any, c models.ChartRule, index map[string]string) (models.ChartRef, string, bool) {
	version := getString(doc, c.VersionPath)
	if version == "" {
		return models.ChartRef{}, "", false
	}

	var chart, repoURL string
	switch {
	case c.ChartPath != "":
		chart = getString(doc, c.ChartPath)
		if chart == "" {
			return models.ChartRef{}, "", false
		}
		switch {
		case c.URLPath != "":
			repoURL = getString(doc, c.URLPath)
		case c.RepoRef != nil:
			name := getString(doc, c.RepoRef.NamePath)
			if name == "" {
				return models.ChartRef{}, "", false
			}
			ns := ""
			if c.RepoRef.NamespacePath != "" {
				ns = getString(doc, c.RepoRef.NamespacePath)
			}
			if ns == "" {
				ns = getString(doc, "metadata.namespace")
			}
			repoURL = index[ns+"/"+name]
		}
		if repoURL == "" {
			return models.ChartRef{}, "", false
		}
		repoURL = stripOCI(repoURL)
	case c.URLPath != "":
		u := stripOCI(getString(doc, c.URLPath))
		if u == "" {
			return models.ChartRef{}, "", false
		}
		chart = path.Base(u)
		repoURL = path.Dir(u)
	default:
		return models.ChartRef{}, "", false
	}

	return models.ChartRef{RepoURL: repoURL, Chart: chart}, version, true
}

func regexExtract(data []byte, c models.ChartRule, action internal.ActionInterface, p string) (models.ChartRef, models.AppFile, bool) {
	if c.ChartPath == "" || c.URLPath == "" || c.VersionPath == "" {
		return models.ChartRef{}, models.AppFile{}, false
	}
	if multiSourceRe.Match(data) {
		action.Infof("File %s contains a multi-source spec; regex fallback will only extract the first source", p)
	}
	chart := reField(data, leafKey(c.ChartPath))
	url := reField(data, leafKey(c.URLPath))
	ver := reField(data, leafKey(c.VersionPath))
	if chart == "" || url == "" || ver == "" {
		return models.ChartRef{}, models.AppFile{}, false
	}
	return models.ChartRef{RepoURL: stripOCI(url), Chart: chart},
		models.AppFile{Path: p, CurrentVersion: ver, VersionPath: c.VersionPath},
		true
}

func reField(data []byte, leaf string) string {
	re := regexp.MustCompile(`(?m)^\s+` + regexp.QuoteMeta(leaf) + `:\s*(\S.*?)\s*$`)
	if m := re.FindSubmatch(data); len(m) == 2 {
		return strings.Trim(string(m[1]), `"'`)
	}
	return ""
}

func decodeDocs(data []byte) ([]map[string]any, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var docs []map[string]any
	for {
		var d map[string]any
		err := dec.Decode(&d)
		if err == io.EOF {
			break
		}
		if err != nil {
			return docs, err
		}
		if d != nil {
			docs = append(docs, d)
		}
	}
	return docs, nil
}

func matchFiles(patterns []string, p string) bool {
	if len(patterns) == 0 {
		return true
	}
	base := filepath.Base(p)
	for _, pat := range patterns {
		if pat == "*" || pat == "" {
			return true
		}
		if ok, _ := filepath.Match(pat, base); ok {
			return true
		}
	}
	return false
}

func getPath(m map[string]any, p string) any {
	cur := any(m)
	for _, part := range strings.Split(p, ".") {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = mm[part]
	}
	return cur
}

func getString(m map[string]any, p string) string {
	if p == "" {
		return ""
	}
	switch v := getPath(m, p).(type) {
	case string:
		return v
	case nil:
		return ""
	case map[string]any, []any:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

func hasPath(m map[string]any, p string) bool {
	if p == "" {
		return false
	}
	cur := any(m)
	parts := strings.Split(p, ".")
	for i, part := range parts {
		mm, ok := cur.(map[string]any)
		if !ok {
			return false
		}
		v, ok := mm[part]
		if !ok {
			return false
		}
		if i == len(parts)-1 {
			return true
		}
		cur = v
	}
	return true
}

func stripOCI(u string) string {
	return strings.TrimPrefix(u, "oci://")
}

func leafKey(p string) string {
	if i := strings.LastIndex(p, "."); i >= 0 {
		return p[i+1:]
	}
	return p
}

func (u *Updater) updateVersion(f models.AppFile, newest *semver.Version, osw internal.OSInterface) error {
	data, err := osw.ReadFile(f.Path)
	if err != nil {
		u.Action.Debugf("Error reading file: %v", err)
		return err
	}
	out := writeVersion(data, f, newest.String())
	if err := osw.WriteFile(f.Path, out, 0644); err != nil {
		u.Action.Debugf("Error writing file: %v", err)
		return err
	}
	return nil
}

func writeVersion(data []byte, f models.AppFile, newest string) []byte {
	if out, ok := replaceVersionAtPath(data, f.DocIndex, f.VersionPath, f.CurrentVersion, newest); ok {
		return out
	}
	return leafLineReplace(data, leafKey(f.VersionPath), newest)
}

func replaceVersionAtPath(data []byte, docIndex int, p, oldValue, newest string) ([]byte, bool) {
	if oldValue == "" {
		return nil, false
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	idx := 0
	for {
		var node yaml.Node
		if err := dec.Decode(&node); err != nil {
			return nil, false
		}
		if idx == docIndex {
			target := nodeAtPath(&node, strings.Split(p, "."))
			if target == nil || target.Kind != yaml.ScalarNode {
				return nil, false
			}
			lineIdx := target.Line - 1
			lines := strings.Split(string(data), "\n")
			if lineIdx < 0 || lineIdx >= len(lines) {
				return nil, false
			}
			if !strings.Contains(lines[lineIdx], oldValue) {
				return nil, false
			}
			lines[lineIdx] = strings.Replace(lines[lineIdx], oldValue, newest, 1)
			return []byte(strings.Join(lines, "\n")), true
		}
		idx++
	}
}

func nodeAtPath(n *yaml.Node, parts []string) *yaml.Node {
	if n.Kind == yaml.DocumentNode {
		if len(n.Content) == 0 {
			return nil
		}
		n = n.Content[0]
	}
	cur := n
	for _, part := range parts {
		if cur.Kind != yaml.MappingNode {
			return nil
		}
		var next *yaml.Node
		for i := 0; i+1 < len(cur.Content); i += 2 {
			if cur.Content[i].Value == part {
				next = cur.Content[i+1]
				break
			}
		}
		if next == nil {
			return nil
		}
		cur = next
	}
	return cur
}

func leafLineReplace(data []byte, leaf, newest string) []byte {
	re := regexp.MustCompile(`(.*` + regexp.QuoteMeta(leaf) + `: ).*`)
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.Contains(line, leaf+":") {
			lines[i] = re.ReplaceAllString(line, "${1}"+newest)
			break
		}
	}
	return []byte(strings.Join(lines, "\n"))
}
