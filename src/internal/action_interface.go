package internal

import (
	"fmt"

	"github.com/sethvargo/go-githubactions"
	"github.com/stretchr/testify/mock"
)

type ActionInterface interface {
	GetInput(name string) string
	Getenv(name string) string
	Debugf(format string, args ...any)
	Fatalf(format string, args ...any)
	Infof(format string, args ...any)
	GetAddress() string
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

func (a *GithubActionInterface) GetAddress() string {
	return fmt.Sprintf("%p", a)
}

type MockActionInterface struct {
	Inputs map[string]string
	Env    map[string]string
	mock.Mock
}

func (m *MockActionInterface) GetInput(name string) string {
	return m.Inputs[name]
}

func (m *MockActionInterface) Getenv(name string) string {
	return m.Env[name]
}

func (m *MockActionInterface) Debugf(format string, args ...any) {
	m.Called(format, args)
}

func (m *MockActionInterface) Fatalf(format string, args ...any) {
	m.Called(format, args)
}

func (m *MockActionInterface) Infof(format string, args ...any) {
	m.Called(format, args)
}

func (m *MockActionInterface) GetAddress() string {
	return fmt.Sprintf("%p", m)
}
