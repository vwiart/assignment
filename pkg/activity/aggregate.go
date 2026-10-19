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
func (a activity) AggregateByUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.aggregate(w, r, chi.URLParam(r, "user_id"))
	}
}

// Aggregate aggregates the activities for all users.
func (a activity) Aggregate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.aggregate(w, r, "")
	}
}

// aggregate writes the total for userID, or for all users when userID is empty, limited to the
// optional from and to query parameters.
func (a activity) aggregate(w http.ResponseWriter, r *http.Request, userID string) {
	logger := slog.Default()

	from, err := parseTimeInput(r.URL.Query().Get("from"))
	if err != nil {
		logger.Error("invalid input format", "err", err)
		writeError(w, http.StatusBadRequest, "invalid input format")
		return
	}

	to, err := parseTimeInput(r.URL.Query().Get("to"))
	if err != nil {
		logger.Error("invalid input format", "err", err)
		writeError(w, http.StatusBadRequest, "invalid input format")
		return
	}

	if !from.IsZero() && !to.IsZero() && to.Before(from) {
		logger.Error("invalid input format", "from", from, "to", to)
		writeError(w, http.StatusBadRequest, "invalid input format")
		return
	}

	aggregate, err := a.store.Aggregate(r.Context(), storage.AggregateOptions{
		UserID: userID,
		From:   from,
		To:     to,
	})
	if errors.Is(err, storage.ErrUserNotFound) {
		logger.Error("unable to aggregate activities", "err", err)
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		logger.Error("unable to aggregate activities", "err", err)
		writeError(w, http.StatusInternalServerError, "unable to aggregate activities")
		return
	}
	writeJSON(w, http.StatusOK, aggregate)
}

func parseTimeInput(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	return time.Parse(time.RFC3339, s)
}
