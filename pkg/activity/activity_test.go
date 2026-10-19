package activity

import (
	"context"
	"uuid"

	"github.com/vwiart/assignment/model"
	storage "github.com/vwiart/assignment/storage/activity"
)

// fakeStore returns canned results and records the arguments it receives.
type fakeStore struct {
	aggregate  model.Aggregate
	activities []model.Activity
	err        error

	gotOpts     *storage.AggregateOptions
	gotListOpts *storage.ListOptions
	gotActivity *model.Activity
	gotID       *uuid.UUID
}

func (f *fakeStore) Aggregate(_ context.Context, opts storage.AggregateOptions) (model.Aggregate, error) {
	f.gotOpts = &opts
	return f.aggregate, f.err
}

func (f *fakeStore) Delete(_ context.Context, id uuid.UUID) error {
	f.gotID = &id
	return f.err
}

func (f *fakeStore) List(_ context.Context, opts storage.ListOptions) ([]model.Activity, error) {
	f.gotListOpts = &opts
	return f.activities, f.err
}

func (f *fakeStore) Submit(_ context.Context, activity model.Activity) (model.Aggregate, error) {
	f.gotActivity = &activity
	return f.aggregate, f.err
}
