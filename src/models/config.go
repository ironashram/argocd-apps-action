package models

type RepoCredential struct {
	URLPrefix string
	Username  string
	Password  string
}

type Config struct {
	SkipPreRelease     bool
	TargetBranch       string
	CreatePr           bool
	AppsFolder         string
	Token              string
	Repo               string
	Workspace          string
	Owner              string
	Name               string
	Labels             []string
	FileExtensions     []string
	AllowRegexFallback bool
	ApiURL             string
	Provider           string
	Preset             string
	SourcesFile        string
	RepoCreds          []RepoCredential
}
