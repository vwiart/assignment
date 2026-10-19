package activity

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vwiart/assignment/model"
	storage "github.com/vwiart/assignment/storage/activity"
)

func TestAggregateHandlers(t *testing.T) {
	from := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		target     string
		storeErr   error
		wantStatus int
		wantOpts   *storage.AggregateOptions
	}{
		{
			name:       "returns the total without a range",
			target:     "/activities/total",
			wantStatus: http.StatusOK,
			wantOpts:   &storage.AggregateOptions{},
		},
		{
			name:       "passes the user",
			target:     "/users/alice/total",
			wantStatus: http.StatusOK,
			wantOpts:   &storage.AggregateOptions{UserID: "alice"},
		},
		{
			name:       "accepts from alone",
			target:     "/activities/total?from=2026-10-19T00:00:00Z",
			wantStatus: http.StatusOK,
			wantOpts:   &storage.AggregateOptions{From: from},
		},
		{
			name:       "accepts to alone",
			target:     "/users/alice/total?to=2026-10-20T00:00:00Z",
			wantStatus: http.StatusOK,
			wantOpts:   &storage.AggregateOptions{UserID: "alice", To: to},
		},
		{
			name:       "accepts from and to",
			target:     "/activities/total?from=2026-10-19T00:00:00Z&to=2026-10-20T00:00:00Z",
			wantStatus: http.StatusOK,
			wantOpts:   &storage.AggregateOptions{From: from, To: to},
		},
		{
			name:       "accepts from equal to to",
			target:     "/activities/total?from=2026-10-19T00:00:00Z&to=2026-10-19T00:00:00Z",
			wantStatus: http.StatusOK,
			wantOpts:   &storage.AggregateOptions{From: from, To: from},
		},
		{
			name:       "accepts an encoded positive offset",
			target:     "/activities/total?from=2026-10-19T02:00:00%2B02:00&to=2026-10-20T02:00:00%2B02:00",
			wantStatus: http.StatusOK,
			wantOpts:   &storage.AggregateOptions{From: from, To: to},
		},
		{
			name:       "rejects an unencoded positive offset",
			target:     "/activities/total?from=2026-10-19T02:00:00+02:00",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rejects an invalid from",
			target:     "/activities/total?from=yesterday",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rejects an invalid to",
			target:     "/users/alice/total?to=2026-10-20",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rejects to before from",
			target:     "/users/alice/total?from=2026-10-20T00:00:00Z&to=2026-10-19T00:00:00Z",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns 404 for an unknown user",
			target:     "/users/dave/total",
			storeErr:   storage.ErrUserNotFound,
			wantStatus: http.StatusNotFound,
			wantOpts:   &storage.AggregateOptions{UserID: "dave"},
		},
		{
			name:       "returns 500 when the store fails",
			target:     "/activities/total",
			storeErr:   errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantOpts:   &storage.AggregateOptions{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{aggregate: model.Aggregate{Total: 42}, err: tt.storeErr}
			rec := httptest.NewRecorder()
			New(store).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.target, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantOpts == nil {
				assert.Nil(t, store.gotOpts, "store must not be called")
				return
			}
			require.NotNil(t, store.gotOpts)
			assert.Equal(t, tt.wantOpts.UserID, store.gotOpts.UserID)
			assert.True(t, tt.wantOpts.From.Equal(store.gotOpts.From), "from: want %s, got %s", tt.wantOpts.From, store.gotOpts.From)
			assert.True(t, tt.wantOpts.To.Equal(store.gotOpts.To), "to: want %s, got %s", tt.wantOpts.To, store.gotOpts.To)
			if tt.wantStatus == http.StatusOK {
				assert.JSONEq(t, `{"total":42}`, rec.Body.String())
			}
		})
	}
}
