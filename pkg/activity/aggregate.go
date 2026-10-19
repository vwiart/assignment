package activity

import (
	"encoding/json"
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
		logger := slog.Default()

		from, err := parseTimeInput(r.URL.Query().Get("from"))
		if err != nil {
			logger.Error("invalid input format", "err", err)
			http.Error(w, "invalid input format", http.StatusBadRequest)
			return
		}

		to, err := parseTimeInput(r.URL.Query().Get("to"))
		if err != nil {
			logger.Error("invalid input format", "err", err)
			http.Error(w, "invalid input format", http.StatusBadRequest)
			return
		}

		if !from.IsZero() && !to.IsZero() && to.Before(from) {
			logger.Error("invalid input format", "from", from, "to", to)
			http.Error(w, "invalid input format", http.StatusBadRequest)
			return
		}

		aggregate, err := a.store.Aggregate(r.Context(), storage.AggregateOptions{
			UserID: chi.URLParam(r, "user_id"),
			From:   from,
			To:     to,
		})
		if errors.Is(err, storage.ErrUserNotFound) {
			logger.Error("unable to aggregate activities", "err", err)
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		if err != nil {
			logger.Error("unable to list activities", "err", err)
			http.Error(w, "unable to list activities", http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(aggregate); err != nil {
			http.Error(w, "failed to encode body", http.StatusInternalServerError)
			return
		}
	}
}

// AggregateByUser aggregates the activities for all users.
func (a activity) Aggregate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := slog.Default()

		from, err := parseTimeInput(r.URL.Query().Get("from"))
		if err != nil {
			logger.Error("invalid input format", "err", err)
			http.Error(w, "invalid input format", http.StatusBadRequest)
			return
		}

		to, err := parseTimeInput(r.URL.Query().Get("to"))
		if err != nil {
			logger.Error("invalid input format", "err", err)
			http.Error(w, "invalid input format", http.StatusBadRequest)
			return
		}

		if !from.IsZero() && !to.IsZero() && to.Before(from) {
			logger.Error("invalid input format", "from", from, "to", to)
			http.Error(w, "invalid input format", http.StatusBadRequest)
			return
		}

		aggregate, err := a.store.Aggregate(r.Context(), storage.AggregateOptions{
			From: from,
			To:   to,
		})
		if err != nil {
			logger.Error("unable to list activities", "err", err)
			http.Error(w, "unable to list activities", http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(aggregate); err != nil {
			http.Error(w, "failed to encode body", http.StatusInternalServerError)
			return
		}
	}
}

func parseTimeInput(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	return time.Parse(time.RFC3339, s)
}
