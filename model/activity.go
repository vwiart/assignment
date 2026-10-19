package model

import (
	"time"
	"uuid"
)

// Activity represents a user activity.
type Activity struct {
	ID        uuid.UUID `json:"id"`
	UserID    string    `json:"user_id"`
	Date      time.Time `json:"date"`
	StepCount int       `json:"step_count"`
}
