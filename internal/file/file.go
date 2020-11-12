package file

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

func GetFileURLsWithExtensions(fileURLs []string, extensions []string) []string {
	result := []string{}

	fileExtensionExpression := strings.Join(extensions, "|")
	expression := fmt.Sprintf(`(?i)^.*\.(%s)$`, fileExtensionExpression)
	fileURLWithExtensions := regexp.MustCompile(expression)

	for _, fileURL := range fileURLs {
		if fileURLWithExtensions.MatchString(fileURL) {
			result = append(result, fileURL)
		}
	}

	return result
}

func DownloadFile(fileURL string) (string, error) {
	response, err := http.Get(fileURL)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", errors.New(fmt.Sprintf("status code: %d", response.StatusCode))
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}
