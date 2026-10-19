package activity

import (
	"context"
	"uuid"

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
