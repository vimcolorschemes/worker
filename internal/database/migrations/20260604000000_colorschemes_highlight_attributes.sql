-- +goose Up
ALTER TABLE colorscheme_groups ADD COLUMN bold BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE colorscheme_groups ADD COLUMN italic BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE colorscheme_groups ADD COLUMN underline BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE colorscheme_groups ADD COLUMN undercurl BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE colorscheme_groups ADD COLUMN underdouble BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE colorscheme_groups ADD COLUMN underdotted BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE colorscheme_groups ADD COLUMN underdashed BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE colorscheme_groups ADD COLUMN strikethrough BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE colorscheme_groups ADD COLUMN reverse BOOLEAN NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE colorscheme_groups DROP COLUMN reverse;
ALTER TABLE colorscheme_groups DROP COLUMN strikethrough;
ALTER TABLE colorscheme_groups DROP COLUMN underdashed;
ALTER TABLE colorscheme_groups DROP COLUMN underdotted;
ALTER TABLE colorscheme_groups DROP COLUMN underdouble;
ALTER TABLE colorscheme_groups DROP COLUMN undercurl;
ALTER TABLE colorscheme_groups DROP COLUMN underline;
ALTER TABLE colorscheme_groups DROP COLUMN italic;
ALTER TABLE colorscheme_groups DROP COLUMN bold;
