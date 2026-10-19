package activity

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vwiart/assignment/model"
)

func TestSubmitHandler(t *testing.T) {
	valid := `{"step_count": 4200, "date": "2026-10-19T08:30:00+02:00"}`

	tests := []struct {
		name       string
		body       string
		storeErr   error
		wantStatus int
		wantStored bool
	}{
		{name: "stores the activity and returns the total", body: valid, wantStatus: http.StatusOK, wantStored: true},
		{name: "rejects a body that is not JSON", body: `4200 steps`, wantStatus: http.StatusBadRequest},
		{name: "rejects a negative step count", body: `{"step_count": -1, "date": "2026-10-19T08:30:00Z"}`, wantStatus: http.StatusBadRequest},
		{name: "rejects zero steps", body: `{"step_count": 0, "date": "2026-10-19T08:30:00Z"}`, wantStatus: http.StatusBadRequest},
		{name: "rejects more than 100000 steps", body: `{"step_count": 100001, "date": "2026-10-19T08:30:00Z"}`, wantStatus: http.StatusBadRequest},
		{name: "rejects a date that is not RFC 3339", body: `{"step_count": 4200, "date": "2026-10-19"}`, wantStatus: http.StatusBadRequest},
		{name: "returns 500 when the store fails", body: valid, storeErr: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantStored: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{aggregate: model.Aggregate{Total: 42}, err: tt.storeErr}

			rec := httptest.NewRecorder()
			New(store).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/users/alice", strings.NewReader(tt.body)))

			assert.Equal(t, tt.wantStatus, rec.Code)
			if !tt.wantStored {
				assert.Nil(t, store.gotActivity, "store must not be called")
				return
			}
			require.NotNil(t, store.gotActivity)
			assert.NotEqual(t, uuid.Nil(), store.gotActivity.ID)
			assert.Equal(t, "alice", store.gotActivity.UserID)
			assert.Equal(t, 4200, store.gotActivity.StepCount)
			assert.True(t, time.Date(2026, 10, 19, 6, 30, 0, 0, time.UTC).Equal(store.gotActivity.Date))
			if tt.wantStatus == http.StatusOK {
				assert.JSONEq(t, `{"total":42}`, rec.Body.String())
			}
		})
	}
}
