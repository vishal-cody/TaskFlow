package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/vishalyadav/jobplatform/internal/metrics"
	"github.com/vishalyadav/jobplatform/internal/models"
	"github.com/vishalyadav/jobplatform/internal/queue"
	"github.com/vishalyadav/jobplatform/internal/repository"
)

// all ownership checks and state machine enforcement happen here.
type JobOrchestrator struct {
	jobRepo    *repository.JobRepository
	jobLogRepo *repository.JobLogRepository
	outboxRepo *repository.OutboxRepository
	logger     *slog.Logger
}

func NewJobOrchestrator(
	jobRepo *repository.JobRepository,
	jobLogRepo *repository.JobLogRepository,
	outboxRepo *repository.OutboxRepository,
	logger *slog.Logger,
) *JobOrchestrator {
	return &JobOrchestrator{
		jobRepo:    jobRepo,
		jobLogRepo: jobLogRepo,
		outboxRepo: outboxRepo,
		logger:     logger,
	}
}

// createjobrequest holds input for job creation.
type CreateJobRequest struct {
	Type       string          `json:"type"`
	Priority   int             `json:"priority"`
	Payload    json.RawMessage `json:"payload"`
	MaxRetries *int            `json:"max_retries,omitempty"`
}

// createjobresponse is the immediate response after job creation.
type CreateJobResponse struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
}

// createjob validates input, handles idempotency, and creates a new job.
func (s *JobOrchestrator) CreateJob(ctx context.Context, userID uuid.UUID, req CreateJobRequest, idempotencyKey string) (*CreateJobResponse, error) {
	// validate job type.
	if !models.IsValidJobType(req.Type) {
		return nil, &BadRequestError{Message: fmt.Sprintf("unsupported job type: %s", req.Type)}
	}

	// default and validate priority.
	if req.Priority == 0 {
		req.Priority = 5
	}
	if err := models.ValidatePriority(req.Priority); err != nil {
		return nil, &BadRequestError{Message: err.Error()}
	}

	// validate payload is at least valid json.
	if len(req.Payload) == 0 {
		req.Payload = json.RawMessage(`{}`)
	}
	if !json.Valid(req.Payload) {
		return nil, &BadRequestError{Message: "payload must be valid JSON"}
	}

	maxRetries := 3
	if req.MaxRetries != nil {
		if *req.MaxRetries < 0 || *req.MaxRetries > 10 {
			return nil, &BadRequestError{Message: "max_retries must be between 0 and 10"}
		}
		maxRetries = *req.MaxRetries
	}

	job := &models.Job{
		UserID:     userID,
		Type:       req.Type,
		Priority:   req.Priority,
		Payload:    req.Payload,
		MaxRetries: maxRetries,
	}

	tx, err := s.jobRepo.Pool().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// 1. create the job
	if idempotencyKey != "" {
		created, err := s.jobRepo.CreateWithIdempotencyKeyInTx(ctx, tx, job, idempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("creating job with idempotency key: %w", err)
		}
		if !created {
			s.logger.InfoContext(ctx, "returning existing job for idempotency key",
				"job_id", job.ID,
				"idempotency_key", idempotencyKey,
			)
			// don't emit a new outbox event if we're just returning an existing job.
			return &CreateJobResponse{ID: job.ID, Status: job.Status}, nil
		}
	} else {
		if err := s.jobRepo.CreateInTx(ctx, tx, job); err != nil {
			return nil, fmt.Errorf("creating job: %w", err)
		}
	}

	// 2. create the outbox event
	payload, err := queue.BuildMessagePayload(job.ID.String())
	if err != nil {
		return nil, fmt.Errorf("building message payload: %w", err)
	}

	outboxEvent := &models.OutboxEvent{
		AggregateType: "job",
		AggregateID:   job.ID.String(),
		EventType:     models.OutboxEventJobCreated,
		Payload:       payload,
	}

	if err := s.outboxRepo.InsertInTx(ctx, tx, outboxEvent); err != nil {
		return nil, fmt.Errorf("inserting outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	s.logger.InfoContext(ctx, "job created",
		"job_id", job.ID,
		"type", job.Type,
		"priority", job.Priority,
		"user_id", userID,
	)

	metrics.JobsCreatedTotal.WithLabelValues(job.Type).Inc()

	return &CreateJobResponse{
		ID:     job.ID,
		Status: job.Status,
	}, nil
}

// getjob retrieves a job, enforcing ownership.
func (s *JobOrchestrator) GetJob(ctx context.Context, userID, jobID uuid.UUID) (*models.Job, error) {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("getting job: %w", err)
	}
	if job == nil {
		return nil, &NotFoundError{Message: "job not found"}
	}
	if job.UserID != userID {
		return nil, &NotFoundError{Message: "job not found"} // don't reveal existence
	}
	return job, nil
}

// listjobs returns a paginated list of the user's jobs.
func (s *JobOrchestrator) ListJobs(ctx context.Context, userID uuid.UUID, filters models.JobListFilters) (*models.JobListResult, error) {
	// clamp pagination values.
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Limit < 1 {
		filters.Limit = 20
	}
	if filters.Limit > 100 {
		filters.Limit = 100
	}

	result, err := s.jobRepo.ListByUserID(ctx, userID, filters)
	if err != nil {
		return nil, fmt.Errorf("listing jobs: %w", err)
	}
	return result, nil
}

// canceljob transitions a job to cancelled, enforcing ownership and state machine.
func (s *JobOrchestrator) CancelJob(ctx context.Context, userID, jobID uuid.UUID) error {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("getting job for cancellation: %w", err)
	}
	if job == nil {
		return &NotFoundError{Message: "job not found"}
	}
	if job.UserID != userID {
		return &NotFoundError{Message: "job not found"}
	}

	if !models.IsCancellable(job.Status) {
		return &BadRequestError{
			Message: fmt.Sprintf("cannot cancel job in status %q", job.Status),
		}
	}

	if err := models.ValidateTransition(job.Status, models.JobStatusCancelled); err != nil {
		return &BadRequestError{Message: err.Error()}
	}

	if err := s.jobRepo.UpdateStatus(ctx, jobID, job.Status, models.JobStatusCancelled); err != nil {
		return fmt.Errorf("cancelling job: %w", err)
	}

	// write a log entry for the cancellation.
	_ = s.jobLogRepo.Create(ctx, &models.JobLog{
		JobID:   jobID,
		Level:   "INFO",
		Message: "Job cancelled by user",
	})

	s.logger.InfoContext(ctx, "job cancelled",
		"job_id", jobID,
		"user_id", userID,
	)

	return nil
}

// getjoblogs retrieves execution logs for a job, enforcing ownership.
func (s *JobOrchestrator) GetJobLogs(ctx context.Context, userID, jobID uuid.UUID) ([]models.JobLog, error) {
	// ownership check via getjob.
	if _, err := s.GetJob(ctx, userID, jobID); err != nil {
		return nil, err
	}

	logs, err := s.jobLogRepo.ListByJobID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("getting job logs: %w", err)
	}
	return logs, nil
}

// getstats returns job count aggregates for the user's dashboard.
func (s *JobOrchestrator) GetStats(ctx context.Context, userID uuid.UUID) (map[string]int, error) {
	stats, err := s.jobRepo.GetStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting stats: %w", err)
	}
	return stats, nil
}
