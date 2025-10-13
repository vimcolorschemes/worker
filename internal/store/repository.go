package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
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

func (store *RepositoryStore) GetByKey(ctx context.Context, key string) (*Repository, error) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid repository key")
	}
	owner := parts[0]
	name := parts[1]

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

func (store *RepositoryStore) GetAll() []Repository {
	rows, err := store.database.Query(`
		SELECT id, owner, name, description, created_at, updated_at
		FROM repositories
	`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var repositories []Repository
	for rows.Next() {
		var r Repository
		if err := rows.Scan(
			&r.ID,
			&r.Owner,
			&r.Name,
			&r.Description,
			&r.CreatedAt,
			&r.UpdatedAt,
		); err != nil {
			panic(err)
		}
		repositories = append(repositories, r)
	}

	if err := rows.Err(); err != nil {
		panic(err)
	}

	return repositories
}
