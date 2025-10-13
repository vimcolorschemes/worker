package store

import (
	"context"
	"database/sql"
)

type RepositoryStargazersCountSnapshot struct {
	RepositoryID    int64 `db:"repository_id"`
	StargazersCount int   `db:"stargazers_count"`
}

type RepositoryStargarzersCountSnapshotStore struct {
	database *sql.DB
}

func NewRepositoryStargazersCountSnapshotStore(database *sql.DB) *RepositoryStargarzersCountSnapshotStore {
	return &RepositoryStargarzersCountSnapshotStore{database: database}
}

func (store *RepositoryStargarzersCountSnapshotStore) Insert(ctx context.Context, c RepositoryStargazersCountSnapshot) error {
	_, err := store.database.ExecContext(ctx, `
		INSERT INTO repository_stargazers_count_snapshots (repository_id, snapshot_date, stargazers_count)
		VALUES (?, date('now'), ?)
		ON CONFLICT(repository_id, snapshot_date) DO UPDATE SET stargazers_count = excluded.stargazers_count
	`,
		c.RepositoryID,
		c.StargazersCount,
	)
	return err
}
