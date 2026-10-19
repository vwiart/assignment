package activity

import (
	"log/slog"
	"net/http"
	"uuid"

	"github.com/go-chi/chi/v5"
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
		if err := a.store.Delete(ctx, id); err != nil {
			logger.Error("failed to delete", "err", err)
			http.Error(w, "failed to delete", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
