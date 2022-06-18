package file

import (
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/google/go-github/v32/github"
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

func TestGetFilesWithExtensions(t *testing.T) {
	t.Run("should not match a non-suffix substring", func(t *testing.T) {
		url := "http://example.com/test.vim/test.txt"
		files := []*github.RepositoryContent{{DownloadURL: &url}}
		extensions := []string{"vim"}
		result := GetFilesWithExtensions(files, extensions)
		if len(result) != 0 {
			t.Error("Incorrect result for GetFilesWithExtensions, got non empty result")
		}
	})

	t.Run("should match differences in casing", func(t *testing.T) {
		url := "http://example.com/test.ViM"
		files := []*github.RepositoryContent{{DownloadURL: &url}}
		extensions := []string{"vim"}
		result := GetFilesWithExtensions(files, extensions)
		if !reflect.DeepEqual(result, files) {
			t.Errorf("Incorrect result for GetFilesWithExtensions, got: empty result, want: %s, got: %s", files, result)
		}
	})

	t.Run("should return all file s matching the extension", func(t *testing.T) {
		match1 := "http://example.com/test.ViM"
		match2 := "https://helloworld.ca/file.vim"
		url3 := "https://example.com/hello/world.html"
		files := []*github.RepositoryContent{{DownloadURL: &match1}, {DownloadURL: &match2}, {DownloadURL: &url3}}
		extensions := []string{"vim"}
		result := GetFilesWithExtensions(files, extensions)
		expectedResult := []*github.RepositoryContent{{DownloadURL: &match1}, {DownloadURL: &match2}}
		if len(result) != len(expectedResult) {
			t.Errorf("Incorrect result for GetFilesWithExtensions, got: empty result, want: %s, got: %s", expectedResult, result)
		}
	})

	t.Run("should return all file s matching any of the extensions", func(t *testing.T) {
		match1 := "http://example.com/test.ViM"
		match2 := "http://example.com/test.vim"
		match3 := "https://helloworld.ca/file.erb"
		url4 := "https://example.com/hello/world.html"
		files := []*github.RepositoryContent{{DownloadURL: &match1}, {DownloadURL: &match2}, {DownloadURL: &match3}, {DownloadURL: &url4}}
		extensions := []string{"vim", "erb"}
		result := GetFilesWithExtensions(files, extensions)
		expectedResult := []*github.RepositoryContent{{DownloadURL: &match1}, {DownloadURL: &match2}, {DownloadURL: &match3}}
		if len(result) != len(expectedResult) {
			t.Errorf("Incorrect result for GetFilesWithExtensions, got: empty result, want: %s, got: %s", expectedResult, result)
		}
	})

	t.Run("should return an empty array when given empty values", func(t *testing.T) {
		files := []*github.RepositoryContent{}
		extensions := []string{}
		result := GetFilesWithExtensions(files, extensions)
		if len(result) != 0 {
			t.Error("Incorrect result for GetFilesWithExtensions, got non empty result")
		}
	})

	t.Run("should return an empty array when given an empty file s array", func(t *testing.T) {
		files := []*github.RepositoryContent{}
		extensions := []string{"txt"}
		result := GetFilesWithExtensions(files, extensions)
		if len(result) != 0 {
			t.Error("Incorrect result for GetFilesWithExtensions, got non empty result")
		}
	})

	t.Run("should return an empty array when given an empty extensions array", func(t *testing.T) {
		url := "http://example.com/test/test.txt"
		files := []*github.RepositoryContent{{DownloadURL: &url}}
		extensions := []string{}
		result := GetFilesWithExtensions(files, extensions)
		if len(result) != 0 {
			t.Error("Incorrect result for GetFilesWithExtensions, got non empty result")
		}
	})

	t.Run("should return an empty array when the file  does not match the given extension", func(t *testing.T) {
		url := "http://example.com/test/test.txt"
		files := []*github.RepositoryContent{{DownloadURL: &url}}
		extensions := []string{"vim"}
		result := GetFilesWithExtensions(files, extensions)
		if len(result) != 0 {
			t.Error("Incorrect result for GetFilesWithExtensions, got non empty result")
		}
	})

	t.Run("should return an empty array when the file  does not match any of the given extensions", func(t *testing.T) {
		url := "http://example.com/test/test.txt"
		files := []*github.RepositoryContent{{DownloadURL: &url}}
		extensions := []string{"vim", "erb"}
		result := GetFilesWithExtensions(files, extensions)
		if len(result) != 0 {
			t.Error("Incorrect result for GetFilesWithExtensions, got non empty result")
		}
	})

	t.Run("should return an empty array when none of the file s match the given extension", func(t *testing.T) {
		url1 := "http://example.com/test/test.txt"
		url2 := "https://helloworld.ca/file.html"
		files := []*github.RepositoryContent{{DownloadURL: &url1}, {DownloadURL: &url2}}
		extensions := []string{"vim"}
		result := GetFilesWithExtensions(files, extensions)
		if len(result) != 0 {
			t.Error("Incorrect result for GetFilesWithExtensions, got non empty result")
		}
	})

	t.Run("should return an empty array when none of the file s match any of the given extensions", func(t *testing.T) {
		url1 := "http://example.com/test/test.txt"
		url2 := "https://helloworld.ca/file.html"
		files := []*github.RepositoryContent{{DownloadURL: &url1}, {DownloadURL: &url2}}
		extensions := []string{"vim", "md"}
		result := GetFilesWithExtensions(files, extensions)
		if len(result) != 0 {
			t.Error("Incorrect result for GetFilesWithExtensions, got non empty result")
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

	t.Run("should return error when  is not valid", func(t *testing.T) {
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
		err := AppendToFile(content, target)
		if err != nil {
			t.Errorf("Error during AppendToFile, %s", err)
		}

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
		err = AppendToFile(content, target)
		if err != nil {
			t.Errorf("Error during AppendToFile, %s", err)
		}

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
		err = AppendToFile(content, target)
		if err != nil {
			t.Errorf("Error during AppendToFile, %s", err)
		}

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
	t.Run("should download the file locally if the  is valid", func(t *testing.T) {
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

	t.Run("should return error if the  is invalid", func(t *testing.T) {
		cleanUp := setUp(t)
		defer cleanUp(t)

		err := DownloadFile("Wrong ", target)

		if err == nil {
			t.Error("Incorrect result for DownloadFile, got no error when  was invalid")
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
