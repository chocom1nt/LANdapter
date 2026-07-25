package common

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)


type JSONStringSlice []string

func (j *JSONStringSlice) Scan(value interface{}) error {
    if value == nil {
        *j = nil
        return nil
    }
    b, ok := value.([]byte)
    if !ok {
        return fmt.Errorf("expected []byte, got %T", value)
    }
    return json.Unmarshal(b, j)
}

func (j JSONStringSlice) Value() (driver.Value, error) {
    if j == nil {
        return nil, nil
    }
    return json.Marshal(j)
}

type Client struct {
    ID       uuid.UUID `db:"id" json:"id"`
    Hostname string    `db:"hostname" json:"hostname"`
    OS       string    `db:"os" json:"os"`
    MAC      string    `db:"mac" json:"mac"`
    Online   bool      `db:"online" json:"online"`
    LastSeen time.Time `db:"last_seen" json:"last_seen"`
}

type Job struct {
    ID        uuid.UUID       `db:"id"`
    Files     JSONStringSlice `db:"files"`
    CreatedAt time.Time       `db:"created_at"`
}

type JobResult struct {
    ID         uuid.UUID  `db:"id"`
    JobID      uuid.UUID  `db:"job_id"`
    ClientID   uuid.UUID  `db:"client_id"`
    Status     string     `db:"status"`
    Output     *string    `db:"output"`
    Error      *string    `db:"error"`
    StartedAt  *time.Time `db:"started_at"`
    FinishedAt *time.Time `db:"finished_at"`
    SnapshotBefore *json.RawMessage `db:"snapshot_before" json:"snapshot_before,omitempty"`
    SnapshotAfter  *json.RawMessage `db:"snapshot_after" json:"snapshot_after,omitempty"`
}