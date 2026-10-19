package activity

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	storage "github.com/vwiart/assignment/storage/activity"
)

// AggregateByUser aggregates the activities for a given user.
func (a Handler) AggregateByUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.aggregate(w, r, chi.URLParam(r, "user_id"))
	}
}

// Aggregate aggregates the activities for all users.
func (a Handler) Aggregate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.aggregate(w, r, "")
	}
}

// aggregate writes the total for userID, or for all users when userID is empty, limited to the
// optional from and to query parameters.
func (a Handler) aggregate(w http.ResponseWriter, r *http.Request, userID string) {
	logger := slog.Default()

	from, err := parseTimeInput(r.URL.Query().Get("from"))
	if err != nil {
		logger.Warn("invalid input format", "err", err)
		writeError(w, http.StatusBadRequest, "invalid input format")
		return
	}

	to, err := parseTimeInput(r.URL.Query().Get("to"))
	if err != nil {
		logger.Warn("invalid input format", "err", err)
		writeError(w, http.StatusBadRequest, "invalid input format")
		return
	}

	if !from.IsZero() && !to.IsZero() && to.Before(from) {
		logger.Warn("invalid input format", "from", from, "to", to)
		writeError(w, http.StatusBadRequest, "invalid input format")
		return
	}

	aggregate, err := a.store.Aggregate(r.Context(), storage.AggregateOptions{
		UserID: userID,
		From:   from,
		To:     to,
	})
	if errors.Is(err, storage.ErrUserNotFound) {
		logger.Warn("unable to aggregate activities", "err", err)
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		logger.Error("unable to aggregate activities", "err", err)
		writeError(w, http.StatusInternalServerError, "unable to aggregate activities")
		return
	}
	logger.Info("aggregated activities", "user_id", userID, "from", from, "to", to, "total", aggregate.Total)
	writeJSON(w, http.StatusOK, aggregate)
}

func parseTimeInput(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	return time.Parse(time.RFC3339, s)
}
