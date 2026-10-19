package activity

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vwiart/assignment/model"
	storage "github.com/vwiart/assignment/storage/activity"
)

func TestListHandler(t *testing.T) {
	activities := []model.Activity{{
		ID:        uuid.MustParse("01a15480-2665-7ab8-a1a3-7f309df663d4"),
		UserID:    "alice",
		Date:      time.Date(2026, 10, 19, 6, 30, 0, 0, time.UTC),
		StepCount: 4200,
	}}

	tests := []struct {
		name       string
		target     string
		activities []model.Activity
		storeErr   error
		wantStatus int
		wantBody   string
		wantOpts   *storage.ListOptions
	}{
		{
			name:       "returns the first page by default",
			target:     "/activities",
			activities: activities,
			wantStatus: http.StatusOK,
			wantBody:   `[{"id":"01a15480-2665-7ab8-a1a3-7f309df663d4","user_id":"alice","date":"2026-10-19T06:30:00Z","step_count":4200}]`,
			wantOpts:   &storage.ListOptions{Limit: 100},
		},
		{
			name:       "passes limit and after",
			target:     "/activities?limit=1000&after=01a15480-2665-7ab8-a1a3-7f309df663d4",
			activities: activities,
			wantStatus: http.StatusOK,
			wantBody:   `[{"id":"01a15480-2665-7ab8-a1a3-7f309df663d4","user_id":"alice","date":"2026-10-19T06:30:00Z","step_count":4200}]`,
			wantOpts:   &storage.ListOptions{Limit: 1000, After: activities[0].ID},
		},
		{name: "rejects a limit that is not a number", target: "/activities?limit=all", wantStatus: http.StatusBadRequest},
		{name: "rejects a zero limit", target: "/activities?limit=0", wantStatus: http.StatusBadRequest},
		{name: "rejects a limit above 1000", target: "/activities?limit=1001", wantStatus: http.StatusBadRequest},
		{name: "rejects an after that is not a UUID", target: "/activities?after=42", wantStatus: http.StatusBadRequest},
		{name: "returns 204 without activities", target: "/activities", wantStatus: http.StatusNoContent, wantOpts: &storage.ListOptions{Limit: 100}},
		{name: "returns 500 when the store fails", target: "/activities", storeErr: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantOpts: &storage.ListOptions{Limit: 100}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{activities: tt.activities, err: tt.storeErr}

			rec := httptest.NewRecorder()
			New(store).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.target, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantOpts, store.gotListOpts)
			if rec.Code != http.StatusNoContent {
				assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			}
			if rec.Code >= http.StatusBadRequest {
				var body struct{ Error string }
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.NotEmpty(t, body.Error)
			}
			switch tt.wantStatus {
			case http.StatusOK:
				assert.JSONEq(t, tt.wantBody, rec.Body.String())
			case http.StatusNoContent:
				assert.Empty(t, rec.Body.String())
			}
		})
	}
}
