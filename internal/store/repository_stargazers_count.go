package store

import (
	"context"
	"database/sql"
	"time"
)

type RepositoryStargazersCount struct {
	RepositoryID    int64     `db:"repository_id"`
	SnapshotDate    time.Time `db:"snapshot_date"`
	StargazersCount int       `db:"stargazers_count"`
}

type RepositoryStargarzersCountStore struct {
	database *sql.DB
}

func NewRepositoryStargazersCountStore(database *sql.DB) *RepositoryStargarzersCountStore {
	return &RepositoryStargarzersCountStore{database: database}
}

func (store *RepositoryStargarzersCountStore) Insert(ctx context.Context, c RepositoryStargazersCount) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	_, err := store.database.ExecContext(ctx, `
		INSERT INTO repository_stargazers_count_snapshots (repository_id, snapshot_date, stargazers_count)
		VALUES (?, ?, ?)
		ON CONFLICT(repository_id, snapshot_date) DO UPDATE SET stargazers_count = excluded.stargazers_count
	`,
		c.RepositoryID,
		today,
		c.StargazersCount,
	)
	return err
}
