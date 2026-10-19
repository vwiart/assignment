package storage

import (
	"database/sql"
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
		name    string
		opts    AggregateOptions
		want    int
		wantErr error
	}{
		{name: "sums every activity without options", want: 111},
		{name: "sums one user's activities", opts: AggregateOptions{UserID: "alice"}, want: 110},
		{name: "returns zero for a user without activities", opts: AggregateOptions{UserID: "charles"}, want: 0},
		{name: "returns ErrUserNotFound for an unknown user", opts: AggregateOptions{UserID: "dave"}, wantErr: ErrUserNotFound},
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
			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, agg.Total)
		})
	}
}

func TestSubmit(t *testing.T) {
	store := newStore(t)
	date := time.Date(2026, 10, 19, 8, 30, 0, 0, time.UTC)

	tests := []struct {
		name      string
		userID    string
		steps     int
		wantTotal int
		wantErr   bool
		wantErrIs error
	}{
		{name: "returns the total including the new activity", userID: "alice", steps: 100, wantTotal: 100},
		{name: "returns the total across all users", userID: "bob", steps: 7, wantTotal: 107},
		{name: "rejects an unknown user", userID: "dave", steps: 1, wantErr: true, wantErrIs: ErrUserNotFound},
		{name: "rejects zero steps", userID: "alice", steps: 0, wantErr: true},
		{name: "rejects more than 100000 steps", userID: "alice", steps: 100_001, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg, err := store.Submit(t.Context(), model.Activity{ID: uuid.NewV7(), UserID: tt.userID, Date: date, StepCount: tt.steps})
			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrIs != nil {
					assert.ErrorIs(t, err, tt.wantErrIs)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, agg.Total)
		})
	}

	t.Run("does not store rejected activities", func(t *testing.T) {
		agg, err := store.Aggregate(t.Context(), AggregateOptions{})
		require.NoError(t, err)
		assert.Equal(t, 107, agg.Total)
	})
}

func TestList(t *testing.T) {
	alice := model.Activity{
		ID:        uuid.NewV7(),
		UserID:    "alice",
		Date:      time.Date(2026, 10, 19, 8, 30, 0, 0, time.FixedZone("CEST", 2*60*60)),
		StepCount: 100,
	}
	bob := model.Activity{ID: uuid.NewV7(), UserID: "bob", Date: time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC), StepCount: 7}
	aliceUTC := alice
	aliceUTC.Date = alice.Date.UTC()

	tests := []struct {
		name string
		seed []model.Activity
		want []model.Activity
	}{
		{name: "returns nothing without activities"},
		{name: "returns every activity with its date in UTC", seed: []model.Activity{alice, bob}, want: []model.Activity{aliceUTC, bob}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newStore(t)
			for _, a := range tt.seed {
				_, err := store.Submit(t.Context(), a)
				require.NoError(t, err)
			}

			got, err := store.List(t.Context())
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDelete(t *testing.T) {
	store := newStore(t)
	date := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	kept := model.Activity{ID: uuid.NewV7(), UserID: "alice", Date: date, StepCount: 100}
	deleted := model.Activity{ID: uuid.NewV7(), UserID: "bob", Date: date, StepCount: 7}
	for _, a := range []model.Activity{kept, deleted} {
		_, err := store.Submit(t.Context(), a)
		require.NoError(t, err)
	}

	tests := []struct {
		name      string
		id        uuid.UUID
		wantErr   error
		wantTotal int
	}{
		{name: "deletes one activity", id: deleted.ID, wantTotal: 100},
		{name: "returns ErrNoRows for an activity already deleted", id: deleted.ID, wantErr: sql.ErrNoRows, wantTotal: 100},
		{name: "returns ErrNoRows for an unknown activity", id: uuid.NewV7(), wantErr: sql.ErrNoRows, wantTotal: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, store.Delete(t.Context(), tt.id), tt.wantErr)

			agg, err := store.Aggregate(t.Context(), AggregateOptions{})
			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, agg.Total)
		})
	}
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
