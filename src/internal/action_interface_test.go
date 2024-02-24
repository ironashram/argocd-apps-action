package internal

import (
	"testing"

	"github.com/stretchr/testify/mock"
)

func TestActionInterface_GetInput(t *testing.T) {
	testCases := []struct {
		name     string
		action   *MockActionInterface
		input    string
		expected string
	}{
		{
			name: "Test Case 1",
			action: &MockActionInterface{
				Inputs: map[string]string{
					"input1": "value1",
					"input2": "value2",
				},
			},
			input:    "input1",
			expected: "value1",
		},
		{
			name: "Test Case 2",
			action: &MockActionInterface{
				Inputs: map[string]string{
					"input1": "value1",
					"input2": "value2",
				},
			},
			input:    "input2",
			expected: "value2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.action.GetInput(tc.input)
			if result != tc.expected {
				t.Errorf("Expected result: %s, got: %s", tc.expected, result)
			}
		})
	}
}

func TestActionInterface_Getenv(t *testing.T) {
	testCases := []struct {
		name     string
		action   *MockActionInterface
		env      string
		expected string
	}{
		{
			name: "Test Case 1",
			action: &MockActionInterface{
				Env: map[string]string{
					"env1": "value1",
					"env2": "value2",
				},
			},
			env:      "env1",
			expected: "value1",
		},
		{
			name: "Test Case 2",
			action: &MockActionInterface{
				Env: map[string]string{
					"env1": "value1",
					"env2": "value2",
				},
			},
			env:      "env2",
			expected: "value2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.action.Getenv(tc.env)
			if result != tc.expected {
				t.Errorf("Expected result: %s, got: %s", tc.expected, result)
			}
		})
	}
}

func TestActionInterface_Debugf(t *testing.T) {
	testCases := []struct {
		name   string
		action *MockActionInterface
		format string
		args   []interface{}
	}{
		{
			name: "Test Case 1",
			action: &MockActionInterface{
				Inputs: map[string]string{},
				Env:    map[string]string{},
			},
			format: "Debug message",
			args:   []interface{}{},
		},
		{
			name: "Test Case 2",
			action: &MockActionInterface{
				Inputs: map[string]string{},
				Env:    map[string]string{},
			},
			format: "Another debug message",
			args:   []interface{}{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.action.On("Debugf", tc.format, mock.Anything).Once()
			tc.action.Debugf(tc.format, tc.args...)
		})
	}
}

func TestActionInterface_Fatalf(t *testing.T) {
	testCases := []struct {
		name   string
		action *MockActionInterface
		format string
		args   []interface{}
	}{
		{
			name: "Test Case 1",
			action: &MockActionInterface{
				Inputs: map[string]string{},
				Env:    map[string]string{},
			},
			format: "Fatal error",
			args:   []interface{}{},
		},
		{
			name: "Test Case 2",
			action: &MockActionInterface{
				Inputs: map[string]string{},
				Env:    map[string]string{},
			},
			format: "Another fatal error",
			args:   []interface{}{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.action.On("Fatalf", tc.format, mock.Anything).Once()
			tc.action.Fatalf(tc.format, tc.args...)
		})
	}
}
