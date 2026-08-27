package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// job status constants. these form an explicit state machine —
// only specific transitions are allowed (see validtransition).
const (
	JobStatusQueued     = "queued"
	JobStatusProcessing = "processing"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
	JobStatusRetrying   = "retrying"
	JobStatusCancelled  = "cancelled"
)

// job type constants — the set of job types the platform supports.
const (
	JobTypeReportGeneration = "report_generation"
	JobTypeDataProcessing   = "data_processing"
	JobTypeImageProcessing  = "image_processing"
	JobTypeNotification     = "notification"
)

// validtransitions defines every legal state change.
// transitions not in this map are rejected.
var validTransitions = map[string][]string{
	JobStatusQueued:     {JobStatusProcessing, JobStatusCancelled},
	JobStatusProcessing: {JobStatusCompleted, JobStatusFailed},
	JobStatusFailed:     {JobStatusRetrying},
	JobStatusRetrying:   {JobStatusProcessing},
	// completed and cancelled are terminal — no outgoing transitions
}

// validjobtypes is the set of recognized job types.
var validJobTypes = map[string]bool{
	JobTypeReportGeneration: true,
	JobTypeDataProcessing:   true,
	JobTypeImageProcessing:  true,
	JobTypeNotification:     true,
}

// job represents an asynchronous task submitted to the platform.
type Job struct {
	ID          uuid.UUID        `json:"id"`
	UserID      uuid.UUID        `json:"user_id"`
	Type        string           `json:"type"`
	Status      string           `json:"status"`
	Priority    int              `json:"priority"`
	Payload     json.RawMessage  `json:"payload"`
	Result      *json.RawMessage `json:"result,omitempty"`
	Error       *string          `json:"error,omitempty"`
	Progress    int              `json:"progress"`
	RetryCount  int              `json:"retry_count"`
	MaxRetries  int              `json:"max_retries"`
	CreatedAt   time.Time        `json:"created_at"`
	StartedAt   *time.Time       `json:"started_at,omitempty"`
	CompletedAt *time.Time       `json:"completed_at,omitempty"`
}

// validatetransition checks whether transitioning from → to is allowed.
// returns an error describing why if the transition is invalid.
func ValidateTransition(from, to string) error {
	allowed, exists := validTransitions[from]
	if !exists {
		return fmt.Errorf("status %q is terminal; no transitions allowed", from)
	}
	for _, s := range allowed {
		if s == to {
			return nil
		}
	}
	return fmt.Errorf("invalid transition from %q to %q", from, to)
}

// isvalidjobtype returns true if the given type string is a supported job type.
func IsValidJobType(jobType string) bool {
	return validJobTypes[jobType]
}

// isterminalstatus returns true if the job is in a final state.
func IsTerminalStatus(status string) bool {
	return status == JobStatusCompleted || status == JobStatusCancelled
}

// iscancellable returns true if the job can be cancelled from its current status.
func IsCancellable(status string) bool {
	return status == JobStatusQueued
}

// validatepriority checks that priority is within the allowed range [1, 10].
func ValidatePriority(p int) error {
	if p < 1 || p > 10 {
		return fmt.Errorf("priority must be between 1 and 10, got %d", p)
	}
	return nil
}

// joblistfilters holds query parameters for listing jobs.
type JobListFilters struct {
	Status        *string
	Type          *string
	Priority      *int
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	Page          int
	Limit         int
}

// joblistresult is the paginated response for job listing.
type JobListResult struct {
	Jobs       []Job `json:"jobs"`
	Total      int   `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}
