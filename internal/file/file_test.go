package file

import (
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/vimcolorschemes/worker/internal/test"
)

var target = ".vimcolorschemes-file-test.tmp"

func cleanUp(t *testing.T) {
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		os.Remove(target)
	}
}

func setUp(t *testing.T) func(t *testing.T) {
	cleanUp(t)
	return cleanUp
}

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

func TestGetRemoteFileContent(t *testing.T) {
	t.Run("should return file content on successful query", func(t *testing.T) {
		expectedFileContent := "file content"

		server := test.MockServer(expectedFileContent, http.StatusOK)
		defer server.Close()

		fileContent, err := GetRemoteFileContent(server.URL)

		if err != nil {
			t.Errorf("Got unexpected error: %s", err)
		}

		if fileContent != expectedFileContent {
			t.Errorf("Incorrect file content for GetRemoteFileContent, got: %s, want: %s", fileContent, expectedFileContent)
		}
	})

	t.Run("should return error when status code is not ok", func(t *testing.T) {
		expectedFileContent := "file content"

		server := test.MockServer(expectedFileContent, http.StatusNotFound)
		defer server.Close()

		_, err := GetRemoteFileContent(server.URL)

		if err == nil {
			t.Error("Incorrect result for GetRemoteFileContent, got no error")
		}
	})

	t.Run("should return error when URL is not valid", func(t *testing.T) {
		_, err := GetRemoteFileContent("test")
		if err == nil {
			t.Error("Incorrect result for GetRemoteFileContent, got no error")
		}
	})
}

func TestGetLocalFileContent(t *testing.T) {
	t.Run("should return file content if file exists", func(t *testing.T) {
		cleanUp := setUp(t)
		defer cleanUp(t)

		expectedFileContent := "file content"

		file, err := os.Create(target)
		if err != nil {
			t.Errorf("Incorrect result for GetLocalFileContent, error creating file: %s", err)
		}

		if _, err := os.Stat(target); os.IsNotExist(err) {
			t.Error("Incorrect result for GetLocalFileContent, did not create file")
		}

		_, err = file.WriteString(expectedFileContent)
		if err != nil {
			t.Errorf("Incorrect result for GetLocalFileContent, error writing to file: %s", err)
		}

		fileContent, err := GetLocalFileContent(target)

		if err != nil {
			t.Errorf("Incorrect result for GetLocalFileContent, got error: %s", err)
		}

		if fileContent != expectedFileContent {
			t.Errorf("Incorrect result for GetLocalFileContent, got: %s, want: %s", fileContent, expectedFileContent)
		}
	})

	t.Run("should return error if file does not exist", func(t *testing.T) {
		cleanUp := setUp(t)
		defer cleanUp(t)

		_, err := GetLocalFileContent(target)

		if err == nil {
			t.Error("Incorrect result for GetLocalFileContent, did not get error")
		}
	})
}

func TestAppendToFile(t *testing.T) {
	t.Run("should create file if does not exist", func(t *testing.T) {
		cleanUp := setUp(t)
		defer cleanUp(t)

		content := "test"
		AppendToFile(content, target)

		if _, err := os.Stat(target); os.IsNotExist(err) {
			t.Error("Incorrect result for AppendToFile, did not create file")
		}

		actualContent, err := ioutil.ReadFile(target)
		if err != nil {
			t.Error("Incorrect result for AppendToFile, error reading file")
		}

		if string(actualContent) != content {
			t.Errorf("Incorrect result for AppendToFile, got: %s, want: %s", actualContent, content)
		}
	})

	t.Run("should append to file if exists", func(t *testing.T) {
		cleanUp := setUp(t)
		defer cleanUp(t)

		_, err := os.Create(target)
		if err != nil {
			t.Errorf("Incorrect result for AppendToFile, error creating file: %s", err)
		}

		if _, err := os.Stat(target); os.IsNotExist(err) {
			t.Error("Incorrect result for AppendToFile, did not create file")
		}

		content := "test"
		AppendToFile(content, target)

		actualContent, err := ioutil.ReadFile(target)
		if err != nil {
			t.Error("Incorrect result for AppendToFile, error reading file")
		}

		if string(actualContent) != content {
			t.Errorf("Incorrect result for AppendToFile, got: %s, want: %s", actualContent, content)
		}
	})

	t.Run("should append to file with content", func(t *testing.T) {
		cleanUp := setUp(t)
		defer cleanUp(t)

		file, err := os.Create(target)
		if err != nil {
			t.Errorf("Incorrect result for AppendToFile, error creating file: %s", err)
		}

		if _, err := os.Stat(target); os.IsNotExist(err) {
			t.Error("Incorrect result for AppendToFile, did not create file")
		}

		_, err = file.WriteString("hello, ")
		if err != nil {
			t.Errorf("Incorrect result for AppendToFile, error writing to file: %s", err)
		}

		content := "world"
		AppendToFile(content, target)

		expectedContent := "hello, world"

		actualContent, err := ioutil.ReadFile(target)
		if err != nil {
			t.Error("Incorrect result for AppendToFile, error reading file")
		}

		if string(actualContent) != expectedContent {
			t.Errorf("Incorrect result for AppendToFile, got: %s, want: %s", actualContent, expectedContent)
		}
	})
}

func TestRemoveLinesInFile(t *testing.T) {
	t.Run("should remove line matching regex", func(t *testing.T) {
		cleanUp := setUp(t)
		defer cleanUp(t)

		file, err := os.Create(target)
		if err != nil {
			t.Errorf("Incorrect result for RemoveLinesInFile, error creating file: %s", err)
		}

		if _, err := os.Stat(target); os.IsNotExist(err) {
			t.Error("Incorrect result for RemoveLinesInFile, did not create file")
		}

		_, err = file.WriteString("hello")
		if err != nil {
			t.Errorf("Incorrect result for RemoveLinesInFile, error writing to file: %s", err)
		}

		expression := "hello"

		err = RemoveLinesInFile(expression, target)
		if err != nil {
			t.Errorf("Incorrect result for RemoveLinesInFile, got error: %s", err)
		}

		fileContent, err := ioutil.ReadFile(target)
		if err != nil {
			t.Error("Incorrect result for RemoveLinesInFile, error reading file")
		}

		if string(fileContent) != "" {
			t.Errorf("Incorrect result for RemoveLinesInFile, got: %s, want: %s", fileContent, "")
		}
	})

	t.Run("should remove all lines matching regex", func(t *testing.T) {
		cleanUp := setUp(t)
		defer cleanUp(t)

		file, err := os.Create(target)
		if err != nil {
			t.Errorf("Incorrect result for RemoveLinesInFile, error creating file: %s", err)
		}

		if _, err := os.Stat(target); os.IsNotExist(err) {
			t.Error("Incorrect result for RemoveLinesInFile, did not create file")
		}

		content := "hello\ntest\nhello\ntest"

		_, err = file.WriteString(content)
		if err != nil {
			t.Errorf("Incorrect result for RemoveLinesInFile, error writing to file: %s", err)
		}

		expression := "hello"

		err = RemoveLinesInFile(expression, target)
		if err != nil {
			t.Errorf("Incorrect result for RemoveLinesInFile, got error: %s", err)
		}

		fileContent, err := ioutil.ReadFile(target)
		if err != nil {
			t.Error("Incorrect result for RemoveLinesInFile, error reading file")
		}

		expectedFileContent := "test\ntest"
		if string(fileContent) != expectedFileContent {
			t.Errorf("Incorrect result for RemoveLinesInFile, got: %s, want: %s", fileContent, expectedFileContent)
		}
	})

	t.Run("should keep all lines if no match exists", func(t *testing.T) {
		cleanUp := setUp(t)
		defer cleanUp(t)

		file, err := os.Create(target)
		if err != nil {
			t.Errorf("Incorrect result for RemoveLinesInFile, error creating file: %s", err)
		}

		if _, err := os.Stat(target); os.IsNotExist(err) {
			t.Error("Incorrect result for RemoveLinesInFile, did not create file")
		}

		expectedFileContent := "hello\ntest\nhello\ntest"

		_, err = file.WriteString(expectedFileContent)
		if err != nil {
			t.Errorf("Incorrect result for RemoveLinesInFile, error writing to file: %s", err)
		}

		expression := "world"

		err = RemoveLinesInFile(expression, target)
		if err != nil {
			t.Errorf("Incorrect result for RemoveLinesInFile, got error: %s", err)
		}

		fileContent, err := ioutil.ReadFile(target)
		if err != nil {
			t.Error("Incorrect result for RemoveLinesInFile, error reading file")
		}

		if string(fileContent) != expectedFileContent {
			t.Errorf("Incorrect result for RemoveLinesInFile, got: %s, want: %s", fileContent, expectedFileContent)
		}
	})
}

func TestDownloadFile(t *testing.T) {
	t.Run("should download the file locally if the URL is valid", func(t *testing.T) {
		cleanUp := setUp(t)
		defer cleanUp(t)

		expectedFileContent := "file content"

		server := test.MockServer(expectedFileContent, http.StatusOK)
		defer server.Close()

		err := DownloadFile(server.URL, target)

		if err != nil {
			t.Errorf("Incorrect result for DownloadFile, got error: %s", err)
		}

		// Check if file was downloaded
		fileContent, err := GetLocalFileContent(target)
		if err != nil {
			t.Errorf("Incorrect result for DownloadFile, got error reading file: %s", err)
		}

		if fileContent != expectedFileContent {
			t.Errorf("Incorrect result for DownloadFile, got: %s, want: %s", fileContent, expectedFileContent)
		}
	})

	t.Run("should return error if the URL is invalid", func(t *testing.T) {
		cleanUp := setUp(t)
		defer cleanUp(t)

		err := DownloadFile("Wrong URL", target)

		if err == nil {
			t.Error("Incorrect result for DownloadFile, got no error when URL was invalid")
		}
	})

	t.Run("should return error if the target is invalid", func(t *testing.T) {
		cleanUp := setUp(t)
		defer cleanUp(t)

		server := test.MockServer("content", http.StatusOK)
		defer server.Close()

		invalidTarget := ".."

		err := DownloadFile(server.URL, invalidTarget)

		if err == nil {
			t.Error("Incorrect result for DownloadFile, got no error when target was invalid")
		}
	})
}
