package internal

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockOS_ReadFile(t *testing.T) {
	mockOS := new(MockOS)

	path := "/path/to/file"
	expectedData := []byte("file data")
	expectedError := errors.New("read error")

	mockOS.On("ReadFile", path).Return(expectedData, expectedError)

	data, err := mockOS.ReadFile(path)

	mockOS.AssertExpectations(t)
	assert.Equal(t, expectedData, data)
	assert.Equal(t, expectedError, err)
}

func TestOSWrapper_ReadFile(t *testing.T) {
	tmpfile, err := os.CreateTemp("/tmp", "example")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	expectedData := []byte("file data")
	if _, err := tmpfile.Write(expectedData); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	data, err := (&OSWrapper{}).ReadFile(tmpfile.Name())

	assert.NoError(t, err)
	assert.Equal(t, expectedData, data)
}
