package internal

import (
	"os"
)

type OSInterface interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
}

type OSWrapper struct{}

func (osw *OSWrapper) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (osw *OSWrapper) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}
