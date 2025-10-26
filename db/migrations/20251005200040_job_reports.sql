CREATE TABLE job_reports (
    job TEXT NOT NULL CHECK (job IN ('import', 'update', 'generate')),
    report_data JSONB NOT NULL,
    elapsed_time_in_seconds INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (job, created_at)
);
