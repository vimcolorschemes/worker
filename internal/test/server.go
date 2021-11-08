package test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

// MockServer returns a mock server in order to customize its response
func MockServer(response string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(statusCode)
		_, err := rw.Write([]byte(response))
		if err != nil {
			fmt.Println("Error creating response")
		}
	}))
}
