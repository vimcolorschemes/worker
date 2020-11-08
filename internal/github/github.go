package github

import (
	"context"

	"github.com/vimcolorschemes/worker/internal/dotenv"

	gogithub "github.com/google/go-github/v32/github"

	"golang.org/x/oauth2"
)

func InitGitHubClient() *gogithub.Client {
	ctx := context.Background()

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: dotenv.Get("GITHUB_TOKEN", true)},
	)

	tc := oauth2.NewClient(ctx, ts)

	return gogithub.NewClient(tc)
}
