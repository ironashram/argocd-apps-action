package mocks

import (
	"os"

	"github.com/stretchr/testify/mock"
)

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
