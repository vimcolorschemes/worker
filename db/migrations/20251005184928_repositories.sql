CREATE TABLE repositories (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  owner TEXT NOT NULL,
  description TEXT,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  github_url TEXT NOT NULL UNIQUE CHECK (github_url LIKE 'https?://github.com/%/%'),

  UNIQUE(owner, name)
);
