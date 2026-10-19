package activity

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
	"uuid"

	"github.com/go-chi/chi/v5"
	"github.com/vwiart/assignment/model"
	storage "github.com/vwiart/assignment/storage/activity"
)

// SubmitHandler is an endpoint to submit an activity.
func (a activity) SubmitHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := slog.Default()
		var input submitInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "unable to decode body")
			return
		}
		activity, err := input.normalize(chi.URLParam(r, "user_id"))
		if err != nil {
			logger.Error("failed to normalize body", "err", err)
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		logger.Info("submitting", "activity", activity)
		aggregate, err := a.store.Submit(ctx, activity)
		if errors.Is(err, storage.ErrUserNotFound) {
			logger.Error("unable to submit activity", "err", err)
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		if err != nil {
			logger.Error("unable to submit activity", "err", err)
			writeError(w, http.StatusInternalServerError, "unable to submit activity")
			return
		}

		writeJSON(w, http.StatusOK, submitResponse{ID: activity.ID, Total: aggregate.Total})
	}
}

// submitResponse is the body of a successful submission: the ID of the new activity, so the client
// can delete it later, and the total across all users.
type submitResponse struct {
	ID    uuid.UUID `json:"id"`
	Total int       `json:"total"`
}

// maxStepCount is the largest step count accepted for one submission. It matches the database
// CHECK constraint.
const maxStepCount = 100_000

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
	if input.StepCount < 1 || input.StepCount > maxStepCount {
		errs = append(errs, fmt.Errorf("step count must be between 1 and %d", maxStepCount))
	}
	activity.StepCount = input.StepCount
	d, err := time.Parse(time.RFC3339, input.Date)
	if err != nil {
		errs = append(errs, errors.New("date must be a valid RFC3339"))
	}
	activity.Date = d
	return activity, errors.Join(errs...)
}
