package database

import (
	"database/sql"
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
	Description     string
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
	IsDisabled             bool
	UpdatedAt              time.Time
}

// GenerateData holds the fields set during a generate job.
type GenerateData struct {
	Colorschemes []repository.Colorscheme
}

const (
	jobImport   = "import"
	jobUpdate   = "update"
	jobGenerate = "generate"

	jobStatusSuccess = "success"
	jobStatusError   = "error"

	maxJobEventErrorMessageLength = 2048
	repositoryWriteBatchSize      = 100
)

// RepositoryUpdateData pairs a repository id with the fields set during update.
type RepositoryUpdateData struct {
	ID   int64
	Data UpdateData
}

// GetRepositories gets all repositories stored in the database.
func GetRepositories() ([]repository.Repository, error) {
	return queryRepositoriesBasic(queryAllRepositories)
}

// GetRepository gets the repository matching the repository key.
func GetRepository(repoKey string) (repository.Repository, error) {
	matches := strings.Split(repoKey, "/")
	if len(matches) < 2 {
		return repository.Repository{}, errors.New("key not valid")
	}

	row := db.QueryRow(queryRepositoryByOwnerAndName, matches[0], matches[1])

	repo, err := scanRepositoryWithColorschemes(row)
	if err != nil {
		return repository.Repository{}, err
	}
	return repo, nil
}

// DeleteRepository removes a repository row by id. Foreign keys on
// colorschemes, colorscheme_groups, and repository_job_events are declared
// ON DELETE CASCADE, so SQLite removes the related rows for us.
func DeleteRepository(id int64) error {
	_, err := execWithTransientRetry("DELETE FROM repositories WHERE id = ?", id)
	return err
}

// GetRepositoriesToGenerate gets all repositories that are due for a preview generate.
func GetRepositoriesToGenerate() ([]repository.Repository, error) {
	return queryRepositoriesBasic(queryRepositoriesToGenerate)
}

// SetRepositoryDisabled updates the manual/system scheduler override flag.
func SetRepositoryDisabled(id int64, disabled bool) error {
	_, err := execWithTransientRetry("UPDATE repositories SET is_disabled = ? WHERE id = ?", disabled, id)
	return err
}

// UpsertRepositoryFromImport inserts or updates a repository from import data.
func UpsertRepositoryFromImport(data ImportData) {
	UpsertRepositoriesFromImport([]ImportData{data})
}

// UpsertRepositoriesFromImport inserts or updates repositories from import data in batches.
func UpsertRepositoriesFromImport(data []ImportData) {
	for start := 0; start < len(data); start += repositoryWriteBatchSize {
		end := min(start+repositoryWriteBatchSize, len(data))
		if err := upsertRepositoriesFromImportBatch(data[start:end]); err != nil {
			log.Printf("Error upserting repositories: %s", err)
			panic(err)
		}
	}
}

// UpdateRepositoryFromUpdate updates a repository with update job data.
func UpdateRepositoryFromUpdate(id int64, data UpdateData) {
	UpdateRepositoriesFromUpdate([]RepositoryUpdateData{{ID: id, Data: data}})
}

// UpdateRepositoriesFromUpdate updates repositories with update job data in batches.
func UpdateRepositoriesFromUpdate(updates []RepositoryUpdateData) {
	for start := 0; start < len(updates); start += repositoryWriteBatchSize {
		end := min(start+repositoryWriteBatchSize, len(updates))
		if err := updateRepositoriesFromUpdateBatch(updates[start:end]); err != nil {
			log.Printf("Error updating repositories: %s", err)
			panic(err)
		}
	}
}

func upsertRepositoriesFromImportBatch(data []ImportData) error {
	if len(data) == 0 {
		return nil
	}

	return runWithTransientRetry("import repository batch", func() error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer func() {
			_ = tx.Rollback()
		}()

		values := buildImportRepositoryBatchValues(data)

		_, err = tx.Exec(`INSERT INTO repositories (id, owner_name, owner_avatar_url, name, description, github_url, github_created_at, pushed_at)
			VALUES `+values.rowPlaceholders+`
			ON CONFLICT(id) DO UPDATE SET
				owner_name = excluded.owner_name,
				owner_avatar_url = excluded.owner_avatar_url,
				name = excluded.name,
				description = excluded.description,
				github_url = excluded.github_url,
				github_created_at = excluded.github_created_at,
				pushed_at = excluded.pushed_at`,
			values.args...)
		if err != nil {
			return err
		}

		if err := createRepositoryJobEvents(tx, values.repositoryIDs, jobImport, jobStatusSuccess, "", time.Now().UTC()); err != nil {
			return err
		}

		return tx.Commit()
	})
}

func updateRepositoriesFromUpdateBatch(updates []RepositoryUpdateData) error {
	if len(updates) == 0 {
		return nil
	}

	return runWithTransientRetry("update repository batch", func() error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer func() {
			_ = tx.Rollback()
		}()

		values, err := buildUpdateRepositoryBatchValues(updates)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`WITH updates(id, pushed_at, stargazers_count, stargazers_count_history, week_stargazers_count, is_eligible, is_disabled, updated_at) AS (
				VALUES `+values.rowPlaceholders+`
			)
			UPDATE repositories SET
				pushed_at = (SELECT pushed_at FROM updates WHERE updates.id = repositories.id),
				stargazers_count = (SELECT stargazers_count FROM updates WHERE updates.id = repositories.id),
				stargazers_count_history = (SELECT stargazers_count_history FROM updates WHERE updates.id = repositories.id),
				week_stargazers_count = (SELECT week_stargazers_count FROM updates WHERE updates.id = repositories.id),
				is_eligible = (SELECT is_eligible FROM updates WHERE updates.id = repositories.id),
				is_disabled = (SELECT is_disabled FROM updates WHERE updates.id = repositories.id),
				updated_at = (SELECT updated_at FROM updates WHERE updates.id = repositories.id)
			WHERE id IN (SELECT id FROM updates)`,
			values.args...)
		if err != nil {
			return err
		}

		if err := createRepositoryJobEvents(tx, values.repositoryIDs, jobUpdate, jobStatusSuccess, "", time.Now().UTC()); err != nil {
			return err
		}

		return tx.Commit()
	})
}

type repositoryBatchValues struct {
	rowPlaceholders string
	args            []any
	repositoryIDs   []int64
}

func buildImportRepositoryBatchValues(data []ImportData) repositoryBatchValues {
	values := repositoryBatchValues{
		rowPlaceholders: rowPlaceholders(len(data), 8),
		args:            make([]any, 0, len(data)*8),
		repositoryIDs:   make([]int64, 0, len(data)),
	}

	for _, item := range data {
		values.args = append(values.args,
			item.ID,
			item.OwnerName,
			item.OwnerAvatarURL,
			item.Name,
			item.Description,
			item.GithubURL,
			item.GithubCreatedAt,
			item.PushedAt,
		)
		values.repositoryIDs = append(values.repositoryIDs, item.ID)
	}

	return values
}

func buildUpdateRepositoryBatchValues(updates []RepositoryUpdateData) (repositoryBatchValues, error) {
	values := repositoryBatchValues{
		rowPlaceholders: rowPlaceholders(len(updates), 8),
		args:            make([]any, 0, len(updates)*8),
		repositoryIDs:   make([]int64, 0, len(updates)),
	}

	for _, update := range updates {
		historyJSON, err := json.Marshal(update.Data.StargazersCountHistory)
		if err != nil {
			return repositoryBatchValues{}, err
		}

		values.args = append(values.args,
			update.ID,
			update.Data.PushedAt,
			update.Data.StargazersCount,
			string(historyJSON),
			update.Data.WeekStargazersCount,
			update.Data.IsEligible,
			update.Data.IsDisabled,
			update.Data.UpdatedAt,
		)
		values.repositoryIDs = append(values.repositoryIDs, update.ID)
	}

	return values, nil
}

func rowPlaceholders(rowCount int, columnCount int) string {
	row := "(" + placeholders(columnCount) + ")"
	rows := make([]string, rowCount)
	for index := range rows {
		rows[index] = row
	}
	return strings.Join(rows, ", ")
}

func placeholders(count int) string {
	items := make([]string, count)
	for index := range items {
		items[index] = "?"
	}
	return strings.Join(items, ", ")
}

// UpdateRepositoryFromGenerate updates a repository with generate job data.
func UpdateRepositoryFromGenerate(id int64, data GenerateData) {
	tx, err := beginWithTransientRetry()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.Exec("DELETE FROM colorschemes WHERE repository_id = ?", id)
	if err != nil {
		log.Printf("Error deleting colorschemes: %s", err)
		panic(err)
	}

	hasDark, hasLight := 0, 0
	for _, scheme := range data.Colorschemes {
		if len(scheme.Data.Dark) > 0 {
			hasDark = 1
		}
		if len(scheme.Data.Light) > 0 {
			hasLight = 1
		}
	}
	_, err = tx.Exec("UPDATE repositories SET has_dark = ?, has_light = ? WHERE id = ?", hasDark, hasLight, id)
	if err != nil {
		log.Printf("Error updating has_dark/has_light: %s", err)
		panic(err)
	}

	for _, scheme := range data.Colorschemes {
		result, err := tx.Exec("INSERT INTO colorschemes (repository_id, name) VALUES (?, ?)", id, scheme.Name)
		if err != nil {
			log.Printf("Error inserting colorscheme: %s", err)
			panic(err)
		}
		schemeID, err := result.LastInsertId()
		if err != nil {
			panic(err)
		}
		for _, bg := range []struct {
			value  repository.BackgroundValue
			groups []repository.ColorschemeGroup
		}{
			{repository.LightBackground, scheme.Data.Light},
			{repository.DarkBackground, scheme.Data.Dark},
		} {
			for _, group := range bg.groups {
				_, err = tx.Exec(`
					INSERT INTO colorscheme_groups (
						colorscheme_id,
						background,
						name,
						hex_code,
						bold,
						italic,
						underline,
						undercurl,
						underdouble,
						underdotted,
						underdashed,
						strikethrough,
						reverse
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					schemeID,
					bg.value,
					group.Name,
					group.HexCode,
					group.Bold,
					group.Italic,
					group.Underline,
					group.Undercurl,
					group.Underdouble,
					group.Underdotted,
					group.Underdashed,
					group.Strikethrough,
					group.Reverse,
				)
				if err != nil {
					log.Printf("Error inserting colorscheme group: %s", err)
					panic(err)
				}
			}
		}
	}

	err = createRepositoryJobEvent(tx, id, jobGenerate, jobStatusSuccess, "", time.Now().UTC())
	if err != nil {
		log.Printf("Error creating repository job event: %s", err)
		panic(err)
	}

	if err := tx.Commit(); err != nil {
		panic(err)
	}
}

type repositoryJobEventExecutor interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// CreateRepositoryGenerateErrorEvent stores a failed generate attempt for a repository.
func CreateRepositoryGenerateErrorEvent(repositoryID int64, errorMessage string) error {
	return createRepositoryJobEvent(db, repositoryID, jobGenerate, jobStatusError, errorMessage, time.Now().UTC())
}

func createRepositoryJobEvent(exec repositoryJobEventExecutor, repositoryID int64, job string, status string, errorMessage string, createdAt time.Time) error {
	if execDB, ok := exec.(*sql.DB); ok && execDB == db {
		return runWithTransientRetry("repository job event", func() error {
			return createRepositoryJobEventOnce(execDB, repositoryID, job, status, errorMessage, createdAt)
		})
	}

	return createRepositoryJobEventOnce(exec, repositoryID, job, status, errorMessage, createdAt)
}

func createRepositoryJobEvents(exec repositoryJobEventExecutor, repositoryIDs []int64, job string, status string, errorMessage string, createdAt time.Time) error {
	if execDB, ok := exec.(*sql.DB); ok && execDB == db {
		return runWithTransientRetry("repository job events", func() error {
			return createRepositoryJobEventsOnce(execDB, repositoryIDs, job, status, errorMessage, createdAt)
		})
	}

	return createRepositoryJobEventsOnce(exec, repositoryIDs, job, status, errorMessage, createdAt)
}

func createRepositoryJobEventsOnce(exec repositoryJobEventExecutor, repositoryIDs []int64, job string, status string, errorMessage string, createdAt time.Time) error {
	if len(repositoryIDs) == 0 {
		return nil
	}

	if len(repositoryIDs) == 1 {
		return createRepositoryJobEventOnce(exec, repositoryIDs[0], job, status, errorMessage, createdAt)
	}

	trimmedErrorMessage := errorMessage
	if len(trimmedErrorMessage) > maxJobEventErrorMessageLength {
		trimmedErrorMessage = trimmedErrorMessage[:maxJobEventErrorMessageLength]
	}

	idArgs := make([]any, 0, len(repositoryIDs))
	for _, repositoryID := range repositoryIDs {
		idArgs = append(idArgs, repositoryID)
	}
	idPlaceholders := placeholders(len(repositoryIDs))

	if job == jobGenerate {
		args := append([]any{createdAt}, idArgs...)
		_, err := exec.Exec(
			"UPDATE repositories SET last_generate_event_at = ? WHERE id IN ("+idPlaceholders+")",
			args...,
		)
		if err != nil {
			return err
		}
	}

	insertArgs := make([]any, 0, len(repositoryIDs)*5)
	for _, repositoryID := range repositoryIDs {
		insertArgs = append(insertArgs, repositoryID, job, status, trimmedErrorMessage, createdAt)
	}

	_, err := exec.Exec(
		"INSERT INTO repository_job_events (repository_id, job, status, error_message, created_at) VALUES "+rowPlaceholders(len(repositoryIDs), 5),
		insertArgs...,
	)
	return err
}

func createRepositoryJobEventOnce(exec repositoryJobEventExecutor, repositoryID int64, job string, status string, errorMessage string, createdAt time.Time) error {
	trimmedErrorMessage := errorMessage
	if len(trimmedErrorMessage) > maxJobEventErrorMessageLength {
		trimmedErrorMessage = trimmedErrorMessage[:maxJobEventErrorMessageLength]
	}

	if job == jobGenerate {
		_, err := exec.Exec(
			"UPDATE repositories SET last_generate_event_at = ? WHERE id = ?",
			createdAt,
			repositoryID,
		)
		if err != nil {
			return err
		}
	}

	_, err := exec.Exec(
		"INSERT INTO repository_job_events (repository_id, job, status, error_message, created_at) VALUES (?, ?, ?, ?, ?)",
		repositoryID,
		job,
		status,
		trimmedErrorMessage,
		createdAt,
	)
	return err
}
