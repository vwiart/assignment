package storage

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"uuid"

	"github.com/vwiart/assignment/model"
)

// ErrUserNotFound is returned when an operation refers to a user that does not exist.
var ErrUserNotFound = errors.New("user not found")

// ErrActivityNotFound is returned when an operation refers to an activity that does not exist.
var ErrActivityNotFound = errors.New("activity not found")

type activity struct {
	db *sql.DB
}

// NewActivity instantiates a new activity repository
func NewActivity(db *sql.DB) activity {
	return activity{db: db}
}

// AggregateOptions defines which options are available to aggregate activities.
type AggregateOptions struct {
	UserID   string
	From, To time.Time
}

// Aggregate aggregates the activities for the given input.
func (a activity) Aggregate(ctx context.Context, opts AggregateOptions) (model.Aggregate, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Aggregate{}, err
	}
	agg, err := a.aggregate(ctx, tx, opts)
	if err != nil {
		tx.Rollback()
		return model.Aggregate{}, err
	}

	return agg, tx.Commit()
}

func (a activity) aggregate(ctx context.Context, tx *sql.Tx, opts AggregateOptions) (model.Aggregate, error) {
	if opts.UserID != "" {
		if err := userExists(ctx, tx, opts.UserID); err != nil {
			return model.Aggregate{}, err
		}
	}

	var (
		query strings.Builder
		args  []any
	)
	query.WriteString(`
		SELECT COALESCE(SUM(step_count), 0) AS total
		FROM activity
		WHERE 1=1
	`)
	if opts.UserID != "" {
		query.WriteString(`
			AND user_id = ?
		`)
		args = append(args, opts.UserID)
	}
	if !opts.From.IsZero() {
		query.WriteString(`
			AND date >= ?
		`)
		args = append(args, opts.From.UTC().Format(time.RFC3339))
	}
	if !opts.To.IsZero() {
		query.WriteString(`
			AND date < ?
		`)
		args = append(args, opts.To.UTC().Format(time.RFC3339))
	}

	var agg model.Aggregate
	err := tx.QueryRowContext(ctx, query.String(), args...).Scan(&agg.Total)
	return agg, err
}

// Delete deletes an activity.
func (a activity) Delete(ctx context.Context, activityID uuid.UUID) error {
	rs, err := a.db.ExecContext(ctx, `DELETE FROM activity WHERE id = ?`, activityID)
	if err != nil {
		return err
	}
	v, err := rs.RowsAffected()
	switch {
	case err != nil:
		return err
	case v == 0:
		return ErrActivityNotFound
	case v == 1:
		return nil
	default:
		return errors.New("unexpected amount of rows affected")
	}
}

// ListOptions defines how to page through activities. A zero Limit returns every activity.
type ListOptions struct {
	Limit int
	After uuid.UUID
}

// List lists activities ordered by ID, which is also submission order since IDs are UUIDv7.
func (a activity) List(ctx context.Context, opts ListOptions) ([]model.Activity, error) {
	var (
		query strings.Builder
		args  []any
	)
	query.WriteString(`
		SELECT id, user_id, date, step_count
		FROM activity
		WHERE 1=1
	`)
	if opts.After != uuid.Nil() {
		query.WriteString(`
			AND id > ?
		`)
		args = append(args, opts.After.String())
	}
	query.WriteString(`
		ORDER BY id
	`)
	if opts.Limit > 0 {
		query.WriteString(`
			LIMIT ?
		`)
		args = append(args, opts.Limit)
	}
	rows, err := a.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []model.Activity
	for rows.Next() {
		var id, userID, date string
		var stepCount int
		if err := rows.Scan(&id, &userID, &date, &stepCount); err != nil {
			return nil, err
		}

		aid, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		d, err := time.Parse(time.RFC3339, date)
		if err != nil {
			return nil, err
		}

		data = append(data, model.Activity{
			ID:        aid,
			UserID:    userID,
			Date:      d,
			StepCount: stepCount,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return data, nil
}

// Submit submits an activity and return the updated aggregates.
func (a activity) Submit(ctx context.Context, activity model.Activity) (model.Aggregate, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Aggregate{}, err
	}

	if err := userExists(ctx, tx, activity.UserID); err != nil {
		tx.Rollback()
		return model.Aggregate{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO activity (id, user_id, date, step_count) VALUES (?, ?, ?, ?)
	`, activity.ID, activity.UserID, activity.Date.UTC().Format(time.RFC3339), activity.StepCount); err != nil {
		tx.Rollback()
		return model.Aggregate{}, err
	}

	agg, err := a.aggregate(ctx, tx, AggregateOptions{})
	if err != nil {
		tx.Rollback()
		return model.Aggregate{}, err
	}

	return agg, tx.Commit()
}

// userExists returns ErrUserNotFound when no user has the given ID.
func userExists(ctx context.Context, tx *sql.Tx, userID string) error {
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM user WHERE id = ?)`, userID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrUserNotFound
	}
	return nil
}
