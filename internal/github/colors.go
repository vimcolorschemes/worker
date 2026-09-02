package github

import (
	"context"
	"log"
	"regexp"

	gogithub "github.com/google/go-github/v68/github"
)

var colorschemeFilePattern = regexp.MustCompile(`^colors/.*\.(vim|lua)$`)

// HasColorschemeFiles reports whether the repository ships colorscheme files under
// colors/. A truncated tree resolves to true: discarding a real colorscheme costs
// more than a wasted generate attempt.
func HasColorschemeFiles(ownerName string, name string) (bool, error) {
	tree, response, err := client.Git.GetTree(context.Background(), ownerName, name, "HEAD", true)

	if _, ok := err.(*gogithub.RateLimitError); ok {
		log.Print("Hit rate limit reached")
		waitForRateLimitReset(response.Rate.Reset)
		return HasColorschemeFiles(ownerName, name)
	} else if err != nil {
		return false, err
	}

	for _, entry := range tree.Entries {
		if matchesColorschemePath(entry.GetPath()) {
			return true, nil
		}
	}

	if tree.GetTruncated() {
		log.Printf("Tree for %s/%s is truncated, assuming it ships colorschemes", ownerName, name)
		return true, nil
	}

	return false, nil
}

func matchesColorschemePath(path string) bool {
	return colorschemeFilePattern.MatchString(path)
}
