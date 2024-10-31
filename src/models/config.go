package models

type Config struct {
	SkipPreRelease bool
	TargetBranch   string
	CreatePr       bool
	AppsFolder     string
	Token          string
	Repo           string
	Workspace      string
	Owner          string
	Name           string
	Labels         []string
}
