package internal

import (
	"os"

	"github.com/stretchr/testify/mock"
)

type OSInterface interface {
	ReadFile(path string) ([]byte, error)
}

type MockOS struct {
	mock.Mock
}

func (m *MockOS) ReadFile(path string) ([]byte, error) {
	args := m.Called(path)
	return args.Get(0).([]byte), args.Error(1)
}

type OSWrapper struct{}

func (osw *OSWrapper) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
