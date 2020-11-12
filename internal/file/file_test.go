package file

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/vimcolorschemes/worker/internal/test"
)

func TestGetFileURLsWithExtensions(t *testing.T) {
	t.Run("should not match a non-suffix substring", func(t *testing.T) {
		fileURLs := []string{"http://example.com/test.vim/test.txt"}
		extensions := []string{"vim"}
		result := GetFileURLsWithExtensions(fileURLs, extensions)
		if !reflect.DeepEqual(result, []string{}) {
			t.Error("Incorrect result for GetFileURLsWithExtensions, got non empty result")
		}
	})

	t.Run("should match differences in casing", func(t *testing.T) {
		fileURLs := []string{"http://example.com/test.ViM"}
		extensions := []string{"vim"}
		result := GetFileURLsWithExtensions(fileURLs, extensions)
		if !reflect.DeepEqual(result, fileURLs) {
			t.Errorf("Incorrect result for GetFileURLsWithExtensions, got: empty result, want: %s", fileURLs)
		}
	})

	t.Run("should return all file URLs matching the extension", func(t *testing.T) {
		match1 := "http://example.com/test.ViM"
		match2 := "https://helloworld.ca/file.vim"
		fileURLs := []string{match1, match2, "https://example.com/hello/world.html"}
		extensions := []string{"vim"}
		result := GetFileURLsWithExtensions(fileURLs, extensions)
		if !reflect.DeepEqual(result, []string{match1, match2}) {
			t.Errorf("Incorrect result for GetFileURLsWithExtensions, got: empty result, want: %s", []string{match1, match2})
		}
	})

	t.Run("should return all file URLs matching any of the extensions", func(t *testing.T) {
		match1 := "http://example.com/test.ViM"
		match2 := "http://example.com/test.vim"
		match3 := "https://helloworld.ca/file.erb"
		fileURLs := []string{match1, match2, match3, "https://example.com/hello/world.html"}
		extensions := []string{"vim", "erb"}
		result := GetFileURLsWithExtensions(fileURLs, extensions)
		if !reflect.DeepEqual(result, []string{match1, match2, match3}) {
			t.Errorf("Incorrect result for GetFileURLsWithExtensions, got: empty result, want: %s", []string{match1, match2, match3})
		}
	})

	t.Run("should return an empty array when given empty values", func(t *testing.T) {
		fileURLs := []string{}
		extensions := []string{}
		result := GetFileURLsWithExtensions(fileURLs, extensions)
		if !reflect.DeepEqual(result, []string{}) {
			t.Error("Incorrect result for GetFileURLsWithExtensions, got non empty result")
		}
	})

	t.Run("should return an empty array when given an empty file URLs array", func(t *testing.T) {
		fileURLs := []string{}
		extensions := []string{"txt"}
		result := GetFileURLsWithExtensions(fileURLs, extensions)
		if !reflect.DeepEqual(result, []string{}) {
			t.Error("Incorrect result for GetFileURLsWithExtensions, got non empty result")
		}
	})

	t.Run("should return an empty array when given an empty extensions array", func(t *testing.T) {
		fileURLs := []string{"http://example.com/test/test.txt"}
		extensions := []string{}
		result := GetFileURLsWithExtensions(fileURLs, extensions)
		if !reflect.DeepEqual(result, []string{}) {
			t.Error("Incorrect result for GetFileURLsWithExtensions, got non empty result")
		}
	})

	t.Run("should return an empty array when the file URL does not match the given extension", func(t *testing.T) {
		fileURLs := []string{"http://example.com/test/test.txt"}
		extensions := []string{"vim"}
		result := GetFileURLsWithExtensions(fileURLs, extensions)
		if !reflect.DeepEqual(result, []string{}) {
			t.Error("Incorrect result for GetFileURLsWithExtensions, got non empty result")
		}
	})

	t.Run("should return an empty array when the file URL does not match any of the given extensions", func(t *testing.T) {
		fileURLs := []string{"http://example.com/test/test.txt"}
		extensions := []string{"vim", "erb"}
		result := GetFileURLsWithExtensions(fileURLs, extensions)
		if !reflect.DeepEqual(result, []string{}) {
			t.Error("Incorrect result for GetFileURLsWithExtensions, got non empty result")
		}
	})

	t.Run("should return an empty array when none of the file URLs match the given extension", func(t *testing.T) {
		fileURLs := []string{"http://example.com/test/test.txt", "https://helloworld.ca/file.html"}
		extensions := []string{"vim"}
		result := GetFileURLsWithExtensions(fileURLs, extensions)
		if !reflect.DeepEqual(result, []string{}) {
			t.Error("Incorrect result for GetFileURLsWithExtensions, got non empty result")
		}
	})

	t.Run("should return an empty array when none of the file URLs match any of the given extensions", func(t *testing.T) {
		fileURLs := []string{"http://example.com/test/test.txt", "https://helloworld.ca/file.html"}
		extensions := []string{"vim", "md"}
		result := GetFileURLsWithExtensions(fileURLs, extensions)
		if !reflect.DeepEqual(result, []string{}) {
			t.Error("Incorrect result for GetFileURLsWithExtensions, got non empty result")
		}
	})
}

func TestDownloadFile(t *testing.T) {
	t.Run("should return file content on successful query", func(t *testing.T) {
		expectedFileContent := "file content"

		server := test.MockServer(expectedFileContent, http.StatusOK)
		defer server.Close()

		fileContent, err := DownloadFile(server.URL)

		if err != nil {
			t.Errorf("Got unexpected error: %s", err)
		}

		if fileContent != expectedFileContent {
			t.Errorf("Incorrect file content for DownloadFile, got: %s, want: %s", fileContent, expectedFileContent)
		}
	})

	t.Run("should return error when status code is not ok", func(t *testing.T) {
		expectedFileContent := "file content"

		server := test.MockServer(expectedFileContent, http.StatusNotFound)
		defer server.Close()

		_, err := DownloadFile(server.URL)

		if err == nil {
			t.Error("Incorrect result for DownloadFile, got no error")
		}
	})

	t.Run("should return error when URL is not valid", func(t *testing.T) {
		_, err := DownloadFile("test")
		if err == nil {
			t.Error("Incorrect result for DownloadFile, got no error")
		}
	})
}
