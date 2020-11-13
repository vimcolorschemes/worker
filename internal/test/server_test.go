package test

import (
	"net/http"
	"testing"
)

func TestMockServer(t *testing.T) {
	t.Run("should build a mock server using the response body and status code", func(t *testing.T) {
		server := MockServer("response body", http.StatusBadRequest)
		defer server.Close()

		response, err := http.Get(server.URL)

		if err != nil {
			t.Errorf("Incorrect result for MockServer, got error: %s", err)
		}

		if response.StatusCode != http.StatusBadRequest {
			t.Errorf("Incorrect status code for MockServer, got: %d, want: %d", response.StatusCode, http.StatusBadRequest)
		}
	})
}
