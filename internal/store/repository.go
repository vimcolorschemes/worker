package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Repository struct {
	ID          int64     `db:"id"`
	Name        string    `db:"name"`
	Owner       string    `db:"owner"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type RepositoryStore struct {
	database *sql.DB
}

func NewRepositoryStore(database *sql.DB) *RepositoryStore {
	return &RepositoryStore{database: database}
}

func (store *RepositoryStore) GetByKey(ctx context.Context, owner, name string) (*Repository, error) {
	var r Repository
	err := store.database.QueryRowContext(ctx, `
		SELECT id, owner, name, description, created_at, updated_at
		FROM repositories
		WHERE owner = ? AND name = ?
	`, owner, name).Scan(
		&r.ID,
		&r.Owner,
		&r.Name,
		&r.Description,
		&r.CreatedAt,
		&r.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &r, nil
}

func (store *RepositoryStore) Upsert(ctx context.Context, r Repository) error {
	_, err := store.database.ExecContext(ctx, `
		INSERT INTO repositories (id, owner, name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			owner = excluded.owner,
			name = excluded.name,
			description = excluded.description,
			updated_at = excluded.updated_at;
	`,
		r.ID,
		r.Owner,
		r.Name,
		r.Description,
		r.CreatedAt,
		r.UpdatedAt,
	)
	return err
}
