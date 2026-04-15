package database

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vimcolorschemes/worker/internal/repository"
)

func insertTestRepo(t *testing.T, id int64, ownerName, name string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO repositories (id, owner_name, name) VALUES (?, ?, ?)`, id, ownerName, name)
	if err != nil {
		t.Fatalf("insert test repo: %v", err)
	}
}

func countSearchMatches(t *testing.T, term string) int {
	t.Helper()

	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM repositories_search WHERE repositories_search MATCH ?`, term).Scan(&count)
	if err != nil {
		t.Fatalf("count search matches for %q: %v", term, err)
	}

	return count
}

func TestUpsertRepositoryFromImport(t *testing.T) {
	t.Run("inserts a new repository", func(t *testing.T) {
		setupTestDB(t)
		now := time.Now().UTC().Truncate(time.Second)
		UpsertRepositoryFromImport(ImportData{
			ID:              1,
			OwnerName:       "owner",
			OwnerAvatarURL:  "https://avatar",
			Name:            "repo",
			Description:     "A vim colorscheme repository",
			GithubURL:       "https://github.com/owner/repo",
			GithubCreatedAt: now,
			PushedAt:        now,
		})

		var id int64
		var ownerName, ownerAvatarURL, name, description, githubURL string
		err := db.QueryRow(`SELECT id, owner_name, owner_avatar_url, name, description, github_url FROM repositories WHERE id = 1`).
			Scan(&id, &ownerName, &ownerAvatarURL, &name, &description, &githubURL)
		if err != nil {
			t.Fatalf("query row: %v", err)
		}
		if id != 1 {
			t.Fatalf("id = %d, want 1", id)
		}
		if ownerName != "owner" {
			t.Fatalf("owner_name = %q, want %q", ownerName, "owner")
		}
		if ownerAvatarURL != "https://avatar" {
			t.Fatalf("owner_avatar_url = %q, want %q", ownerAvatarURL, "https://avatar")
		}
		if name != "repo" {
			t.Fatalf("name = %q, want %q", name, "repo")
		}
		if description != "A vim colorscheme repository" {
			t.Fatalf("description = %q, want %q", description, "A vim colorscheme repository")
		}
		if githubURL != "https://github.com/owner/repo" {
			t.Fatalf("github_url = %q, want %q", githubURL, "https://github.com/owner/repo")
		}
	})

	t.Run("updates existing repository on conflict", func(t *testing.T) {
		setupTestDB(t)
		now := time.Now().UTC().Truncate(time.Second)
		UpsertRepositoryFromImport(ImportData{ID: 1, OwnerName: "owner", Name: "repo", Description: "old", PushedAt: now})
		UpsertRepositoryFromImport(ImportData{ID: 1, OwnerName: "new-owner", Name: "new-repo", Description: "new", PushedAt: now})

		var ownerName, name, description string
		err := db.QueryRow(`SELECT owner_name, name, description FROM repositories WHERE id = 1`).Scan(&ownerName, &name, &description)
		if err != nil {
			t.Fatalf("query row: %v", err)
		}
		if ownerName != "new-owner" {
			t.Fatalf("owner_name = %q, want %q", ownerName, "new-owner")
		}
		if name != "new-repo" {
			t.Fatalf("name = %q, want %q", name, "new-repo")
		}
		if description != "new" {
			t.Fatalf("description = %q, want %q", description, "new")
		}
	})

	t.Run("creates an import job event", func(t *testing.T) {
		setupTestDB(t)
		now := time.Now().UTC().Truncate(time.Second)
		UpsertRepositoryFromImport(ImportData{ID: 1, OwnerName: "owner", Name: "repo", PushedAt: now})

		var eventCount int
		err := db.QueryRow(`SELECT COUNT(*) FROM repository_job_events WHERE repository_id = 1 AND job = 'import' AND status = 'success'`).Scan(&eventCount)
		if err != nil {
			t.Fatalf("query row: %v", err)
		}
		if eventCount != 1 {
			t.Fatalf("eventCount = %d, want 1", eventCount)
		}
	})

	t.Run("keeps trigram search index in sync", func(t *testing.T) {
		setupTestDB(t)
		now := time.Now().UTC().Truncate(time.Second)
		UpsertRepositoryFromImport(ImportData{ID: 1, OwnerName: "morhetz", Name: "gruvbox", Description: "retro groove", PushedAt: now})

		initialMatches := countSearchMatches(t, "gruv")
		if initialMatches != 1 {
			t.Fatalf("initialMatches = %d, want 1", initialMatches)
		}

		UpsertRepositoryFromImport(ImportData{ID: 1, OwnerName: "morhetz", Name: "tokyonight", Description: "city lights", PushedAt: now})

		oldMatches := countSearchMatches(t, "gruv")
		if oldMatches != 0 {
			t.Fatalf("oldMatches = %d, want 0", oldMatches)
		}

		newMatches := countSearchMatches(t, "night")
		if newMatches != 1 {
			t.Fatalf("newMatches = %d, want 1", newMatches)
		}
	})
}

func TestRepositoriesSearchTriggers(t *testing.T) {
	t.Run("backfills existing repositories during migration", func(t *testing.T) {
		databasePath := filepath.Join(t.TempDir(), "test.db")
		unmigratedDB, err := sql.Open("libsql", "file:"+databasePath)
		if err != nil {
			t.Fatalf("sql.Open returned error: %v", err)
		}
		defer func() {
			_ = unmigratedDB.Close()
		}()

		if _, err := unmigratedDB.Exec(`CREATE TABLE repositories (
			id INTEGER PRIMARY KEY,
			owner_name TEXT NOT NULL DEFAULT '',
			owner_avatar_url TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			github_url TEXT NOT NULL DEFAULT '',
			stargazers_count INTEGER NOT NULL DEFAULT 0,
			stargazers_count_history TEXT NOT NULL DEFAULT '[]',
			week_stargazers_count INTEGER NOT NULL DEFAULT 0,
			github_created_at DATETIME,
			pushed_at DATETIME,
			last_generate_event_at DATETIME,
			is_eligible BOOLEAN NOT NULL DEFAULT 0,
			updated_at DATETIME,
			featured_rank INTEGER,
			has_dark INTEGER NOT NULL DEFAULT 0,
			has_light INTEGER NOT NULL DEFAULT 0
		)`); err != nil {
			t.Fatalf("create repositories table: %v", err)
		}

		if _, err := unmigratedDB.Exec(`INSERT INTO repositories (id, owner_name, name, description) VALUES (1, 'morhetz', 'gruvbox', 'retro groove')`); err != nil {
			t.Fatalf("seed repositories row: %v", err)
		}

		if _, err := unmigratedDB.Exec(`CREATE TABLE goose_db_version (id INTEGER PRIMARY KEY AUTOINCREMENT, version_id bigint NOT NULL, is_applied boolean NOT NULL, tstamp timestamp DEFAULT (datetime('now')))`); err != nil {
			t.Fatalf("create goose_db_version table: %v", err)
		}

		if _, err := unmigratedDB.Exec(`INSERT INTO goose_db_version (version_id, is_applied) VALUES (1, 1)`); err != nil {
			t.Fatalf("seed goose version: %v", err)
		}

		if err := applyMigrations(unmigratedDB); err != nil {
			t.Fatalf("applyMigrations returned error: %v", err)
		}

		db = unmigratedDB
		defer func() {
			db = nil
		}()

		if got := countSearchMatches(t, "gruv"); got != 1 {
			t.Fatalf("countSearchMatches(gruv) = %d, want 1", got)
		}
	})

	t.Run("removes deleted repositories from search index", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "morhetz", "gruvbox")

		if got := countSearchMatches(t, "gruv"); got != 1 {
			t.Fatalf("countSearchMatches(gruv) before delete = %d, want 1", got)
		}

		if _, err := db.Exec(`DELETE FROM repositories WHERE id = ?`, 1); err != nil {
			t.Fatalf("delete repository: %v", err)
		}

		if got := countSearchMatches(t, "gruv"); got != 0 {
			t.Fatalf("countSearchMatches(gruv) after delete = %d, want 0", got)
		}
	})

	t.Run("ignores updates to non-search columns", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "morhetz", "gruvbox")

		if got := countSearchMatches(t, "gruv"); got != 1 {
			t.Fatalf("countSearchMatches(gruv) before non-search update = %d, want 1", got)
		}

		if _, err := db.Exec(`UPDATE repositories SET stargazers_count = ?, week_stargazers_count = ? WHERE id = ?`, 42, 7, 1); err != nil {
			t.Fatalf("update repository stats: %v", err)
		}

		if got := countSearchMatches(t, "gruv"); got != 1 {
			t.Fatalf("countSearchMatches(gruv) after non-search update = %d, want 1", got)
		}
	})
}

func TestGetRepositories(t *testing.T) {
	t.Run("returns empty slice when no repositories", func(t *testing.T) {
		setupTestDB(t)
		repos, err := GetRepositories()
		if err != nil {
			t.Fatalf("GetRepositories returned error: %v", err)
		}
		if len(repos) != 0 {
			t.Fatalf("len(repos) = %d, want 0", len(repos))
		}
	})

	t.Run("returns all repositories", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner1", "repo1")
		insertTestRepo(t, 2, "owner2", "repo2")

		repos, err := GetRepositories()
		if err != nil {
			t.Fatalf("GetRepositories returned error: %v", err)
		}
		if len(repos) != 2 {
			t.Fatalf("len(repos) = %d, want 2", len(repos))
		}
		ids := map[int64]bool{repos[0].ID: true, repos[1].ID: true}
		if !ids[1] || !ids[2] {
			t.Fatalf("expected IDs 1 and 2, got %v", ids)
		}
	})

	t.Run("excludes disabled repositories", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner1", "repo1")
		insertTestRepo(t, 2, "owner2", "repo2")

		if err := SetRepositoryDisabled(2, true); err != nil {
			t.Fatalf("SetRepositoryDisabled: %v", err)
		}

		repos, err := GetRepositories()
		if err != nil {
			t.Fatalf("GetRepositories returned error: %v", err)
		}
		if len(repos) != 1 {
			t.Fatalf("len(repos) = %d, want 1", len(repos))
		}
		if repos[0].ID != 1 {
			t.Fatalf("ID = %d, want 1", repos[0].ID)
		}
	})
}

func TestGetRepository(t *testing.T) {
	t.Run("returns the repository by owner and name", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner", "repo")

		repo, err := GetRepository("owner/repo")
		if err != nil {
			t.Fatalf("GetRepository returned error: %v", err)
		}
		if repo.ID != 1 {
			t.Fatalf("ID = %d, want 1", repo.ID)
		}
		if repo.Owner.Name != "owner" {
			t.Fatalf("Owner.Name = %q, want %q", repo.Owner.Name, "owner")
		}
		if repo.Name != "repo" {
			t.Fatalf("Name = %q, want %q", repo.Name, "repo")
		}
	})

	t.Run("is case-insensitive", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner", "repo")

		repo, err := GetRepository("OWNER/REPO")
		if err != nil {
			t.Fatalf("GetRepository returned error: %v", err)
		}
		if repo.ID != 1 {
			t.Fatalf("ID = %d, want 1", repo.ID)
		}
	})

	t.Run("returns disabled repository for direct lookup", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner", "repo")

		if err := SetRepositoryDisabled(1, true); err != nil {
			t.Fatalf("SetRepositoryDisabled: %v", err)
		}

		repo, err := GetRepository("owner/repo")
		if err != nil {
			t.Fatalf("GetRepository returned error: %v", err)
		}
		if !repo.IsDisabled {
			t.Fatal("IsDisabled = false, want true")
		}
	})

	t.Run("returns error for invalid key", func(t *testing.T) {
		setupTestDB(t)
		_, err := GetRepository("noslash")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when not found", func(t *testing.T) {
		setupTestDB(t)
		_, err := GetRepository("x/y")
		if err != sql.ErrNoRows {
			t.Fatalf("expected sql.ErrNoRows, got %v", err)
		}
	})
}

func TestGetRepositoriesToGenerate(t *testing.T) {
	insertRepoForGenerate := func(t *testing.T, id int64, isEligible int, pushedAt interface{}) {
		t.Helper()
		_, err := db.Exec(
			`INSERT INTO repositories (id, owner_name, name, is_eligible, pushed_at) VALUES (?, ?, ?, ?, ?)`,
			id, "owner", "repo", isEligible, pushedAt,
		)
		if err != nil {
			t.Fatalf("insert repo: %v", err)
		}
	}

	insertGenerateEvent := func(t *testing.T, id int64, createdAt time.Time) {
		t.Helper()
		_, err := db.Exec(
			`INSERT INTO repository_job_events (repository_id, job, created_at) VALUES (?, 'generate', ?)`,
			id,
			createdAt,
		)
		if err != nil {
			t.Fatalf("insert generate event: %v", err)
		}

		_, err = db.Exec(`UPDATE repositories SET last_generate_event_at = ? WHERE id = ?`, createdAt, id)
		if err != nil {
			t.Fatalf("update repo last_generate_event_at: %v", err)
		}
	}

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("returns repos where is_eligible=1 and pushed_at > last generate event", func(t *testing.T) {
		setupTestDB(t)
		insertRepoForGenerate(t, 1, 1, base.Add(time.Hour))
		insertGenerateEvent(t, 1, base)

		repos, err := GetRepositoriesToGenerate()
		if err != nil {
			t.Fatalf("GetRepositoriesToGenerate returned error: %v", err)
		}
		if len(repos) != 1 {
			t.Fatalf("len(repos) = %d, want 1", len(repos))
		}
		if repos[0].ID != 1 {
			t.Fatalf("ID = %d, want 1", repos[0].ID)
		}
	})

	t.Run("excludes repos where is_eligible=0", func(t *testing.T) {
		setupTestDB(t)
		insertRepoForGenerate(t, 1, 0, base.Add(time.Hour))
		insertGenerateEvent(t, 1, base)

		repos, err := GetRepositoriesToGenerate()
		if err != nil {
			t.Fatalf("GetRepositoriesToGenerate returned error: %v", err)
		}
		if len(repos) != 0 {
			t.Fatalf("len(repos) = %d, want 0", len(repos))
		}
	})

	t.Run("excludes repos where last generate event >= pushed_at", func(t *testing.T) {
		setupTestDB(t)
		insertRepoForGenerate(t, 1, 1, base)
		insertGenerateEvent(t, 1, base.Add(time.Hour))

		repos, err := GetRepositoriesToGenerate()
		if err != nil {
			t.Fatalf("GetRepositoriesToGenerate returned error: %v", err)
		}
		if len(repos) != 0 {
			t.Fatalf("len(repos) = %d, want 0", len(repos))
		}
	})

	t.Run("excludes disabled repos", func(t *testing.T) {
		setupTestDB(t)
		insertRepoForGenerate(t, 1, 1, base.Add(time.Hour))
		insertGenerateEvent(t, 1, base)

		if err := SetRepositoryDisabled(1, true); err != nil {
			t.Fatalf("SetRepositoryDisabled: %v", err)
		}

		repos, err := GetRepositoriesToGenerate()
		if err != nil {
			t.Fatalf("GetRepositoriesToGenerate returned error: %v", err)
		}
		if len(repos) != 0 {
			t.Fatalf("len(repos) = %d, want 0", len(repos))
		}
	})

	t.Run("includes repos where no generate event exists", func(t *testing.T) {
		setupTestDB(t)
		insertRepoForGenerate(t, 1, 1, base)

		repos, err := GetRepositoriesToGenerate()
		if err != nil {
			t.Fatalf("GetRepositoriesToGenerate returned error: %v", err)
		}
		if len(repos) != 1 {
			t.Fatalf("len(repos) = %d, want 1", len(repos))
		}
	})
}

func TestUpdateRepositoryFromUpdate(t *testing.T) {
	t.Run("updates all fields correctly", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner", "repo")

		UpdateRepositoryFromUpdate(1, UpdateData{
			StargazersCount:     42,
			WeekStargazersCount: 7,
			IsEligible:          true,
		})

		var stargazersCount, weekStargazersCount int
		var isEligible bool
		err := db.QueryRow(`SELECT stargazers_count, week_stargazers_count, is_eligible FROM repositories WHERE id = 1`).
			Scan(&stargazersCount, &weekStargazersCount, &isEligible)
		if err != nil {
			t.Fatalf("query row: %v", err)
		}
		if stargazersCount != 42 {
			t.Fatalf("stargazers_count = %d, want 42", stargazersCount)
		}
		if weekStargazersCount != 7 {
			t.Fatalf("week_stargazers_count = %d, want 7", weekStargazersCount)
		}
		if !isEligible {
			t.Fatal("is_eligible = false, want true")
		}
	})

	t.Run("updates disabled flag", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner", "repo")

		UpdateRepositoryFromUpdate(1, UpdateData{IsDisabled: true})

		var isDisabled bool
		err := db.QueryRow(`SELECT is_disabled FROM repositories WHERE id = 1`).Scan(&isDisabled)
		if err != nil {
			t.Fatalf("query row: %v", err)
		}
		if !isDisabled {
			t.Fatal("is_disabled = false, want true")
		}
	})

	t.Run("roundtrips stargazers_count_history JSON", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner", "repo")

		history := []repository.StargazersCountHistoryItem{
			{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), StargazersCount: 10},
			{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), StargazersCount: 20},
		}
		UpdateRepositoryFromUpdate(1, UpdateData{StargazersCountHistory: history})

		repo, err := GetRepository("owner/repo")
		if err != nil {
			t.Fatalf("GetRepository: %v", err)
		}
		if len(repo.StargazersCountHistory) != 2 {
			t.Fatalf("StargazersCountHistory len = %d, want 2", len(repo.StargazersCountHistory))
		}
		if repo.StargazersCountHistory[0].StargazersCount != 10 {
			t.Fatalf("StargazersCount[0] = %d, want 10", repo.StargazersCountHistory[0].StargazersCount)
		}
		if repo.StargazersCountHistory[1].StargazersCount != 20 {
			t.Fatalf("StargazersCount[1] = %d, want 20", repo.StargazersCountHistory[1].StargazersCount)
		}
	})

	t.Run("creates an update job event", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner", "repo")

		UpdateRepositoryFromUpdate(1, UpdateData{})

		var eventCount int
		err := db.QueryRow(`SELECT COUNT(*) FROM repository_job_events WHERE repository_id = 1 AND job = 'update' AND status = 'success'`).Scan(&eventCount)
		if err != nil {
			t.Fatalf("query row: %v", err)
		}
		if eventCount != 1 {
			t.Fatalf("eventCount = %d, want 1", eventCount)
		}
	})
}

func TestUpdateRepositoryFromGenerate(t *testing.T) {
	t.Run("saves colorschemes and groups", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner", "repo")

		UpdateRepositoryFromGenerate(1, GenerateData{
			Colorschemes: []repository.Colorscheme{
				{
					Name: "myscheme",
					Data: repository.ColorschemeData{
						Light: []repository.ColorschemeGroup{{Name: "Normal", HexCode: "#ffffff"}},
						Dark:  []repository.ColorschemeGroup{{Name: "Normal", HexCode: "#000000"}},
					},
				},
			},
		})

		repo, err := GetRepository("owner/repo")
		if err != nil {
			t.Fatalf("GetRepository: %v", err)
		}
		if len(repo.Colorschemes) != 1 {
			t.Fatalf("Colorschemes len = %d, want 1", len(repo.Colorschemes))
		}
		scheme := repo.Colorschemes[0]
		if scheme.Name != "myscheme" {
			t.Fatalf("Name = %q, want %q", scheme.Name, "myscheme")
		}
		if len(scheme.Data.Light) != 1 {
			t.Fatalf("Data.Light len = %d, want 1", len(scheme.Data.Light))
		}
		if len(scheme.Data.Dark) != 1 {
			t.Fatalf("Data.Dark len = %d, want 1", len(scheme.Data.Dark))
		}
	})

	t.Run("replaces existing colorschemes", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner", "repo")

		UpdateRepositoryFromGenerate(1, GenerateData{
			Colorschemes: []repository.Colorscheme{{Name: "scheme1"}},
		})
		UpdateRepositoryFromGenerate(1, GenerateData{
			Colorschemes: []repository.Colorscheme{{Name: "scheme2"}},
		})

		repo, err := GetRepository("owner/repo")
		if err != nil {
			t.Fatalf("GetRepository: %v", err)
		}
		if len(repo.Colorschemes) != 1 {
			t.Fatalf("Colorschemes len = %d, want 1", len(repo.Colorschemes))
		}
		if repo.Colorschemes[0].Name != "scheme2" {
			t.Fatalf("Name = %q, want %q", repo.Colorschemes[0].Name, "scheme2")
		}
	})

	t.Run("creates a generate job event", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner", "repo")

		UpdateRepositoryFromGenerate(1, GenerateData{})

		var eventCount int
		err := db.QueryRow(`SELECT COUNT(*) FROM repository_job_events WHERE repository_id = 1 AND job = 'generate' AND status = 'success'`).Scan(&eventCount)
		if err != nil {
			t.Fatalf("query row: %v", err)
		}
		if eventCount != 1 {
			t.Fatalf("eventCount = %d, want 1", eventCount)
		}
	})

	t.Run("keeps only latest success event per repo and job", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner", "repo")

		UpdateRepositoryFromGenerate(1, GenerateData{})
		UpdateRepositoryFromGenerate(1, GenerateData{})

		var eventCount int
		err := db.QueryRow(`SELECT COUNT(*) FROM repository_job_events WHERE repository_id = 1 AND job = 'generate' AND status = 'success'`).Scan(&eventCount)
		if err != nil {
			t.Fatalf("query row: %v", err)
		}
		if eventCount != 1 {
			t.Fatalf("eventCount = %d, want 1", eventCount)
		}
	})

	t.Run("stores capped error message for generate failure", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner", "repo")

		longError := strings.Repeat("x", 3000)
		err := CreateRepositoryGenerateErrorEvent(1, longError)
		if err != nil {
			t.Fatalf("CreateRepositoryGenerateErrorEvent: %v", err)
		}

		var status, errorMessage string
		err = db.QueryRow(`SELECT status, error_message FROM repository_job_events WHERE repository_id = 1 AND job = 'generate' ORDER BY id DESC LIMIT 1`).Scan(&status, &errorMessage)
		if err != nil {
			t.Fatalf("query row: %v", err)
		}
		if status != "error" {
			t.Fatalf("status = %q, want %q", status, "error")
		}
		if len(errorMessage) != 2048 {
			t.Fatalf("len(errorMessage) = %d, want 2048", len(errorMessage))
		}
	})

	t.Run("keeps only latest event per status", func(t *testing.T) {
		setupTestDB(t)
		insertTestRepo(t, 1, "owner", "repo")

		if err := CreateRepositoryGenerateErrorEvent(1, "first"); err != nil {
			t.Fatalf("CreateRepositoryGenerateErrorEvent: %v", err)
		}
		if err := CreateRepositoryGenerateErrorEvent(1, "second"); err != nil {
			t.Fatalf("CreateRepositoryGenerateErrorEvent: %v", err)
		}
		UpdateRepositoryFromGenerate(1, GenerateData{})
		UpdateRepositoryFromGenerate(1, GenerateData{})

		var errorCount int
		err := db.QueryRow(`SELECT COUNT(*) FROM repository_job_events WHERE repository_id = 1 AND job = 'generate' AND status = 'error'`).Scan(&errorCount)
		if err != nil {
			t.Fatalf("query row: %v", err)
		}
		if errorCount != 1 {
			t.Fatalf("errorCount = %d, want 1", errorCount)
		}

		var successCount int
		err = db.QueryRow(`SELECT COUNT(*) FROM repository_job_events WHERE repository_id = 1 AND job = 'generate' AND status = 'success'`).Scan(&successCount)
		if err != nil {
			t.Fatalf("query row: %v", err)
		}
		if successCount != 1 {
			t.Fatalf("successCount = %d, want 1", successCount)
		}
	})
}
