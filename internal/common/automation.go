package common

import (
	"time"

	"github.com/google/uuid"
)

// GroupPolicy describes an install package set and the operating-system selector
// used to target a group of clients. It is intentionally declarative; it does not
// execute arbitrary shell commands.
type GroupPolicy struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	Name        string          `db:"name" json:"name"`
	Description string          `db:"description" json:"description"`
	OSFilter    string          `db:"os_filter" json:"os_filter"`
	Files       JSONStringSlice `db:"files" json:"files"`
	Mode        string          `db:"mode" json:"mode"`
	Enabled     bool            `db:"enabled" json:"enabled"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}

// Schedule is an interval-based schedule associated with a GroupPolicy. Interval
// schedules avoid pretending to support arbitrary cron syntax before that parser
// and its time-zone semantics are implemented.
type Schedule struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	PolicyID        uuid.UUID  `db:"policy_id" json:"policy_id"`
	Name            string     `db:"name" json:"name"`
	IntervalMinutes int        `db:"interval_minutes" json:"interval_minutes"`
	Enabled         bool       `db:"enabled" json:"enabled"`
	LastRun         *time.Time `db:"last_run" json:"last_run,omitempty"`
	NextRun         time.Time  `db:"next_run" json:"next_run"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}
