# AGENTS.md

Guidance for coding agents working in this repository.

## Context

This is a solution to a timeboxed (about 2 hours) senior backend assignment: a step count
service. The brief is `xensam-technical-assignment-senior-backend.pdf` at the repository root.
It is gitignored. Read it before changing behaviour.

The reviewers care about a clear working solution, documented assumptions, readable code and
the trade-offs behind it. Production-grade completeness is not expected. Prefer small, readable
changes over new abstractions or dependencies.

[README.md](README.md) is part of the submission. When you change behaviour, update the
matching sections: API, Assumptions, Known issues and Possible improvements. Remove an entry
from Known issues once it is fixed and tested.

## Commands

```bash
go build ./...
go vet ./...
go test ./...
go run ./cmd/main.go
```

`go run ./cmd` listens on `SERVER_PORT` (default `8080`) and writes to `data/db` relative to the
working directory. To try the server without touching the developer's database, run the built
binary from a scratch directory that contains an empty `data/` folder.

## Layout

| Path | Role |
| ---- | ---- |
| `cmd/main.go` | Wiring: opens the database, runs migrations, registers chi routes. |
| `model/` | Plain structs shared across layers, with JSON tags. |
| `pkg/activity/` | HTTP handlers, one file per endpoint. Each method returns an `http.HandlerFunc`. Handlers depend on the `Storage` interface in `activity.go`. |
| `storage/db.go` | `Open` (SQLite with WAL, foreign keys and busy timeout) and `ApplyMigrations`. |
| `storage/migrations/` | Embedded SQL migrations. |
| `storage/activity/` | SQLite implementation of `Storage`. |
| `data/` | Runtime database files. Gitignored except `.gitkeep`. |

## Conventions

- **Go 1.27.** Use the standard library `uuid` package (`uuid.NewV7`, `uuid.Parse`). Do not add
  `github.com/google/uuid`; it is only an indirect dependency of the SQLite driver.
- **SQLite driver** is `modernc.org/sqlite` (pure Go, driver name `"sqlite"`). Do not switch to a
  cgo driver.
- **SQL** uses `?` placeholders. Never build queries with values formatted into the string.
  Optional filters are appended to a `strings.Builder` after `WHERE 1=1`.
- **Dates** are stored as UTC RFC 3339 text and compared as strings. Convert to UTC before
  writing or comparing.
- **Logging** uses `log/slog` through `slog.Default()`. Log the underlying error; send the client
  a short message without internal details.
- **Errors** are wrapped with `fmt.Errorf("unable to …: %w", err)` in the storage setup code.
  Map storage errors to HTTP status codes in the handlers, not in the store.
- **Doc comments** start with the identifier name.

## Migrations

- Files are named `NNNN_description.sql`, with strictly increasing versions.
- The applied version is kept in `PRAGMA user_version`. Each file runs in one transaction
  together with the version bump.
- Never edit a migration that has been applied. Add a new file instead. `0001_init.sql` already
  exists in developers' local databases.
- When you add a migration, update the expected version in `storage/db_test.go`.

## Tests

- Use `testify`: `require` for preconditions, `assert` for checks.
- Group cases under `t.Run` with sentence-style names, as in `storage/db_test.go`.
- Use `t.Context()` and a database in `t.TempDir()`. Never use `data/db` in tests.
- Test handlers with `httptest` and a fake `Storage`. Test the store against a real temporary
  SQLite database with migrations applied.
- Every bug fix comes with a test that fails before the fix.

## Do not commit

- `data/db*` files.
- The assignment PDF.
