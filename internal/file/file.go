package file

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/google/go-github/v62/github"
)

// GetFileURLsWithExtensions returns all URLs with a certain extension given a URL list
func GetFilesWithExtensions(files []*github.RepositoryContent, extensions []string) []*github.RepositoryContent {
	result := []*github.RepositoryContent{}

	fileExtensionExpression := strings.Join(extensions, "|")
	expression := fmt.Sprintf(`(?i)^.*\.(%s)$`, fileExtensionExpression)
	fileURLWithExtensions := regexp.MustCompile(expression)

	for _, file := range files {
		downloadURL := file.GetDownloadURL()
		if downloadURL != "" && fileURLWithExtensions.MatchString(downloadURL) {
			result = append(result, file)
		}
	}

	return result
}

// GetRemoteFileContent returns the file content of a file at a URL
func GetRemoteFileContent(fileURL string) (string, error) {
	response, err := http.Get(fileURL)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status code: %d", response.StatusCode)
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}

// GetLocalFileContent returns the file content of a local file at a path
func GetLocalFileContent(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// AppendToFile adds content to a local file
func AppendToFile(content string, path string) error {
	log.Printf("Appending to %s", path)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	return file.Sync()
}

// RemoveLinesInFile deletes lines matching a regex from a local file
func RemoveLinesInFile(expression string, path string) error {
	log.Printf("Removing lines matching %s in %s", expression, path)

	compiledExpression := regexp.MustCompile(expression)

	fileContent, err := GetLocalFileContent(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(fileContent), "\n")
	newLines := []string{}

	for _, line := range lines {
		if !compiledExpression.MatchString(line) {
			newLines = append(newLines, line)
		}
	}

	newFileContent := strings.Join(newLines, "\n")

	err = os.WriteFile(path, []byte(newFileContent), 0600)
	if err != nil {
		return err
	}

	return nil
}
