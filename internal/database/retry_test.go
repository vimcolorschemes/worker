package database

import (
	"context"
	"fmt"
	"testing"
)

func TestIsTransientLibSQLError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "invalid token is transient",
			err:  testError("Error executing statement: invalid token"),
			want: true,
		},
		{
			name: "generation mismatch is transient",
			err:  testError("stream not found: generation mismatch: 2 != 1"),
			want: true,
		},
		{
			name: "sqlite busy is transient",
			err:  testError(`Error { message: "SQLite error: database is locked", code: "SQLITE_BUSY" }`),
			want: true,
		},
		{
			name: "operation timeout is transient",
			err:  fmt.Errorf("exec timed out: %w", context.DeadlineExceeded),
			want: true,
		},
		{
			name: "hrana unexpected multiple responses is transient",
			err:  testError("error code = 2: Error executing statement: Hrana: `unexpected response: `Unexpected multiple responses from server"),
			want: true,
		},
		{
			name: "constraint failure is not transient",
			err:  testError("UNIQUE constraint failed"),
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := isTransientLibSQLError(test.err)
			if got != test.want {
				t.Fatalf("isTransientLibSQLError() = %v, want %v", got, test.want)
			}
		})
	}
}

type testError string

func (e testError) Error() string {
	return string(e)
}
