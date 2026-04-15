package cli

import (
	"errors"
	"testing"
	"time"

	gogithub "github.com/google/go-github/v68/github"
	repoHelper "github.com/vimcolorschemes/worker/internal/repository"
)

func TestUpdateRepository(t *testing.T) {
	originalGetGithubRepository := getGithubRepository
	originalIsGithub404 := isGithub404
	t.Cleanup(func() {
		getGithubRepository = originalGetGithubRepository
		isGithub404 = originalIsGithub404
	})

	t.Run("disables repository with no commits", func(t *testing.T) {
		getGithubRepository = func(ownerName string, name string) (*gogithub.Repository, error) {
			return &gogithub.Repository{}, nil
		}

		repo, hadError, deleted := updateRepository(repoHelper.Repository{
			Owner: repoHelper.Owner{Name: "owner"},
			Name:  "repo",
		})

		if hadError {
			t.Fatal("hadError = true, want false")
		}
		if deleted {
			t.Fatal("deleted = true, want false")
		}
		if repo.IsEligible {
			t.Fatal("IsEligible = true, want false")
		}
		if !repo.IsDisabled {
			t.Fatal("IsDisabled = false, want true")
		}
	})

	t.Run("keeps repository enabled on successful update", func(t *testing.T) {
		stargazersCount := 10
		pushedAt := gogithub.Timestamp{Time: time.Now().UTC()}
		getGithubRepository = func(ownerName string, name string) (*gogithub.Repository, error) {
			return &gogithub.Repository{
				StargazersCount: &stargazersCount,
				PushedAt:        &pushedAt,
			}, nil
		}

		repo, hadError, deleted := updateRepository(repoHelper.Repository{
			Owner:           repoHelper.Owner{Name: "owner"},
			Name:            "repo",
			GithubCreatedAt: time.Now().UTC().Add(-24 * time.Hour),
		})

		if hadError {
			t.Fatal("hadError = true, want false")
		}
		if deleted {
			t.Fatal("deleted = true, want false")
		}
		if repo.IsDisabled {
			t.Fatal("IsDisabled = true, want false")
		}
		if !repo.IsEligible {
			t.Fatal("IsEligible = false, want true")
		}
	})

	t.Run("does not disable repository on non-404 fetch error", func(t *testing.T) {
		getGithubRepository = func(ownerName string, name string) (*gogithub.Repository, error) {
			return nil, errors.New("boom")
		}
		isGithub404 = func(err error) bool {
			return false
		}

		repo, hadError, deleted := updateRepository(repoHelper.Repository{
			Owner: repoHelper.Owner{Name: "owner"},
			Name:  "repo",
		})

		if !hadError {
			t.Fatal("hadError = false, want true")
		}
		if deleted {
			t.Fatal("deleted = true, want false")
		}
		if repo.IsDisabled {
			t.Fatal("IsDisabled = true, want false")
		}
		if repo.IsEligible {
			t.Fatal("IsEligible = true, want false")
		}
	})
}
