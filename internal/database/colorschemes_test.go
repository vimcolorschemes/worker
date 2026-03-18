package database

import (
	"testing"
	"time"

	"github.com/vimcolorschemes/worker/internal/repository"
)

func TestLoadColorschemes(t *testing.T) {
	t.Run("returns empty slice for repo with no schemes", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner", "repo")

		schemes, err := loadColorschemes(1)
		if err != nil {
			t.Fatalf("loadColorschemes: %v", err)
		}
		if len(schemes) != 0 {
			t.Fatalf("len(schemes) = %d, want 0", len(schemes))
		}
	})

	t.Run("returns schemes with light and dark groups", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner", "repo")
		if _, err := db.Exec(`INSERT INTO colorschemes (id, repository_id, name) VALUES (1, 1, 'myscheme')`); err != nil {
			t.Fatalf("insert colorscheme: %v", err)
		}
		if _, err := db.Exec(`INSERT INTO colorscheme_groups (colorscheme_id, background, name, hex_code) VALUES (1, 'light', 'Normal', '#fff')`); err != nil {
			t.Fatalf("insert light colorscheme_group: %v", err)
		}
		if _, err := db.Exec(`INSERT INTO colorscheme_groups (colorscheme_id, background, name, hex_code) VALUES (1, 'dark', 'Normal', '#000')`); err != nil {
			t.Fatalf("insert dark colorscheme_group: %v", err)
		}

		schemes, err := loadColorschemes(1)
		if err != nil {
			t.Fatalf("loadColorschemes: %v", err)
		}
		if len(schemes) != 1 {
			t.Fatalf("len(schemes) = %d, want 1", len(schemes))
		}
		if schemes[0].Name != "myscheme" {
			t.Fatalf("Name = %q, want %q", schemes[0].Name, "myscheme")
		}
		if len(schemes[0].Data.Light) != 1 {
			t.Fatalf("Data.Light len = %d, want 1", len(schemes[0].Data.Light))
		}
		if len(schemes[0].Data.Dark) != 1 {
			t.Fatalf("Data.Dark len = %d, want 1", len(schemes[0].Data.Dark))
		}
	})

	t.Run("derives backgrounds from groups", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner", "repo")
		if _, err := db.Exec(`INSERT INTO colorschemes (id, repository_id, name) VALUES (1, 1, 'darkonly')`); err != nil {
			t.Fatalf("insert colorscheme: %v", err)
		}
		if _, err := db.Exec(`INSERT INTO colorscheme_groups (colorscheme_id, background, name, hex_code) VALUES (1, 'dark', 'Normal', '#000')`); err != nil {
			t.Fatalf("insert colorscheme_group: %v", err)
		}

		schemes, err := loadColorschemes(1)
		if err != nil {
			t.Fatalf("loadColorschemes: %v", err)
		}
		if len(schemes[0].Backgrounds) != 1 {
			t.Fatalf("Backgrounds len = %d, want 1", len(schemes[0].Backgrounds))
		}
		if schemes[0].Backgrounds[0] != repository.DarkBackground {
			t.Fatalf("Backgrounds[0] = %q, want %q", schemes[0].Backgrounds[0], repository.DarkBackground)
		}
	})

	t.Run("handles scheme with no groups", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner", "repo")
		if _, err := db.Exec(`INSERT INTO colorschemes (id, repository_id, name) VALUES (1, 1, 'nogroups')`); err != nil {
			t.Fatalf("insert colorscheme: %v", err)
		}

		schemes, err := loadColorschemes(1)
		if err != nil {
			t.Fatalf("loadColorschemes: %v", err)
		}
		if len(schemes) != 1 {
			t.Fatalf("len(schemes) = %d, want 1", len(schemes))
		}
		if len(schemes[0].Backgrounds) != 0 {
			t.Fatalf("Backgrounds len = %d, want 0", len(schemes[0].Backgrounds))
		}
		if schemes[0].Data.Light != nil {
			t.Fatal("Data.Light should be nil")
		}
		if schemes[0].Data.Dark != nil {
			t.Fatal("Data.Dark should be nil")
		}
	})
}

func TestScanRepository(t *testing.T) {
	t.Run("populates all nullable time fields", func(t *testing.T) {
		setupTestDB(t)
		now := time.Now().UTC().Truncate(time.Second)
		_, err := db.Exec(
			`INSERT INTO repositories (id, owner_name, name, github_created_at, pushed_at, updated_at) VALUES (1, 'owner', 'repo', ?, ?, ?)`,
			now, now, now,
		)
		if err != nil {
			t.Fatalf("insert repo: %v", err)
		}

		row := db.QueryRow("SELECT " + repositorySelectColumns + " FROM repositories WHERE id = 1")
		repo, err := scanRepository(row)
		if err != nil {
			t.Fatalf("scanRepository: %v", err)
		}
		if repo.GithubCreatedAt.IsZero() {
			t.Fatal("GithubCreatedAt should not be zero")
		}
		if repo.PushedAt.IsZero() {
			t.Fatal("PushedAt should not be zero")
		}
		if repo.UpdatedAt.IsZero() {
			t.Fatal("UpdatedAt should not be zero")
		}
	})

	t.Run("handles NULL time fields", func(t *testing.T) {
		setupTestDB(t)
		_, err := db.Exec(`INSERT INTO repositories (id, owner_name, name) VALUES (1, 'owner', 'repo')`)
		if err != nil {
			t.Fatalf("insert repo: %v", err)
		}

		row := db.QueryRow("SELECT " + repositorySelectColumns + " FROM repositories WHERE id = 1")
		repo, err := scanRepository(row)
		if err != nil {
			t.Fatalf("scanRepository: %v", err)
		}
		if !repo.GithubCreatedAt.IsZero() {
			t.Fatalf("GithubCreatedAt should be zero, got %v", repo.GithubCreatedAt)
		}
		if !repo.PushedAt.IsZero() {
			t.Fatalf("PushedAt should be zero, got %v", repo.PushedAt)
		}
		if !repo.UpdatedAt.IsZero() {
			t.Fatalf("UpdatedAt should be zero, got %v", repo.UpdatedAt)
		}
	})
}
