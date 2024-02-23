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

type Index struct {
	Entries map[string][]struct {
		Version string `yaml:"version"`
	} `yaml:"entries"`
}
