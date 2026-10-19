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
func (a Handler) DeleteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := slog.Default()

		id, err := uuid.Parse(chi.URLParam(r, "activity_id"))
		if err != nil {
			logger.Warn("invalid input", "err", err)
			writeError(w, http.StatusBadRequest, "invalid input")
			return
		}
		err = a.store.Delete(ctx, id)
		if errors.Is(err, storage.ErrActivityNotFound) {
			logger.Warn("failed to delete", "err", err)
			writeError(w, http.StatusNotFound, "activity not found")
			return
		}
		if err != nil {
			logger.Error("failed to delete", "err", err)
			writeError(w, http.StatusInternalServerError, "failed to delete")
			return
		}
		logger.Info("deleted activity", "id", id)
		w.WriteHeader(http.StatusNoContent)
	}
}
