package storage

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrate(t *testing.T) {
	t.Run("applies the embedded migrations", func(t *testing.T) {
		db, err := Open(t.Context(), filepath.Join(t.TempDir(), "db"))
		require.NoError(t, err)
		t.Cleanup(func() { db.Close() })

		require.NoError(t, ApplyMigrations(t.Context(), db))

		assert.Equal(t, 1, userVersion(t, db))
		var users []string
		rows, err := db.QueryContext(t.Context(), "SELECT id FROM user ORDER BY id")
		require.NoError(t, err)
		defer rows.Close()
		for rows.Next() {
			var id string
			require.NoError(t, rows.Scan(&id))
			users = append(users, id)
		}
		require.NoError(t, rows.Err())
		assert.Equal(t, []string{"alice", "bob", "charles"}, users)
	})

	t.Run("does nothing when the database is up to date", func(t *testing.T) {
		db, err := Open(t.Context(), filepath.Join(t.TempDir(), "db"))
		require.NoError(t, err)
		t.Cleanup(func() { db.Close() })

		require.NoError(t, ApplyMigrations(t.Context(), db))

		// Re-running 0001 would fail on the seeded users' primary key.
		require.NoError(t, ApplyMigrations(t.Context(), db))
		assert.Equal(t, 1, userVersion(t, db))
	})
}

func TestMigrationVersion(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		want    int
		wantErr bool
	}{
		{name: "parses the version prefix", file: "0001_init.sql", want: 1},
		{name: "parses a multi-digit version", file: "0042_add_index.sql", want: 42},
		{name: "rejects a name without a description", file: "0001.sql", wantErr: true},
		{name: "rejects a non-numeric prefix", file: "init_users.sql", wantErr: true},
		{name: "rejects version zero", file: "0000_init.sql", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := migrationVersion(tt.file)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func userVersion(t *testing.T, conn *sql.DB) int {
	t.Helper()
	var version int
	require.NoError(t, conn.QueryRowContext(t.Context(), "PRAGMA user_version").Scan(&version))
	return version
}
