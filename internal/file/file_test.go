package file

import (
	"os"
	"testing"
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

		actualContent, err := os.ReadFile(target)
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

		actualContent, err := os.ReadFile(target)
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

		actualContent, err := os.ReadFile(target)
		if err != nil {
			t.Error("Incorrect result for AppendToFile, error reading file")
		}

		if string(actualContent) != expectedContent {
			t.Errorf("Incorrect result for AppendToFile, got: %s, want: %s", actualContent, expectedContent)
		}
	})
}
