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

		log.Print("Found ", vimColorSchemeName, " at ", vimFileURL)

		vimColorSchemes = append(vimColorSchemes, repository.VimColorScheme{
			Name:    vimColorSchemeName,
			FileURL: vimFileURL,
		})
	}

	if len(vimColorSchemes) == 0 {
		return []repository.VimColorScheme{}, errors.New("no vim color schemes found")
	}

	return vimColorSchemes, nil
}

// NormalizeVimColorSchemeColors fixes issues with colors such as background
// and foreground being the same color for a color group
func NormalizeVimColorSchemeColors(colors []repository.VimColorSchemeGroup) []repository.VimColorSchemeGroup {
	colorMap := make(map[string]string)
	for i := 0; i < len(colors); i++ {
		colorMap[colors[i].Name] = colors[i].HexCode
	}

	normalizeStatusLineColors(&colorMap)

	result := make([]repository.VimColorSchemeGroup, 0, len(colors))

	for _, color := range colors {
		result = append(result, repository.VimColorSchemeGroup{
			Name:    color.Name,
			HexCode: colorMap[color.Name],
		})
	}

	return result
}

func normalizeStatusLineColors(colorMap *map[string]string) {
	if (*colorMap)["StatusLineBg"] != (*colorMap)["StatusLineFg"] {
		return
	}

	if (*colorMap)["NormalBg"] != "" {
		(*colorMap)["StatusLineBg"] = (*colorMap)["NormalBg"]
	}

	if (*colorMap)["NormalFg"] != "" {
		(*colorMap)["StatusLineFg"] = (*colorMap)["NormalFg"]
	}
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

func containsURL(colorSchemes []repository.VimColorScheme, fileURL string) bool {
	for _, colorScheme := range colorSchemes {
		if colorScheme.FileURL == fileURL {
			return true
		}
	}

	return false
}
