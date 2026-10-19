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
		activities []model.Activity
		storeErr   error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "returns every activity",
			activities: activities,
			wantStatus: http.StatusOK,
			wantBody:   `[{"id":"01a15480-2665-7ab8-a1a3-7f309df663d4","user_id":"alice","date":"2026-10-19T06:30:00Z","step_count":4200}]`,
		},
		{name: "returns 204 without activities", wantStatus: http.StatusNoContent},
		{name: "returns 500 when the store fails", storeErr: errors.New("boom"), wantStatus: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{activities: tt.activities, err: tt.storeErr}

			rec := httptest.NewRecorder()
			New(store).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/activities", nil))

			assert.Equal(t, tt.wantStatus, rec.Code)
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
