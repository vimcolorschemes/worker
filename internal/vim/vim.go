package vim

import (
	"errors"
	"log"
	"regexp"

	"github.com/vimcolorschemes/worker/internal/file"
)

func GetVimColorSchemeNames(vimFileURLs []string) ([]string, error) {
	vimColorSchemeNames := []string{}

	for _, vimFileURL := range vimFileURLs {
		fileContent, err := file.DownloadFile(vimFileURL)
		if err != nil {
			log.Print("Error downloading file: ", vimFileURL)
			continue
		}

		vimColorSchemeName, err := GetVimColorSchemeName(&fileContent)
		if err != nil || vimColorSchemeName == "" || contains(vimColorSchemeNames, vimColorSchemeName) {
			continue
		}

		log.Print("Found ", vimColorSchemeName)

		vimColorSchemeNames = append(vimColorSchemeNames, vimColorSchemeName)
	}

	if len(vimColorSchemeNames) == 0 {
		return []string{}, errors.New("no vim color schemes found")
	}

	return vimColorSchemeNames, nil
}

func GetVimColorSchemeName(fileContent *string) (string, error) {
	vimColorSchemeName := regexp.MustCompile(`let g?:?colors?_name ?= ?('|")([a-zA-Z0-9-_ \(\)]+)('|")`)

	matches := vimColorSchemeName.FindStringSubmatch(*fileContent)

	// name match is at index 2
	if len(matches) < 3 {
		return "", errors.New("no vim color scheme match")
	}

	return matches[2], nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
