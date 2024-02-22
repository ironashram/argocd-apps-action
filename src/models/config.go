package models

type Config struct {
	TargetBranch string
	CreatePr     bool
	AppsFolder   string
	Token        string
	Repo         string
	Workspace    string
}
