CREATE TABLE repository_job_reports (
  job TEXT NOT NULL CHECK (job IN ('import', 'update', 'generate')),
  repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  success BOOLEAN NOT NULL,
  error TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY (job, repository_id, created_at)
);
