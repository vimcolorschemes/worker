package file

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/google/go-github/v32/github"
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

// DownloadFile downloads a file from a URL at a local target path
func DownloadFile(fileURL string, target string) error {
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(fileURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
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

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}

// GetLocalFileContent returns the file content of a local file at a path
func GetLocalFileContent(path string) (string, error) {
	content, err := ioutil.ReadFile(path)
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

	err = ioutil.WriteFile(path, []byte(newFileContent), 0600)
	if err != nil {
		return err
	}

	return nil
}
