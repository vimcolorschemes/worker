package cli

import (
	"sort"
	"strings"

	repoHelper "github.com/vimcolorschemes/worker/internal/repository"
)

// Ordered colorscheme construction for the generate job.
//
// Color data is decoded into a Go map, so ranging over it yields random order.
// That order becomes cs.id insertion order in the database (ORDER BY cs.id on read),
// which is what the frontend lists. Sorting here with the repository's main
// scheme first keeps that listing deterministic.
func buildRepositoryColorschemes(data map[string]repoHelper.ColorschemeData, repoName string) []repoHelper.Colorscheme {
	names := make([]string, 0, len(data))
	for name := range data {
		names = append(names, name)
	}

	orderedColorschemeNames(names, repoName)

	colorschemes := make([]repoHelper.Colorscheme, 0, len(names))
	for _, name := range names {
		var backgrounds []repoHelper.BackgroundValue
		if data[name].Light != nil {
			backgrounds = append(backgrounds, repoHelper.LightBackground)
		}
		if data[name].Dark != nil {
			backgrounds = append(backgrounds, repoHelper.DarkBackground)
		}

		colorschemes = append(colorschemes, repoHelper.Colorscheme{
			Name:        name,
			Data:        data[name],
			Backgrounds: backgrounds,
		})
	}

	return colorschemes
}

// Sorts names in place: exact repo-name match first, rest alphabetical.
// The match is case-insensitive and ignores common plugin affixes so
// tokyonight.nvim still promotes tokyonight.
func orderedColorschemeNames(names []string, repoName string) {
	main := canonicalRepositoryName(repoName)
	raw := strings.ToLower(strings.TrimSpace(repoName))

	sort.Slice(names, func(i, j int) bool {
		iMain := isMainName(names[i], raw, main)
		jMain := isMainName(names[j], raw, main)
		if iMain != jMain {
			return iMain
		}
		return names[i] < names[j]
	})
}

func isMainName(schemeName, rawRepoName, canonicalRepoName string) bool {
	scheme := strings.ToLower(strings.TrimSpace(schemeName))
	if scheme == "" {
		return false
	}
	if scheme == rawRepoName {
		return true
	}
	return canonicalRepoName != "" && scheme == canonicalRepoName
}

// Normalizes a repository name for main-scheme comparison. Only the repo side
// is normalized: normalizing scheme names too would promote foo-theme when the
// repo is just foo.
func canonicalRepositoryName(name string) string {
	canonical := strings.ToLower(strings.TrimSpace(name))
	for {
		stripped := stripOneAffix(canonical)
		if stripped == canonical || stripped == "" {
			return canonical
		}
		canonical = stripped
	}
}

func stripOneAffix(name string) string {
	for _, suffix := range repositoryNameSuffixes {
		if strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, suffix)
		}
	}
	for _, prefix := range repositoryNamePrefixes {
		if strings.HasPrefix(name, prefix) {
			return strings.TrimPrefix(name, prefix)
		}
	}
	return name
}

var repositoryNameSuffixes = []string{
	".nvim", "-nvim", "_nvim",
	".vim", "-vim", "_vim",
	".neovim", "-neovim", "_neovim",
	"-themes", "-theme",
	"-colorschemes", "-colorscheme",
	"-colourschemes", "-colourscheme",
	"_colorschemes", "_colorscheme",
	"_colourschemes", "_colourscheme",
	"-color-scheme", "-colour-scheme",
	"-colors", "-colours",
}

var repositoryNamePrefixes = []string{
	"nvim-", "nvim_",
	"vim-", "vim_",
	"neovim-", "neovim_",
}
