package mocks

import (
	"github.com/stretchr/testify/mock"
)

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
