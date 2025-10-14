package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
)

type Background string

const (
	BackgroundLight Background = "light"
	BackgroundDark  Background = "dark"
)

type ColorschemeVariant struct {
	Name         string                       `db:"colorscheme_name"`
	RepositoryID int64                        `db:"repository_id"`
	Background   Background                   `db:"background"`
	ColorData    []ColorschemeGroupDefinition `db:"color_data"`
}

type ColorschemeGroupDefinition struct {
	Name    string
	HexCode string
}

type ColorschemeVariantStore struct {
	database *sql.DB
}

func NewColorschemeVariantStore(database *sql.DB) *ColorschemeVariantStore {
	return &ColorschemeVariantStore{database: database}
}

func (store *ColorschemeVariantStore) UpsertColorschemeVariant(ctx context.Context, colorschemeVariant *ColorschemeVariant) error {
	colorData, err := json.Marshal(colorschemeVariant.ColorData)
	if err != nil {
		return err
	}

	log.Printf("Upserting colorscheme variant %s: %v", colorschemeVariant.Name, colorData)

	_, err = store.database.ExecContext(ctx, `
			INSERT INTO colorscheme_variants (colorscheme_name, repository_id, background, color_data)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(colorscheme_name, repository_id, background)
			DO UPDATE SET color_data = excluded.color_data
		`, colorschemeVariant.Name, colorschemeVariant.RepositoryID, colorschemeVariant.Background, string(colorData))

	return err
}
