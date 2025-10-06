ALTER TABLE repositories ADD COLUMN cached_stargazers_count INTEGER DEFAULT 0;
CREATE INDEX idx_repositories_cached_stargazers_count ON repositories(cached_stargazers_count);

ALTER TABLE repositories ADD COLUMN cached_weekly_stargazers_count INTEGER DEFAULT 0;
CREATE INDEX idx_repositories_cached_weekly_stargazers_count ON repositories(cached_weekly_stargazers_count);

CREATE TABLE repository_stargazers_count_snapshots (
  repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  snapshot_date TEXT NOT NULL,
  stargazers_count INTEGER NOT NULL,

  PRIMARY KEY (repository_id, snapshot_date)
);

CREATE TRIGGER update_cached_stargazers_count
AFTER INSERT ON repository_stargazers_count_snapshots
FOR EACH ROW
BEGIN
  UPDATE repositories
  SET cached_stargazers_count = (
    SELECT stargazers_count
    FROM repository_stargazers_count_snapshots
    WHERE repository_id = NEW.repository_id
    ORDER BY snapshot_date DESC, stargazers_count DESC
    LIMIT 1
  )
  WHERE id = NEW.repository_id;
END;

CREATE TRIGGER update_cached_weekly_stargazers_count
AFTER INSERT ON repository_stargazers_count_snapshots
FOR EACH ROW
BEGIN
  UPDATE repositories
  SET cached_weekly_stargazers_count = 
    NEW.stargazers_count -
      (
        SELECT COALESCE(
          (
            SELECT stargazers_count
            FROM repository_stargazers_count_snapshots
            WHERE repository_id = NEW.repository_id
              AND snapshot_date >= date(NEW.snapshot_date, '-7 days')
            ORDER BY snapshot_date ASC, stargazers_count DESC
            LIMIT 1
          ),
          0
        )
      )
  WHERE id = NEW.repository_id;
END;

