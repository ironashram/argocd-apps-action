package models

type RepoRef struct {
	NamePath      string `yaml:"namePath"`
	NamespacePath string `yaml:"namespacePath"`
}

type RepoRule struct {
	Files         []string `yaml:"files"`
	NamePath      string   `yaml:"namePath"`
	NamespacePath string   `yaml:"namespacePath"`
	URLPath       string   `yaml:"urlPath"`
	SkipIfSet     string   `yaml:"skipIfSet"`
}

type ChartRule struct {
	Files         []string `yaml:"files"`
	ChartPath     string   `yaml:"chartPath"`
	VersionPath   string   `yaml:"versionPath"`
	URLPath       string   `yaml:"urlPath"`
	RepoRef       *RepoRef `yaml:"repoRef"`
	RegexFallback bool     `yaml:"regexFallback"`
}

type SourcesConfig struct {
	Repositories []RepoRule  `yaml:"repositories"`
	Charts       []ChartRule `yaml:"charts"`
}
