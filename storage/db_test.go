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

func userVersion(t *testing.T, conn *sql.DB) int {
	t.Helper()
	var version int
	require.NoError(t, conn.QueryRowContext(t.Context(), "PRAGMA user_version").Scan(&version))
	return version
}

func tableExists(t *testing.T, conn *sql.DB, name string) bool {
	t.Helper()
	var count int
	err := conn.QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", name).Scan(&count)
	require.NoError(t, err)
	return count == 1
}
