package utils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetHTTPResponse(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/success" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Success response")
		} else if r.URL.Path == "/error" {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "Error response")
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "Not found")
		}
	}))
	defer server.Close()

	// Test cases
	testCases := []struct {
		name     string
		url      string
		expected []byte
		err      error
	}{
		{
			name:     "Test Case 1: Successful request",
			url:      server.URL + "/success",
			expected: []byte("Success response"),
			err:      nil,
		},
		{
			name:     "Test Case 2: Error response",
			url:      server.URL + "/error",
			expected: nil,
			err:      fmt.Errorf("HTTP request failed with status code 500"),
		},
		{
			name:     "Test Case 3: Not found",
			url:      server.URL + "/notfound",
			expected: nil,
			err:      fmt.Errorf("HTTP request failed with status code 404"),
		},
	}

	// Run tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := GetHTTPResponse(tc.url)
			if err != nil {
				if tc.err == nil || err.Error() != tc.err.Error() {
					t.Errorf("Expected error: %v, got: %v", tc.err, err)
				}
			} else {
				if string(result) != string(tc.expected) {
					t.Errorf("Expected result: %s, got: %s", tc.expected, result)
				}
			}
		})
	}
}
