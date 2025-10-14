CREATE TABLE colorscheme_variants (
  colorscheme_name TEXT NOT NULL,
  repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  background TEXT NOT NULL CHECK (background IN ('light', 'dark')),
  color_data JSON NOT NULL,

  PRIMARY KEY (colorscheme_name, repository_id, background)
  FOREIGN KEY (colorscheme_name, repository_id) REFERENCES colorschemes(name, repository_id) ON DELETE CASCADE
)
