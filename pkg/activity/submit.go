package activity

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"
	"uuid"

	"github.com/go-chi/chi/v5"
	"github.com/vwiart/assignment/model"
)

// SubmitHandler is an endpoint to submit an activity.
func (a activity) SubmitHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := slog.Default()
		var input submitInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "unable to decode body", http.StatusBadRequest)
			return
		}
		activity, err := input.normalize(chi.URLParam(r, "user_id"))
		if err != nil {
			logger.Error("failed to normalize body", "err", err)
			http.Error(w, "failed to normalize body", http.StatusBadRequest)
			return
		}

		logger.Info("submitting", "activity", activity)
		aggregate, err := a.store.Submit(ctx, activity)
		if err != nil {
			logger.Error("unable to submit activity", "err", err)
			http.Error(w, "unable to submit activity", http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(aggregate); err != nil {
			logger.Error("failed to encode body", "err", err)
			http.Error(w, "failed to encode body", http.StatusInternalServerError)
			return
		}
	}
}

type submitInput struct {
	StepCount int    `json:"step_count"`
	Date      string `json:"date"`
}

func (input submitInput) normalize(userID string) (model.Activity, error) {
	var (
		activity model.Activity
		errs     []error
	)
	activity.ID = uuid.NewV7()
	activity.UserID = userID
	if input.StepCount < 0 {
		errs = append(errs, errors.New("step count must be strictly greater than 0"))
	}
	activity.StepCount = input.StepCount
	d, err := time.Parse(time.RFC3339, input.Date)
	if err != nil {
		errs = append(errs, errors.New("date must be a valid RFC3339"))
	}
	activity.Date = d
	return activity, errors.Join(errs...)
}
