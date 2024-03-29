package internal

import (
	"os"

	"github.com/stretchr/testify/mock"
)

type OSInterface interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
}

type MockOS struct {
	mock.Mock
}

func (m *MockOS) ReadFile(path string) ([]byte, error) {
	args := m.Called(path)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockOS) WriteFile(filename string, data []byte, perm os.FileMode) error {
	args := m.Called(filename, data, perm)
	return args.Error(0)
}

type OSWrapper struct{}

func (osw *OSWrapper) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (osw *OSWrapper) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}
