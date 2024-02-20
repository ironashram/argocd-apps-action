package internal

import "github.com/sethvargo/go-githubactions"

type ActionInterface interface {
	GetInput(name string) string
	Getenv(name string) string
	Debugf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

var _ ActionInterface = &GithubActionInterface{}

type GithubActionInterface struct {
	action *githubactions.Action
}

func NewGithubActionInterface() *GithubActionInterface {
	return &GithubActionInterface{
		action: githubactions.New(),
	}
}

func (g *GithubActionInterface) GetInput(name string) string {
	return g.action.GetInput(name)
}

func (g *GithubActionInterface) Getenv(name string) string {
	return g.action.Getenv(name)
}

func (g *GithubActionInterface) Debugf(format string, args ...interface{}) {
	g.action.Debugf(format, args...)
}

func (g *GithubActionInterface) Fatalf(format string, args ...interface{}) {
	g.action.Fatalf(format, args...)
}

type Config struct {
	TargetBranch string
	CreatePr     bool
	AppsFolder   string
	Token        string
	Repo         string
	Workspace    string
}

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
