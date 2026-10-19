package activity

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	storage "github.com/vwiart/assignment/storage/activity"
)

func TestDeleteHandler(t *testing.T) {
	id := uuid.MustParse("01a15480-2665-7ab8-a1a3-7f309df663d4")

	tests := []struct {
		name       string
		target     string
		storeErr   error
		wantStatus int
		wantID     *uuid.UUID
	}{
		{name: "deletes the activity", target: "/activities/" + id.String(), wantStatus: http.StatusNoContent, wantID: &id},
		{name: "rejects an id that is not a UUID", target: "/activities/42", wantStatus: http.StatusBadRequest},
		{name: "returns 404 for an unknown activity", target: "/activities/" + id.String(), storeErr: storage.ErrActivityNotFound, wantStatus: http.StatusNotFound, wantID: &id},
		{name: "returns 500 when the store fails", target: "/activities/" + id.String(), storeErr: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantID: &id},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{err: tt.storeErr}

			rec := httptest.NewRecorder()
			New(store).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, tt.target, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)
			if rec.Code != http.StatusNoContent {
				assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			}
			if rec.Code >= http.StatusBadRequest {
				var body struct{ Error string }
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.NotEmpty(t, body.Error)
			}
			assert.Equal(t, tt.wantID, store.gotID)
		})
	}
}
