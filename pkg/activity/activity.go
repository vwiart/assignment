package activity

import (
	"context"
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
