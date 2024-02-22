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

type MockActionInterface struct {
	Inputs map[string]string
	Env    map[string]string
}

func (m MockActionInterface) GetInput(name string) string {
	return m.Inputs[name]
}

func (m MockActionInterface) Getenv(name string) string {
	return m.Env[name]
}

func (m MockActionInterface) Debugf(format string, args ...interface{}) {
}

func (m MockActionInterface) Fatalf(format string, args ...interface{}) {
}
