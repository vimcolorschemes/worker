package vim

import (
	"errors"
	"log"
	"regexp"
	"strings"

	"github.com/vimcolorschemes/worker/internal/file"
	"github.com/vimcolorschemes/worker/internal/repository"
)

// GetVimColorSchemes returns vim color schemes found given a list of vim file URLs
func GetVimColorSchemes(vimFileURLs []string) ([]repository.VimColorScheme, error) {
	vimColorSchemes := []repository.VimColorScheme{}

	for _, vimFileURL := range vimFileURLs {
		if containsURL(vimColorSchemes, vimFileURL) {
			continue
		}

		fileContent, err := file.GetRemoteFileContent(vimFileURL)
		if err != nil {
			log.Print("Error downloading file: ", vimFileURL)
			continue
		}

		if !isVimColorScheme(&fileContent) {
			continue
		}

		vimColorSchemeName, err := getVimColorSchemeName(&fileContent)
		if err != nil || vimColorSchemeName == "" {
			continue
		}

		usesXtermColors := !getSupportsTermGuiColors(&fileContent)

		log.Print("Found ", vimColorSchemeName, " at ", vimFileURL)

		vimColorSchemes = append(vimColorSchemes, repository.VimColorScheme{
			Name:            vimColorSchemeName,
			FileURL:         vimFileURL,
			UsesXtermColors: usesXtermColors,
		})
	}

	if len(vimColorSchemes) == 0 {
		return []repository.VimColorScheme{}, errors.New("no vim color schemes found")
	}

	return vimColorSchemes, nil
}

func isVimColorScheme(fileContent *string) bool {
	vimNormalHighlight := regexp.MustCompile("Normal")
	isAVimColorScheme := vimNormalHighlight.MatchString(*fileContent)

	return isAVimColorScheme
}

func getVimColorSchemeName(fileContent *string) (string, error) {
	vimColorSchemeName := regexp.MustCompile(`(let g?:?|vim\.g\.)colors?_name ?= ?('|")([a-zA-Z0-9-_ \(\)]+)('|")`)

	matches := vimColorSchemeName.FindStringSubmatch(*fileContent)

	// name match is at index 3
	if len(matches) < 4 {
		return "", errors.New("no vim color scheme match")
	}

	expression := regexp.MustCompile(`[() ]`)
	cleanedName := expression.ReplaceAllString(matches[3], "")

	return strings.ToLower(cleanedName), nil
}

func getSupportsTermGuiColors(fileContent *string) bool {
	// if a gui_running check is present, xterms colors are prefered
	guiRunning := regexp.MustCompile(`if has\(.gui_running.\)`)
	if guiRunning.MatchString(*fileContent) {
		return false
	}

	// if 5 or more hex codes are found in the code, a support for termguicolors
	// is assumed
	hexCode := regexp.MustCompile(`#?[0-9a-fA-F]{6}`)
	matches := hexCode.FindAllString(*fileContent, 6)

	return len(matches) >= 5
}

func containsURL(colorSchemes []repository.VimColorScheme, fileURL string) bool {
	for _, colorScheme := range colorSchemes {
		if colorScheme.FileURL == fileURL {
			return true
		}
	}

	return false
}
