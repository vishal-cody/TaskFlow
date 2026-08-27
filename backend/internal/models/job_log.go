package models

import (
	"time"

	"github.com/google/uuid"
)

// joblog represents a single log entry for a job's execution history.
type JobLog struct {
	ID        int64     `json:"id"`
	JobID     uuid.UUID `json:"job_id"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"timestamp"` // JSON key is "timestamp" for API consumers
}
