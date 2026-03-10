package database

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/vimcolorschemes/worker/internal/repository"
)

// ImportData holds the fields set during an import job.
type ImportData struct {
	ID              int64
	OwnerName       string
	OwnerAvatarURL  string
	Name            string
	GithubURL       string
	GithubCreatedAt time.Time
	PushedAt        time.Time
}

// UpdateData holds the fields set during an update job.
type UpdateData struct {
	PushedAt               time.Time
	StargazersCount        int
	StargazersCountHistory []repository.StargazersCountHistoryItem
	WeekStargazersCount    int
	IsEligible             bool
	UpdatedAt              time.Time
}

// GenerateData holds the fields set during a generate job.
type GenerateData struct {
	ColorSchemes []repository.ColorScheme
	GeneratedAt  time.Time
}

// GetRepositories gets all repositories stored in the database.
func GetRepositories() ([]repository.Repository, error) {
	return queryRepositories(queryAllRepositories)
}

// GetRepository gets the repository matching the repository key.
func GetRepository(repoKey string) (repository.Repository, error) {
	matches := strings.Split(repoKey, "/")
	if len(matches) < 2 {
		return repository.Repository{}, errors.New("key not valid")
	}

	row := db.QueryRow(queryRepositoryByOwnerAndName, matches[0], matches[1])

	repo, err := scanRepositoryWithColorSchemes(row)
	if err != nil {
		return repository.Repository{}, err
	}
	return repo, nil
}

// GetRepositoriesToGenerate gets all repositories that are due for a preview generate.
func GetRepositoriesToGenerate() ([]repository.Repository, error) {
	return queryRepositories(queryRepositoriesToGenerate)
}

// UpsertRepositoryFromImport inserts or updates a repository from import data.
func UpsertRepositoryFromImport(data ImportData) {
	_, err := db.Exec(`INSERT INTO repositories (id, owner_name, owner_avatar_url, name, github_url, github_created_at, pushed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			owner_name = excluded.owner_name,
			owner_avatar_url = excluded.owner_avatar_url,
			name = excluded.name,
			github_url = excluded.github_url,
			github_created_at = excluded.github_created_at,
			pushed_at = excluded.pushed_at`,
		data.ID, data.OwnerName, data.OwnerAvatarURL, data.Name, data.GithubURL, data.GithubCreatedAt, data.PushedAt)
	if err != nil {
		log.Printf("Error upserting repository: %s", err)
		panic(err)
	}
}

// UpdateRepositoryFromUpdate updates a repository with update job data.
func UpdateRepositoryFromUpdate(id int64, data UpdateData) {
	historyJSON, err := json.Marshal(data.StargazersCountHistory)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(`UPDATE repositories SET pushed_at = ?, stargazers_count = ?, stargazers_count_history = ?, week_stargazers_count = ?, is_eligible = ?, updated_at = ? WHERE id = ?`,
		data.PushedAt, data.StargazersCount, string(historyJSON), data.WeekStargazersCount, data.IsEligible, data.UpdatedAt, id)
	if err != nil {
		log.Printf("Error updating repository: %s", err)
		panic(err)
	}
}

// UpdateRepositoryFromGenerate updates a repository with generate job data.
func UpdateRepositoryFromGenerate(id int64, data GenerateData) {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM color_schemes WHERE repository_id = ?", id)
	if err != nil {
		log.Printf("Error deleting color schemes: %s", err)
		panic(err)
	}

	for _, scheme := range data.ColorSchemes {
		result, err := tx.Exec("INSERT INTO color_schemes (repository_id, name) VALUES (?, ?)", id, scheme.Name)
		if err != nil {
			log.Printf("Error inserting color scheme: %s", err)
			panic(err)
		}
		schemeID, err := result.LastInsertId()
		if err != nil {
			panic(err)
		}
		for _, bg := range []struct {
			value  repository.BackgroundValue
			groups []repository.ColorSchemeGroup
		}{
			{repository.LightBackground, scheme.Data.Light},
			{repository.DarkBackground, scheme.Data.Dark},
		} {
			for _, group := range bg.groups {
				_, err = tx.Exec("INSERT INTO color_scheme_groups (color_scheme_id, background, name, hex_code) VALUES (?, ?, ?, ?)",
					schemeID, bg.value, group.Name, group.HexCode)
				if err != nil {
					log.Printf("Error inserting color scheme group: %s", err)
					panic(err)
				}
			}
		}
	}

	_, err = tx.Exec("UPDATE repositories SET generated_at = ? WHERE id = ?",
		data.GeneratedAt, id)
	if err != nil {
		log.Printf("Error updating repository: %s", err)
		panic(err)
	}

	if err := tx.Commit(); err != nil {
		panic(err)
	}
}
