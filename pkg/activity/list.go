package activity

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"uuid"

	storage "github.com/vwiart/assignment/storage/activity"
)

const (
	// defaultListLimit is the page size when the limit query parameter is not set.
	defaultListLimit = 100
	// maxListLimit is the largest page size a client can ask for.
	maxListLimit = 1000
)

// ListHandler is an endpoint to list activities, one page at a time.
func (a activity) ListHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := slog.Default()

		opts, err := parseListOptions(r.URL.Query())
		if err != nil {
			logger.Error("invalid input", "err", err)
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		activities, err := a.store.List(r.Context(), opts)
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

// parseListOptions reads the optional limit and after query parameters.
func parseListOptions(query url.Values) (storage.ListOptions, error) {
	opts := storage.ListOptions{Limit: defaultListLimit}
	if s := query.Get("limit"); s != "" {
		limit, err := strconv.Atoi(s)
		if err != nil || limit < 1 || limit > maxListLimit {
			return opts, fmt.Errorf("limit must be between 1 and %d", maxListLimit)
		}
		opts.Limit = limit
	}
	if s := query.Get("after"); s != "" {
		after, err := uuid.Parse(s)
		if err != nil {
			return opts, errors.New("after must be an activity id")
		}
		opts.After = after
	}
	return opts, nil
}
