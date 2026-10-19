# Step count service

A small HTTP service that collects users' step counts and returns totals, built for the
Xensam senior backend technical assignment (time box: about 2 hours).

Users submit a number of steps for a date. Every submission is kept, and the service returns:

- the total across all users, in the response to every submission;
- the total across all users;
- the total for one user;
- either total limited to a date-time range.

## Prerequisites

Install these before running the service or the tests:

- Go **1.27** or newer, from <https://go.dev/dl/>. Check it with `go version`. The code uses the
  standard library `uuid` package, which is not in older Go versions.

Nothing else is needed. There is no C toolchain to install: SQLite runs through
[`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite), a pure-Go driver, and Go downloads
the dependencies on the first build.

## Running

```bash
go run cmd/main.go
```

The server listens on port `8080`. Open <http://localhost:8080/> for a small page that submits
step counts and shows the total across all users and the total for one user, optionally limited
to a date range. It also lists submissions in a table, 20 per page and ordered by ID, with a
button to fetch the next page and a button on each row to delete it after a confirmation. The
table reloads its current page after a submission or a deletion, so it stays full. Set
`SERVER_PORT` to use another port:

```bash
SERVER_PORT=9000 go run cmd/main.go
```

Data is stored in a SQLite database at `data/db`, relative to the working directory, so it
survives restarts. Run the server from the repository root, or set `DATABASE_PATH` to use another
file. The directory must exist. Migrations run on startup. To start from an empty database, stop
the server and delete `data/db*`.

```bash
DATABASE_PATH=/tmp/steps.db go run cmd/main.go
```

On `SIGINT` (Ctrl+C) or `SIGTERM`, the server stops accepting connections, gives in-flight requests
up to 10 seconds to finish, then closes the database.

## Tests

```bash
go test ./...
```

- [storage/db_test.go](storage/db_test.go) checks migration file names and that migrations are
  applied once.
- [storage/activity/activity_test.go](storage/activity/activity_test.go) runs submit, list,
  delete, totals and range bounds against a temporary SQLite database.
- [pkg/activity](pkg/activity) has one test file per endpoint. Each one sends requests through
  the real router to the handlers, backed by a fake store.
- [pkg/ui/ui_test.go](pkg/ui/ui_test.go) checks that the page is served as HTML. The page itself
  was checked by hand in a browser.

## API

All bodies are JSON and sent with `Content-Type: application/json`. Dates use RFC 3339. Every
error has the same shape:

```json
{"error": "step count must be between 1 and 100000"}
```

| Method   | Route                       | Purpose                                                  |
| -------- | --------------------------- | -------------------------------------------------------- |
| `POST`   | `/users/{user_id}`          | Submit a step count. Returns its ID and the total across all users. |
| `GET`    | `/activities/total`         | Total steps across all users.                            |
| `GET`    | `/users/{user_id}/total`    | Total steps for one user.                                |
| `GET`    | `/activities`               | List submissions, one page at a time.                    |
| `DELETE` | `/activities/{activity_id}` | Delete one submission.                                   |

Both `total` routes accept optional `from` and `to` query parameters to limit the range.

### Submit a step count

```bash
curl -X POST localhost:8080/users/alice \
  -d '{"step_count": 4200, "date": "2026-10-19T08:30:00+02:00"}'
```

```json
{"id": "01a15480-2665-7ab8-a1a3-7f309df663d4", "total": 4200}
```

| Status | When                                              |
| ------ | ------------------------------------------------- |
| `200`  | Stored. The body holds the new submission's `id`, to delete it later, and the total across all users. |
| `400`  | Body is not JSON, `step_count` is not between 1 and 100,000, or `date` is not RFC 3339. |
| `404`  | The user does not exist. |
| `413`  | Body is over 1 KiB. |
| `500`  | Storage error. |
| `503`  | The request took more than 100 ms (see below). |

### Get totals

```bash
curl localhost:8080/activities/total
curl localhost:8080/users/alice/total
curl 'localhost:8080/activities/total?from=2026-10-19T00:00:00Z&to=2026-10-20T00:00:00Z'
```

```json
{"total": 4200}
```

`from` and `to` are RFC 3339 timestamps and are both optional. The range includes `from` and
excludes `to` (`from <= date < to`), so adjacent ranges never count a submission twice. An invalid
timestamp, or a `to` before `from`, returns `400`. A total for an unknown user returns `404`.

In a query string, `+` means a space. A timestamp with a positive offset such as `+02:00` must
encode the `+` as `%2B`, or it fails to parse and returns `400`:

```bash
curl 'localhost:8080/activities/total?from=2026-10-19T00:00:00%2B02:00&to=2026-10-20T00:00:00%2B02:00'
```

`curl -G --data-urlencode` does the encoding for you:

```bash
curl -G localhost:8080/activities/total \
  --data-urlencode 'from=2026-10-19T00:00:00+02:00' \
  --data-urlencode 'to=2026-10-20T00:00:00+02:00'
```

Offsets of `Z` or `-hh:mm` need no encoding.

### List and delete submissions

```bash
curl localhost:8080/activities
curl 'localhost:8080/activities?limit=50&after=01a15480-2665-7ab8-a1a3-7f309df663d4'
curl -X DELETE localhost:8080/activities/01a15480-2665-7ab8-a1a3-7f309df663d4
```

`GET /activities` returns submissions ordered by `id`. IDs are UUIDv7, so this is also
submission order. `limit` sets the page size (default 100, at most 1,000). To get the next page,
pass the last `id` of the current page as `after`. An invalid `limit` or `after` returns `400`, and
a page with no submissions returns `204 No Content`. `DELETE` returns
`204 No Content` on success, `400` when the ID is not a UUID and `404` when no submission has
that ID.

### Timeouts

Every request has 100 ms: to be read, and to be handled. A request that takes longer gets
`503` with `{"error":"request timed out"}`. That body is sent as `text/plain`, because Go's
`http.TimeoutHandler` writes it. The request's context is cancelled, so a query still waiting for
the database stops. There is no retry, so the client decides whether to send the request again.
Connections are kept open for 60 seconds between requests.

## Data model

Migrations live in [storage/migrations](storage/migrations).

- `user (id TEXT PRIMARY KEY)`, seeded with `alice`, `bob` and `charles`. The service does not create users.
- `activity (id, user_id, date, step_count)`. One row per submission.
  - `id` is a UUIDv7, generated by the server.
  - `user_id` is a foreign key to `user`.
  - `date` is stored as UTC text in RFC 3339.
  - `step_count` must be between 1 and 100,000 (a database `CHECK`).

Indexes on `(user_id, date)` and `(date)` support the range totals.

## Assumptions

**Several submissions on the same date.** Each submission is a separate row. Nothing is merged
or overwritten, and every row counts towards the totals. The idea is to give the
user an option to manage their activity by allowing them to delete one if they 
made a mistake. The intent is to avoid complex merging implementation that could
bring more harm than good.

The brief's "each submission should be retained" is read together with "a user may submit
multiple step counts on the same day": two submissions on the same day are both kept and both
counted. Deleting a submission is a separate, explicit correction made by the user, so it does not
go against that rule.

**Invalid or negative step counts.** A count outside 1 to 100,000 is rejected with `400`, on the
assumption that one submission covers at most one day of walking. A database `CHECK` enforces the
same range.

**Other input checks are loose on purpose.** A submission is accepted when:

- the body has data after the JSON object: `{...}{garbage}` returns `200`, and only the first
  object is read;
- the body has unknown fields, such as `"extra": 1`, which are ignored;
- the date is far in the future or the past. Any RFC 3339 timestamp is stored, from year `0001`
  to `9999`.

The brief does not say how strict the service should be, so only what the totals depend on is
checked: a step count in range, a valid timestamp and a known user.

**Date ranges with no data.** Return `{"total": 0}`, as do users with no submissions.

**Unknown users.** Users are a fixed, seeded list. A submission or a total for any other user
returns `404`. The foreign key on `activity.user_id` is a second guard.

**Time zones and date boundaries.** Clients send a full timestamp with an offset. The server
converts it to UTC before storing it. A "day" is not defined on the server; the client chooses
the boundaries by passing `from` and `to`.

Dates are compared to the second. Submission dates and the `from`/`to` bounds are written with
`time.RFC3339`, which drops fractions of a second. A bound such as `10:00:00.500Z` therefore acts
as `10:00:00Z`, and a submission at `10:00:00.200Z` counts as being at `10:00:00Z`. This case is
not handled. (The comment in `0001_init.sql` gives the format as `.000Z`; the code does not write
milliseconds.)

## Design decisions and trade-offs

- **SQLite** keeps the service to one binary and one file, and meets the persistence requirement.
  SQLite allows one writer at a time, so the pool holds a single connection: database/sql queues
  concurrent requests instead of SQLite rejecting them with `SQLITE_BUSY`. Reads queue behind
  writes too, which is fine at this scale. A shared database server would be needed to run more
  than one instance.
- **Migrations** are SQL files embedded in the binary. The schema version is kept in
  `PRAGMA user_version`, and each file runs in its own transaction. This avoids a migration library.
- **Totals are computed at read time** with `SUM`, backed by indexes. This is simple and always
  correct. With much more data, a per-user, per-day rollup table would be cheaper to read.
- **Submit inserts the row and computes the total in one transaction**, so the returned total
  includes the new submission.
- **Layers**: HTTP handlers in [pkg/activity](pkg/activity) depend on a `Storage` interface;
  the SQLite implementation is in [storage/activity](storage/activity). The interface lets handlers
  be tested with a fake store.
- **Store errors** are sentinel values (`ErrUserNotFound`, `ErrActivityNotFound`) that the
  handlers map to `404`. An unknown user is found with an existence query in the same transaction,
  not by parsing SQLite's foreign key error. It costs one primary key lookup and keeps the
  handlers independent of driver error codes.
- **Pagination** uses the `id` as a cursor (keyset) instead of an offset. Pages stay stable while
  submissions are added and each page is one index range scan. There is no total count or jump to
  page N.
- **Routing** uses [chi](https://github.com/go-chi/chi) for path parameters.
- **Timeouts** are short (100 ms) and fail hard with `503`, without retries. A submission whose
  transaction commits just as the timeout fires is stored even though the client sees `503`. A
  client that sends it again counts the steps twice. Returning the new `id` lets the client check
  the list before it sends a submission again.
- **UI** is one HTML file with plain JavaScript, embedded in the binary and calling the same JSON
  API. There is no build step.

## Known issues

None known. Accepted limits are listed under "Assumptions" and "Design decisions and trade-offs".

## Possible improvements

Not done, because they depend on information this assignment does not give:

- Let users be created, or accept any user ID, depending on how users are managed upstream.
- Add a per-day rollup table if totals become slow. At this scale, indexed `SUM` queries are fast
  enough and always consistent.
