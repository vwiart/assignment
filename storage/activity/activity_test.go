package storage

import (
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vwiart/assignment/model"
	db "github.com/vwiart/assignment/storage"
)

func TestAggregate(t *testing.T) {
	store := newStore(t)
	submit(t, store, "alice", "2026-10-19T00:00:00Z", 100)
	submit(t, store, "alice", "2026-10-19T12:00:00Z", 10)
	submit(t, store, "bob", "2026-10-20T00:00:00Z", 1)

	tests := []struct {
		name string
		opts AggregateOptions
		want int
	}{
		{name: "sums every activity without options", want: 111},
		{name: "sums one user's activities", opts: AggregateOptions{UserID: "alice"}, want: 110},
		{name: "returns zero for a user without activities", opts: AggregateOptions{UserID: "charles"}, want: 0},
		{name: "includes an activity at from", opts: AggregateOptions{From: parse(t, "2026-10-19T12:00:00Z")}, want: 11},
		{name: "excludes an activity at to", opts: AggregateOptions{To: parse(t, "2026-10-20T00:00:00Z")}, want: 110},
		{
			name: "sums activities in [from, to)",
			opts: AggregateOptions{From: parse(t, "2026-10-19T00:00:00Z"), To: parse(t, "2026-10-20T00:00:00Z")},
			want: 110,
		},
		{
			name: "returns zero for an empty range",
			opts: AggregateOptions{From: parse(t, "2026-10-19T12:00:00Z"), To: parse(t, "2026-10-19T12:00:00Z")},
			want: 0,
		},
		{
			name: "returns zero for a range without activities",
			opts: AggregateOptions{From: parse(t, "2026-10-19T01:00:00Z"), To: parse(t, "2026-10-19T11:00:00Z")},
			want: 0,
		},
		{
			name: "converts bounds with an offset to UTC",
			opts: AggregateOptions{From: parse(t, "2026-10-19T02:00:00+02:00"), To: parse(t, "2026-10-20T02:00:00+02:00")},
			want: 110,
		},
		{
			name: "combines user and range",
			opts: AggregateOptions{UserID: "bob", From: parse(t, "2026-10-19T00:00:00Z"), To: parse(t, "2026-10-20T00:00:00Z")},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg, err := store.Aggregate(t.Context(), tt.opts)
			require.NoError(t, err)
			assert.Equal(t, tt.want, agg.Total)
		})
	}
}

func TestSubmit(t *testing.T) {
	t.Run("returns the total across all users", func(t *testing.T) {
		store := newStore(t)

		assert.Equal(t, 100, submit(t, store, "alice", "2026-10-19T00:00:00Z", 100).Total)
		assert.Equal(t, 107, submit(t, store, "bob", "2026-10-20T00:00:00Z", 7).Total)
	})
}

func newStore(t *testing.T) activity {
	t.Helper()
	conn, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "db"))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, db.ApplyMigrations(t.Context(), conn))
	return NewActivity(conn)
}

func submit(t *testing.T, store activity, userID, date string, steps int) model.Aggregate {
	t.Helper()
	agg, err := store.Submit(t.Context(), model.Activity{
		ID:        uuid.NewV7(),
		UserID:    userID,
		Date:      parse(t, date),
		StepCount: steps,
	})
	require.NoError(t, err)
	return agg
}

func parse(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	return d
}
