package database

import "github.com/vimcolorschemes/worker/internal/repository"

const (
	repositorySelectColumns = `
		id,
		owner_name,
		owner_avatar_url,
		name,
		github_url,
		stargazers_count,
		stargazers_count_history,
		week_stargazers_count,
		github_created_at,
		pushed_at,
		is_eligible,
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
	`

	queryRepositoriesToGenerate = `
		SELECT ` + repositorySelectColumns + `
		FROM repositories r
		LEFT JOIN (
			SELECT repository_id, MAX(created_at) AS last_generate_event_at
			FROM repository_job_events
			WHERE job = 'generate'
			GROUP BY repository_id
		) ge ON ge.repository_id = r.id
		WHERE r.is_eligible = 1
		  AND (ge.last_generate_event_at IS NULL OR r.pushed_at > ge.last_generate_event_at)
	`
)

// queryRepositories executes a repository query and hydrates color scheme data.
func queryRepositories(query string, args ...any) ([]repository.Repository, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var repositories []repository.Repository
	for rows.Next() {
		repo, err := scanRepositoryWithColorSchemes(rows)
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
