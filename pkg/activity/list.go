package activity

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// DeleteHandler is an endpoint to list all activities.
func (a activity) ListHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := slog.Default()

		activities, err := a.store.List(r.Context())
		if err != nil {
			logger.Error("unable to list activities", "err", err)
			http.Error(w, "unable to list activities", http.StatusInternalServerError)
			return
		}
		if len(activities) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err := json.NewEncoder(w).Encode(activities); err != nil {
			http.Error(w, "failed to encode body", http.StatusInternalServerError)
			return
		}
	}
}
