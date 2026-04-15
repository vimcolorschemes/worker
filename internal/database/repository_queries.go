package database

import "github.com/vimcolorschemes/worker/internal/repository"

const (
	repositorySelectColumns = `
		id,
		owner_name,
		owner_avatar_url,
		name,
		description,
		github_url,
		stargazers_count,
		stargazers_count_history,
		week_stargazers_count,
		github_created_at,
		pushed_at,
		is_eligible,
		is_disabled,
		updated_at
	`

	queryRepositoryByOwnerAndName = `
		SELECT ` + repositorySelectColumns + `
		FROM repositories
		WHERE owner_name = ? COLLATE NOCASE
		  AND name = ? COLLATE NOCASE
	`

	queryAllRepositories = `
		SELECT ` + repositorySelectColumns + `
		FROM repositories
		WHERE is_disabled = 0
	`

	queryRepositoriesToGenerate = `
		SELECT ` + repositorySelectColumns + `
		FROM repositories
		WHERE is_disabled = 0
		  AND is_eligible = 1
		  AND (last_generate_event_at IS NULL OR pushed_at > last_generate_event_at)
	`
)

// queryRepositoriesBasic executes a repository query without hydrating colorscheme data.
func queryRepositoriesBasic(query string, args ...any) ([]repository.Repository, error) {
	rows, err := queryWithTransientRetry(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var repositories []repository.Repository
	for rows.Next() {
		repo, err := scanRepository(rows)
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, repo)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return repositories, nil
}
