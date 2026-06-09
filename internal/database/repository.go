package database

import (
	"context"
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
	repositoryWriteBatchSize      = 25
	repositoryWriteLogInterval    = 100
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

// UpsertRepositoriesFromImport inserts or updates repositories from import data.
func UpsertRepositoriesFromImport(data []ImportData) {
	if len(data) == 0 {
		return
	}

	startedAt := time.Now()
	eventCreatedAt := startedAt.UTC()
	log.Printf("Writing %d imported repositories to database", len(data))
	for start := 0; start < len(data); start += repositoryWriteBatchSize {
		end := min(start+repositoryWriteBatchSize, len(data))
		if err := upsertRepositoriesFromImportAdaptive(data[start:end], eventCreatedAt); err != nil {
			log.Printf("Error upserting repositories: %s", err)
			panic(err)
		}
		logRepositoryWriteProgress("imported repositories", end, len(data))
	}
	log.Printf("Finished writing %d imported repositories in %s", len(data), time.Since(startedAt).Round(time.Millisecond))
}

// UpdateRepositoryFromUpdate updates a repository with update job data.
func UpdateRepositoryFromUpdate(id int64, data UpdateData) {
	UpdateRepositoriesFromUpdate([]RepositoryUpdateData{{ID: id, Data: data}})
}

// UpdateRepositoriesFromUpdate updates repositories with update job data.
func UpdateRepositoriesFromUpdate(updates []RepositoryUpdateData) {
	if len(updates) == 0 {
		return
	}

	startedAt := time.Now()
	eventCreatedAt := startedAt.UTC()
	log.Printf("Writing %d repository updates to database", len(updates))
	for start := 0; start < len(updates); start += repositoryWriteBatchSize {
		end := min(start+repositoryWriteBatchSize, len(updates))
		if err := updateRepositoriesFromUpdateAdaptive(updates[start:end], eventCreatedAt); err != nil {
			log.Printf("Error updating repositories: %s", err)
			panic(err)
		}
		logRepositoryWriteProgress("repository updates", end, len(updates))
	}
	log.Printf("Finished writing %d repository updates in %s", len(updates), time.Since(startedAt).Round(time.Millisecond))
}

func logRepositoryWriteProgress(label string, completed int, total int) {
	if total == 0 {
		return
	}
	if completed == total || completed%repositoryWriteLogInterval == 0 {
		log.Printf("Wrote %d/%d %s", completed, total, label)
	}
}

func upsertRepositoryFromImportOnce(data ImportData, eventCreatedAt time.Time) error {
	return upsertRepositoriesFromImportBatch([]ImportData{data}, eventCreatedAt)
}

func upsertRepositoriesFromImportAdaptive(data []ImportData, eventCreatedAt time.Time) error {
	if len(data) == 0 {
		return nil
	}

	err := upsertRepositoriesFromImportBatch(data, eventCreatedAt)
	if err == nil {
		return nil
	}

	if len(data) == 1 {
		log.Printf("Error upserting repository batch, falling back to single write: %s", err)
		return upsertRepositoryFromImportOnce(data[0], eventCreatedAt)
	}

	midpoint := len(data) / 2
	log.Printf("Error upserting repository batch of %d, retrying as %d and %d: %s", len(data), midpoint, len(data)-midpoint, err)
	if err := upsertRepositoriesFromImportAdaptive(data[:midpoint], eventCreatedAt); err != nil {
		return err
	}
	return upsertRepositoriesFromImportAdaptive(data[midpoint:], eventCreatedAt)
}

func upsertRepositoriesFromImportBatch(data []ImportData, eventCreatedAt time.Time) error {
	if len(data) == 0 {
		return nil
	}

	values := buildImportRepositoryBatchValues(data)

	return runRepositoryWriteTransaction(func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO repositories (id, owner_name, owner_avatar_url, name, description, github_url, github_created_at, pushed_at)
			VALUES `+values.rowPlaceholders+`
			ON CONFLICT(id) DO UPDATE SET
				owner_name = excluded.owner_name,
				owner_avatar_url = excluded.owner_avatar_url,
				name = excluded.name,
				description = excluded.description,
				github_url = excluded.github_url,
				github_created_at = excluded.github_created_at,
				pushed_at = excluded.pushed_at
			WHERE
				repositories.owner_name IS NOT excluded.owner_name OR
				repositories.owner_avatar_url IS NOT excluded.owner_avatar_url OR
				repositories.name IS NOT excluded.name OR
				repositories.description IS NOT excluded.description OR
				repositories.github_url IS NOT excluded.github_url OR
				repositories.github_created_at IS NOT excluded.github_created_at OR
				repositories.pushed_at IS NOT excluded.pushed_at`,
			values.args...)
		if err != nil {
			return err
		}

		return createRepositoryJobEventsContext(ctx, tx, values.repositoryIDs, jobImport, jobStatusSuccess, "", eventCreatedAt)
	})
}

func updateRepositoryFromUpdateOnce(update RepositoryUpdateData, eventCreatedAt time.Time) error {
	return updateRepositoriesFromUpdateBatch([]RepositoryUpdateData{update}, eventCreatedAt)
}

func updateRepositoriesFromUpdateAdaptive(updates []RepositoryUpdateData, eventCreatedAt time.Time) error {
	if len(updates) == 0 {
		return nil
	}

	err := updateRepositoriesFromUpdateBatch(updates, eventCreatedAt)
	if err == nil {
		return nil
	}

	if len(updates) == 1 {
		log.Printf("Error updating repository batch, falling back to single write: %s", err)
		return updateRepositoryFromUpdateOnce(updates[0], eventCreatedAt)
	}

	midpoint := len(updates) / 2
	log.Printf("Error updating repository batch of %d, retrying as %d and %d: %s", len(updates), midpoint, len(updates)-midpoint, err)
	if err := updateRepositoriesFromUpdateAdaptive(updates[:midpoint], eventCreatedAt); err != nil {
		return err
	}
	return updateRepositoriesFromUpdateAdaptive(updates[midpoint:], eventCreatedAt)
}

func updateRepositoriesFromUpdateBatch(updates []RepositoryUpdateData, eventCreatedAt time.Time) error {
	if len(updates) == 0 {
		return nil
	}

	values, err := buildUpdateRepositoryBatchValues(updates)
	if err != nil {
		return err
	}

	return runRepositoryWriteTransaction(func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `WITH updates(id, pushed_at, stargazers_count, stargazers_count_history, week_stargazers_count, is_eligible, is_disabled, updated_at) AS (
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

		return createRepositoryJobEventsContext(ctx, tx, values.repositoryIDs, jobUpdate, jobStatusSuccess, "", eventCreatedAt)
	})
}

func runRepositoryWriteTransaction(run func(context.Context, *sql.Tx) error) error {
	return runWithTransientRetry("repository write transaction", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
		defer cancel()

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()

		if err := run(ctx, tx); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}

		committed = true
		return nil
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

type repositoryJobEventContextExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// CreateRepositoryGenerateErrorEvent stores a failed generate attempt for a repository.
func CreateRepositoryGenerateErrorEvent(repositoryID int64, errorMessage string) error {
	return createRepositoryJobEvent(db, repositoryID, jobGenerate, jobStatusError, errorMessage, time.Now().UTC())
}

func createRepositoryJobEvent(exec repositoryJobEventExecutor, repositoryID int64, job string, status string, errorMessage string, createdAt time.Time) error {
	if execDB, ok := exec.(*sql.DB); ok && execDB == db {
		return createRepositoryJobEventWithRetry(repositoryID, job, status, errorMessage, createdAt)
	}

	return createRepositoryJobEventOnce(exec, repositoryID, job, status, errorMessage, createdAt)
}

func createRepositoryJobEventWithRetry(repositoryID int64, job string, status string, errorMessage string, createdAt time.Time) error {
	return runRepositoryWriteTransaction(func(ctx context.Context, tx *sql.Tx) error {
		return createRepositoryJobEventsContext(ctx, tx, []int64{repositoryID}, job, status, errorMessage, createdAt)
	})
}

func createRepositoryJobEvents(exec repositoryJobEventExecutor, repositoryIDs []int64, job string, status string, errorMessage string, createdAt time.Time) error {
	if execDB, ok := exec.(*sql.DB); ok && execDB == db {
		return runWithTransientRetry("repository job events", func() error {
			return createRepositoryJobEventsOnce(execDB, repositoryIDs, job, status, errorMessage, createdAt)
		})
	}

	return createRepositoryJobEventsOnce(exec, repositoryIDs, job, status, errorMessage, createdAt)
}

func createRepositoryJobEventsContext(ctx context.Context, exec repositoryJobEventContextExecutor, repositoryIDs []int64, job string, status string, errorMessage string, createdAt time.Time) error {
	if len(repositoryIDs) == 0 {
		return nil
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
		_, err := exec.ExecContext(
			ctx,
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

	_, err := exec.ExecContext(
		ctx,
		repositoryJobEventInsertQuery(len(repositoryIDs)),
		insertArgs...,
	)
	return err
}

func createRepositoryJobEventsOnce(exec repositoryJobEventExecutor, repositoryIDs []int64, job string, status string, errorMessage string, createdAt time.Time) error {
	if len(repositoryIDs) == 0 {
		return nil
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

	_, err := exec.Exec(repositoryJobEventInsertQuery(len(repositoryIDs)), insertArgs...)
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

	_, err := exec.Exec(repositoryJobEventInsertQuery(1), repositoryID, job, status, trimmedErrorMessage, createdAt)
	return err
}

func repositoryJobEventInsertQuery(rowCount int) string {
	return `WITH event_values(repository_id, job, status, error_message, created_at) AS (
			VALUES ` + rowPlaceholders(rowCount, 5) + `
		)
		INSERT INTO repository_job_events (repository_id, job, status, error_message, created_at)
		SELECT DISTINCT repository_id, job, status, error_message, created_at
		FROM event_values
		WHERE NOT EXISTS (
			SELECT 1
			FROM repository_job_events existing
			WHERE existing.repository_id = event_values.repository_id
			  AND existing.job = event_values.job
			  AND existing.status = event_values.status
			  AND existing.error_message IS event_values.error_message
			  AND existing.created_at = event_values.created_at
		)`
}
