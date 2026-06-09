package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

const transientLibSQLRetryAttempts = 8
const dbOperationTimeout = 30 * time.Second

func isTransientLibSQLError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	errMsg := strings.ToLower(err.Error())

	for _, marker := range []string{
		"invalid token",
		"unexpected multiple responses",
		"unexpected response",
		"stream not found: generation mismatch",
		"generation mismatch",
		"stream not found",
		"sqlite_busy",
		"database is locked",
		"context deadline exceeded",
	} {
		if strings.Contains(errMsg, marker) {
			return true
		}
	}

	return false
}

func execWithTransientRetry(query string, args ...any) (sql.Result, error) {
	var lastErr error
	for attempt := 0; attempt <= transientLibSQLRetryAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
		result, err := db.ExecContext(ctx, query, args...)
		cancel()
		if err == nil {
			return result, nil
		}

		if !isTransientLibSQLError(err) {
			return nil, err
		}

		lastErr = err
		if attempt == transientLibSQLRetryAttempts {
			break
		}

		log.Printf("DB exec failed with transient libsql error, retrying: %s", err)
		if pingErr := pingWithTimeout(); pingErr != nil {
			return nil, fmt.Errorf("ping before exec retry: %w", pingErr)
		}
		time.Sleep(transientLibSQLRetryBackoff(attempt))
	}

	return nil, lastErr
}

func runWithTransientRetry(operation string, run func() error) error {
	var lastErr error
	for attempt := 0; attempt <= transientLibSQLRetryAttempts; attempt++ {
		err := run()
		if err == nil {
			return nil
		}

		if !isTransientLibSQLError(err) {
			return err
		}

		lastErr = err
		if attempt == transientLibSQLRetryAttempts {
			break
		}

		log.Printf("DB %s failed with transient libsql error, retrying: %s", operation, err)
		if pingErr := pingWithTimeout(); pingErr != nil {
			return fmt.Errorf("ping before %s retry: %w", operation, pingErr)
		}
		time.Sleep(transientLibSQLRetryBackoff(attempt))
	}

	return lastErr
}

func queryWithTransientRetry(query string, args ...any) (*sql.Rows, error) {
	var lastErr error
	for attempt := 0; attempt <= transientLibSQLRetryAttempts; attempt++ {
		rows, err := db.Query(query, args...)
		if err == nil {
			return rows, nil
		}

		if !isTransientLibSQLError(err) {
			return nil, err
		}

		lastErr = err
		if attempt == transientLibSQLRetryAttempts {
			break
		}

		log.Printf("DB query failed with transient libsql error, retrying: %s", err)
		if pingErr := pingWithTimeout(); pingErr != nil {
			return nil, fmt.Errorf("ping before query retry: %w", pingErr)
		}
		time.Sleep(transientLibSQLRetryBackoff(attempt))
	}

	return nil, lastErr
}

func beginWithTransientRetry() (*sql.Tx, error) {
	var lastErr error
	for attempt := 0; attempt <= transientLibSQLRetryAttempts; attempt++ {
		tx, err := db.Begin()
		if err == nil {
			return tx, nil
		}

		if !isTransientLibSQLError(err) {
			return nil, err
		}

		lastErr = err
		if attempt == transientLibSQLRetryAttempts {
			break
		}

		log.Printf("DB begin failed with transient libsql error, retrying: %s", err)
		if pingErr := pingWithTimeout(); pingErr != nil {
			return nil, fmt.Errorf("ping before begin retry: %w", pingErr)
		}
		time.Sleep(transientLibSQLRetryBackoff(attempt))
	}

	return nil, lastErr
}

func pingWithTimeout() error {
	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()
	return db.PingContext(ctx)
}

func transientLibSQLRetryBackoff(attempt int) time.Duration {
	backoff := time.Duration(attempt+1) * 2 * time.Second
	if backoff > 15*time.Second {
		return 15 * time.Second
	}
	return backoff
}
