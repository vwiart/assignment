package activity

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"uuid"

	"github.com/go-chi/chi/v5"
	"github.com/vwiart/assignment/model"
	storage "github.com/vwiart/assignment/storage/activity"
)

type activity struct {
	store Storage
}

// New instantiates a new activity.
func New(s Storage) activity {
	return activity{store: s}
}

// Storage are all the methods we need to access the repository layer.
type Storage interface {
	Aggregate(ctx context.Context, opts storage.AggregateOptions) (model.Aggregate, error)
	Delete(ctx context.Context, activityID uuid.UUID) error
	List(ctx context.Context) ([]model.Activity, error)
	Submit(ctx context.Context, activity model.Activity) (model.Aggregate, error)
}

// Routes returns a router serving every activity endpoint.
func (a activity) Routes() http.Handler {
	mux := chi.NewMux()
	mux.Get("/activities", a.ListHandler())
	mux.Get("/activities/total", a.Aggregate())
	mux.Delete("/activities/{activity_id}", a.DeleteHandler())
	mux.Post("/users/{user_id}", a.SubmitHandler())
	mux.Get("/users/{user_id}/total", a.AggregateByUser())
	return mux
}

// errorResponse is the body of every error response.
type errorResponse struct {
	Error string `json:"error"`
}

// writeJSON writes v as a JSON body with the given status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Default().Error("unable to encode body", "err", err)
	}
}

// writeError writes an errorResponse with the given status.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
