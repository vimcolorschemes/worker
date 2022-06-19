package vim

import (
	"errors"
	"log"
	"regexp"
	"strings"

	gogithub "github.com/google/go-github/v32/github"
	"github.com/vimcolorschemes/worker/internal/file"
	"github.com/vimcolorschemes/worker/internal/github"
	"github.com/vimcolorschemes/worker/internal/repository"
)

// GetVimColorSchemes returns vim color schemes found given a list of vim file URLs
func GetVimColorSchemes(githubRepository *gogithub.Repository, vimFiles []*gogithub.RepositoryContent) ([]repository.VimColorScheme, error) {
	vimColorSchemes := []repository.VimColorScheme{}

	for _, vimFile := range vimFiles {
		downloadURL := vimFile.GetDownloadURL()

		if downloadURL == "" {
			continue
		}

		if containsURL(vimColorSchemes, downloadURL) {
			continue
		}

		fileContent, err := file.GetRemoteFileContent(downloadURL)
		if err != nil {
			log.Print("Error downloading file: ", downloadURL)
			continue
		}

		vimColorSchemeName, isLua, err := getColorSchemeName(&fileContent)
		if err != nil || vimColorSchemeName == "" {
			continue
		}

		log.Print("Found ", vimColorSchemeName, " at ", downloadURL)

		if isLua {
			log.Print(vimColorSchemeName, " is a lua color scheme")
		}

		lastCommitAt := github.GetFileLastCommitAt(githubRepository, vimFile)

		vimColorSchemes = append(vimColorSchemes,
			repository.VimColorScheme{
				Name:         vimColorSchemeName,
				FileURL:      downloadURL,
				IsLua:        isLua,
				LastCommitAt: lastCommitAt,
			},
		)
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

func getColorSchemeName(fileContent *string) (string, bool, error) {
	luaName, err := getLuaColorSchemeName(fileContent)
	if luaName != "" && err == nil {
		return luaName, true, nil
	}

	vimName, err := getVimColorSchemeName(fileContent)
	if err != nil {
		return "", false, err
	}

	return vimName, false, nil

}

func getVimColorSchemeName(fileContent *string) (string, error) {
	vimColorSchemeName := regexp.MustCompile(`(let g?:?|vim\.g\.)colors?_name ?= ?['"]([a-zA-Z0-9-_ \(\)]+)['"]`)

	matches := vimColorSchemeName.FindStringSubmatch(*fileContent)

	// name match is at index 2
	if len(matches) < 3 {
		return "", errors.New("no vim color scheme match")
	}

	expression := regexp.MustCompile(`[() ]`)
	cleanedName := expression.ReplaceAllString(matches[2], "")

	return strings.ToLower(cleanedName), nil
}

func getLuaColorSchemeName(fileContent *string) (string, error) {
	lua := regexp.MustCompile("lua")
	if !lua.MatchString(*fileContent) {
		return "", errors.New("No lua mentions")
	}

	vimColorSchemeName := regexp.MustCompile(`require\(['"]([a-zA-Z0-9-_ \(\)]+)['"]\)`)
	matches := vimColorSchemeName.FindStringSubmatch(*fileContent)

	// name match is at index 1
	if len(matches) < 2 {
		return "", errors.New("no lua color scheme match")
	}

	return matches[1], nil
}

func containsURL(colorSchemes []repository.VimColorScheme, fileURL string) bool {
	for _, colorScheme := range colorSchemes {
		if colorScheme.FileURL == fileURL {
			return true
		}
	}

	return false
}
