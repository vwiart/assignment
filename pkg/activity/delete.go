package activity

import (
	"errors"
	"log/slog"
	"net/http"
	"uuid"

	"github.com/go-chi/chi/v5"
	storage "github.com/vwiart/assignment/storage/activity"
)

// DeleteHandler is an endpoint to delete an activity.
func (a activity) DeleteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := slog.Default()

		id, err := uuid.Parse(chi.URLParam(r, "activity_id"))
		if err != nil {
			logger.Error("invalid input", "err", err)
			http.Error(w, "invalid input", http.StatusBadRequest)
			return
		}
		err = a.store.Delete(ctx, id)
		if errors.Is(err, storage.ErrActivityNotFound) {
			logger.Error("failed to delete", "err", err)
			http.Error(w, "activity not found", http.StatusNotFound)
			return
		}
		if err != nil {
			logger.Error("failed to delete", "err", err)
			http.Error(w, "failed to delete", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
