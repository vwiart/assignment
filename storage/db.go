package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Open(ctx context.Context, path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to open database: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("unable to connect: %w", err)
	}
	return db, nil
}

func ApplyMigrations(ctx context.Context, db *sql.DB) error {
	fsys, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("unable to load migrations: %w", err)
	}

	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return fmt.Errorf("unable to list migrations: %w", err)
	}

	versions := make([]int, len(names))
	for i, name := range names {
		version, err := migrationVersion(name)
		if err != nil {
			return err
		}
		if i > 0 && version <= versions[i-1] {
			return fmt.Errorf("migration %s: version %d is not greater than %d", name, version, versions[i-1])
		}
		versions[i] = version
	}

	var current int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		return fmt.Errorf("unable to read database version: %w", err)
	}

	for i, name := range names {
		if versions[i] <= current {
			continue
		}
		if err := applyMigration(ctx, db, fsys, name, versions[i]); err != nil {
			return err
		}
	}
	return nil
}

// migrationVersion returns the version NNNN of a file named NNNN_description.sql.
func migrationVersion(name string) (int, error) {
	prefix, _, found := strings.Cut(name, "_")
	if !found {
		return 0, fmt.Errorf("migration %s: name must be NNNN_description.sql", name)
	}
	version, err := strconv.Atoi(prefix)
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("migration %s: invalid version %q", name, prefix)
	}
	return version, nil
}

// applyMigration runs the file and sets the database version in a single transaction.
func applyMigration(ctx context.Context, db *sql.DB, fsys fs.FS, name string, version int) error {
	content, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("unable to read migration %s: %w", name, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("unable to start migration %s: %w", name, err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("unable to apply migration %s: %w", name, err)
	}
	// PRAGMA does not accept placeholders; version is an int parsed from the file name.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
		return fmt.Errorf("unable to set database version for migration %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("unable to commit migration %s: %w", name, err)
	}
	return nil
}
