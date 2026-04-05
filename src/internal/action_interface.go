package internal

import (
	"github.com/sethvargo/go-githubactions"
)

type ActionInterface interface {
	GetInput(name string) string
	Getenv(name string) string
	Debugf(format string, args ...any)
	Fatalf(format string, args ...any)
	Infof(format string, args ...any)
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

func (g *GithubActionInterface) Debugf(format string, args ...any) {
	g.action.Debugf(format, args...)
}

func (g *GithubActionInterface) Fatalf(format string, args ...any) {
	g.action.Fatalf(format, args...)
}

func (g *GithubActionInterface) Infof(format string, args ...any) {
	g.action.Infof(format, args...)
}
