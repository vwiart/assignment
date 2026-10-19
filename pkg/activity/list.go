package activity

import (
	"log/slog"
	"net/http"
)

// ListHandler is an endpoint to list all activities.
func (a activity) ListHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := slog.Default()

		activities, err := a.store.List(r.Context())
		if err != nil {
			logger.Error("unable to list activities", "err", err)
			writeError(w, http.StatusInternalServerError, "unable to list activities")
			return
		}
		if len(activities) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSON(w, http.StatusOK, activities)
	}
}
