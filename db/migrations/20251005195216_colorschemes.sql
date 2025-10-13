CREATE TABLE colorschemes (
  name TEXT NOT NULL,
  repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,

  PRIMARY KEY (name, repository_id)
);

CREATE TABLE colorscheme_color_data (
  colorscheme_name TEXT NOT NULL,
  repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  background TEXT NOT NULL CHECK (background IN ('light', 'dark')),
  color_data JSON NOT NULL,

  PRIMARY KEY (colorscheme_name, repository_id, background)
)
