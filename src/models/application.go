package models

type Source struct {
	Chart          string `yaml:"chart"`
	RepoURL        string `yaml:"repoURL"`
	TargetRevision string `yaml:"targetRevision"`
}

type Spec struct {
	Source Source `yaml:"source"`
}

type Application struct {
	Spec Spec `yaml:"spec"`
}

type IndexEntry struct {
	Version string   `yaml:"version"`
	URLs    []string `yaml:"urls"`
}

type Index struct {
	Entries map[string][]IndexEntry `yaml:"entries"`
}

type Pin struct {
	Path   string
	Value  string
	Digest bool
	Opaque bool
	Ref    bool
}

type ChartRef struct {
	RepoURL string
	Chart   string
}

type AppFile struct {
	Path           string
	CurrentVersion string
	VersionPath    string
	DocIndex       int
	Pins           []Pin
}
