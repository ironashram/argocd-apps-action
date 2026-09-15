package argoaction

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ironashram/argocd-apps-action/models"
)

type PinState string

const (
	PinAhead     PinState = "AHEAD"
	PinRedundant PinState = "REDUNDANT"
	PinBehind    PinState = "BEHIND"
	PinDiffers   PinState = "DIFFERS"
)

type PinReport struct {
	Pin        models.Pin
	Default    string
	AppVersion bool
	State      PinState
}

type PinScan struct {
	Reports []PinReport
	Skipped []string
}

func scanPins(pins []models.Pin, lookup func(path string) (string, bool)) PinScan {
	return scanPinsWithSource(pins, func(p string) (string, bool, bool) {
		v, ok := lookup(p)
		return v, false, ok
	})
}

func scanPinsWithSource(pins []models.Pin, lookup func(path string) (string, bool, bool)) PinScan {
	var scan PinScan
	for _, pin := range pins {
		if pin.Opaque {
			scan.Skipped = append(scan.Skipped, pin.Path)
			continue
		}
		raw, appVersion, ok := lookup(pin.Path)
		if !ok || raw == "" {
			scan.Skipped = append(scan.Skipped, pin.Path)
			continue
		}
		chartDefault := raw
		if pin.Ref {
			_, tag, split := splitImageRef(raw)
			if !split {
				scan.Skipped = append(scan.Skipped, pin.Path)
				continue
			}
			chartDefault = tag
		}
		report := comparePin(pin, chartDefault)
		report.AppVersion = appVersion
		scan.Reports = append(scan.Reports, report)
	}
	return scan
}

var tagCoreRe = regexp.MustCompile(`^v?\d+(\.\d+)*`)

func (s PinScan) counts() (behind, redundant int) {
	for _, r := range s.Reports {
		switch r.State {
		case PinBehind:
			behind++
		case PinRedundant:
			redundant++
		}
	}
	return behind, redundant
}

func (s PinScan) titleSuffix() string {
	behind, redundant := s.counts()
	var parts []string
	if behind > 0 {
		parts = append(parts, fmt.Sprintf("%d pin%s behind", behind, plural(behind)))
	}
	if redundant > 0 {
		parts = append(parts, fmt.Sprintf("%d redundant", redundant))
	}
	if len(parts) == 0 {
		return ""
	}
	return " [" + strings.Join(parts, ", ") + "]"
}

func (s PinScan) bodySection(chart, version string) string {
	if len(s.Reports) == 0 {
		return ""
	}

	width := 0
	for _, r := range s.Reports {
		if len(r.State) > width {
			width = len(r.State)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "\nImage pins checked against %s %s defaults:\n", chart, version)
	for _, r := range s.Reports {
		source := "chart default"
		if r.AppVersion {
			source = "chart appVersion"
		}
		detail := fmt.Sprintf("(%s %s)", source, r.Default)
		if r.State == PinRedundant {
			detail = "(matches " + source + ")"
		}
		fmt.Fprintf(&b, "- %-*s %s = %s %s\n", width, r.State, r.Pin.Path, r.Pin.Value, detail)
	}
	return b.String()
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// A custom sources file says where the chart is, not where values are.
func valuesPathsFor(rule models.ChartRule) models.ChartRule {
	if rule.ValuesPath != "" || rule.ValuesStringPath != "" || rule.ParametersPath != "" || rule.FileParametersPath != "" {
		return rule
	}
	rule.ValuesPath = "spec.values"
	rule.ValuesStringPath = "spec.source.helm.values"
	rule.ParametersPath = "spec.source.helm.parameters"
	rule.FileParametersPath = "spec.source.helm.fileParameters"
	return rule
}

func collectPins(doc map[string]any, rule models.ChartRule) []models.Pin {
	rule = valuesPathsFor(rule)
	var pins []models.Pin

	for _, p := range []string{rule.ValuesPath, "spec.source.helm.valuesObject"} {
		if p == "" {
			continue
		}
		if tree, ok := getPath(doc, p).(map[string]any); ok {
			pins = append(pins, walkPins(tree, "")...)
		}
	}

	if raw := getString(doc, rule.ValuesStringPath); raw != "" {
		var tree map[string]any
		if err := yaml.Unmarshal([]byte(raw), &tree); err == nil {
			pins = append(pins, walkPins(tree, "")...)
		}
	}

	if list, ok := getPath(doc, rule.ParametersPath).([]any); ok {
		pins = append(pins, parameterPins(list)...)
	}

	if list, ok := getPath(doc, rule.FileParametersPath).([]any); ok {
		for _, entry := range list {
			e, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			name, _ := e["name"].(string)
			if isPinPath(name) {
				pins = append(pins, models.Pin{Path: name, Opaque: true})
			}
		}
	}

	sort.Slice(pins, func(i, j int) bool { return pins[i].Path < pins[j].Path })
	return dedupePins(pins)
}

func pinsFor(doc map[string]any, rule models.ChartRule, identity map[string][]models.Pin) []models.Pin {
	pins := collectPins(doc, rule)
	if name := getString(doc, "metadata.name"); name != "" {
		pins = append(pins, identity[getString(doc, "metadata.namespace")+"/"+name]...)
	}
	sort.SliceStable(pins, func(i, j int) bool { return pins[i].Path < pins[j].Path })
	return dedupePins(pins)
}

func dedupePins(pins []models.Pin) []models.Pin {
	out := pins[:0]
	seen := map[string]bool{}
	for _, p := range pins {
		if seen[p.Path] {
			continue
		}
		seen[p.Path] = true
		out = append(out, p)
	}
	return out
}

func parameterPins(list []any) []models.Pin {
	var pins []models.Pin
	for _, entry := range list {
		e, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		name, _ := e["name"].(string)
		if !isPinPath(name) {
			continue
		}
		pins = append(pins, models.Pin{
			Path:   name,
			Value:  scalarString(e["value"]),
			Digest: strings.HasSuffix(name, ".digest") || name == "digest",
		})
	}
	return pins
}

func isPinPath(name string) bool {
	switch {
	case name == "image.tag", name == "image.digest":
		return true
	case strings.HasSuffix(name, ".image.tag"), strings.HasSuffix(name, ".image.digest"):
		return true
	default:
		return false
	}
}

func walkPins(node map[string]any, prefix string) []models.Pin {
	var pins []models.Pin
	for key, value := range node {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		if key == "image" {
			switch v := value.(type) {
			case map[string]any:
				if tag := scalarString(v["tag"]); tag != "" {
					pins = append(pins, models.Pin{Path: path + ".tag", Value: tag})
				}
				if digest := scalarString(v["digest"]); digest != "" {
					pins = append(pins, models.Pin{Path: path + ".digest", Value: digest, Digest: true})
				}
			case string:
				if _, tag, ok := splitImageRef(v); ok {
					pins = append(pins, models.Pin{Path: path, Value: tag, Ref: true})
				}
			}
		}

		if child, ok := value.(map[string]any); ok {
			pins = append(pins, walkPins(child, path)...)
		}
	}
	return pins
}

// A registry host may carry a port, so only a colon after the last slash is a tag.
func splitImageRef(ref string) (string, string, bool) {
	slash := strings.LastIndex(ref, "/")
	colon := strings.LastIndex(ref, ":")
	if colon < 0 || colon < slash {
		return ref, "", false
	}
	return ref[:colon], ref[colon+1:], true
}

func scalarString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	case map[string]any, []any:
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}

func comparePin(pin models.Pin, chartDefault string) PinReport {
	r := PinReport{Pin: pin, Default: chartDefault}
	switch {
	case pin.Value == chartDefault:
		r.State = PinRedundant
	case pin.Digest:
		r.State = PinDiffers
	default:
		r.State = compareTags(pin.Value, chartDefault)
	}
	return r
}

func compareTags(pinned, chartDefault string) PinState {
	pinCore, pinRest, pinDigest := splitTag(pinned)
	defCore, defRest, defDigest := splitTag(chartDefault)
	if pinCore == "" || defCore == "" || pinRest != defRest {
		return PinDiffers
	}
	switch compareNumericCores(pinCore, defCore) {
	case 1:
		return PinAhead
	case -1:
		return PinBehind
	}
	if pinDigest != defDigest {
		return PinDiffers
	}
	return PinRedundant
}

// A digest suffix says nothing about ordering, a variant suffix does.
func splitTag(tag string) (string, string, string) {
	digest := ""
	if at := strings.Index(tag, "@"); at >= 0 {
		tag, digest = tag[:at], tag[at:]
	}
	core := tagCoreRe.FindString(tag)
	return core, tag[len(core):], digest
}

func compareNumericCores(a, b string) int {
	as := strings.Split(strings.TrimPrefix(a, "v"), ".")
	bs := strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		av, bv := 0, 0
		if i < len(as) {
			av, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bv, _ = strconv.Atoi(bs[i])
		}
		if av != bv {
			if av > bv {
				return 1
			}
			return -1
		}
	}
	return 0
}
