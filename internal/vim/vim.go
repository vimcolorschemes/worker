package vim

import (
	"errors"
	"log"
	"regexp"

	"github.com/vimcolorschemes/worker/internal/file"
	"github.com/vimcolorschemes/worker/internal/repository"
)

func GetVimColorSchemes(vimFileURLs []string) ([]repository.ColorScheme, error) {
	vimColorSchemes := []repository.ColorScheme{}

	for _, vimFileURL := range vimFileURLs {
		if containsURL(vimColorSchemes, vimFileURL) {
			continue
		}

		fileContent, err := file.DownloadFile(vimFileURL)
		if err != nil {
			log.Print("Error downloading file: ", vimFileURL)
			continue
		}

		if !IsVimColorScheme(&fileContent) {
			continue
		}

		vimColorSchemeName, err := GetVimColorSchemeName(&fileContent)
		if err != nil || vimColorSchemeName == "" {
			continue
		}

		log.Print("Found ", vimColorSchemeName, " at ", vimFileURL)

		vimColorSchemes = append(vimColorSchemes, repository.ColorScheme{Name: vimColorSchemeName, FileURL: vimFileURL})
	}

	if len(vimColorSchemes) == 0 {
		return []repository.ColorScheme{}, errors.New("no vim color schemes found")
	}

	return vimColorSchemes, nil
}

func IsVimColorScheme(fileContent *string) bool {
	vimNormalHighlight := regexp.MustCompile(`hi!? Normal`)
	isAVimColorScheme := vimNormalHighlight.MatchString(*fileContent)

	return isAVimColorScheme
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

func containsURL(colorSchemes []repository.ColorScheme, fileURL string) bool {
	for _, colorScheme := range colorSchemes {
		if colorScheme.FileURL == fileURL {
			return true
		}
	}

	return false
}
