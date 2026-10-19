package activity

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
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
		{name: "deletes the activity", target: "/users/" + id.String(), wantStatus: http.StatusOK, wantID: &id},
		{name: "rejects an id that is not a UUID", target: "/users/42", wantStatus: http.StatusBadRequest},
		{name: "returns 500 when the store fails", target: "/users/" + id.String(), storeErr: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantID: &id},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{err: tt.storeErr}

			rec := httptest.NewRecorder()
			New(store).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, tt.target, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantID, store.gotID)
		})
	}
}
