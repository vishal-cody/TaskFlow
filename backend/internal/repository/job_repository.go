package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vishalyadav/jobplatform/internal/models"
)

type JobRepository struct {
	pool *pgxpool.Pool
}

func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{pool: pool}
}

// create inserts a new job and returns it with generated fields populated.
// it uses the pool directly (non-transactional).
func (r *JobRepository) Create(ctx context.Context, job *models.Job) error {
	job.ID = uuid.New()
	job.Status = models.JobStatusQueued
	job.Progress = 0
	job.RetryCount = 0

	err := r.pool.QueryRow(ctx,
		`INSERT INTO jobs (id, user_id, type, status, priority, payload, max_retries)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at`,
		job.ID, job.UserID, job.Type, job.Status, job.Priority, job.Payload, job.MaxRetries,
	).Scan(&job.CreatedAt)

	if err != nil {
		return fmt.Errorf("inserting job: %w", err)
	}
	return nil
}

// createintx inserts a new job using an existing transaction.
func (r *JobRepository) CreateInTx(ctx context.Context, tx pgx.Tx, job *models.Job) error {
	job.ID = uuid.New()
	job.Status = models.JobStatusQueued
	job.Progress = 0
	job.RetryCount = 0

	err := tx.QueryRow(ctx,
		`INSERT INTO jobs (id, user_id, type, status, priority, payload, max_retries)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at`,
		job.ID, job.UserID, job.Type, job.Status, job.Priority, job.Payload, job.MaxRetries,
	).Scan(&job.CreatedAt)

	if err != nil {
		return fmt.Errorf("inserting job in tx: %w", err)
	}
	return nil
}

// createwithidempotencykeyintx atomically creates a job and stores the idempotency key using the provided transaction.
// if the key already exists for this user, returns the existing job id without creating a duplicate.
// returns (job, created, error) where created indicates if a new job was made.
func (r *JobRepository) CreateWithIdempotencyKeyInTx(ctx context.Context, tx pgx.Tx, job *models.Job, idempotencyKey string) (bool, error) {
	// check if key already exists for this user.
	var existingJobID uuid.UUID
	err := tx.QueryRow(ctx,
		`SELECT job_id FROM idempotency_keys
		 WHERE key = $1 AND user_id = $2`,
		idempotencyKey, job.UserID,
	).Scan(&existingJobID)

	if err == nil {
		// key exists — load and return the existing job.
		job.ID = existingJobID
		err = r.scanJob(ctx, tx, job)
		if err != nil {
			return false, fmt.Errorf("loading existing job for idempotency key: %w", err)
		}
		return false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("checking idempotency key: %w", err)
	}

	// key doesn't exist — create the job.
	err = r.CreateInTx(ctx, tx, job)
	if err != nil {
		return false, fmt.Errorf("creating job with idempotency key: %w", err)
	}

	// store the idempotency key.
	_, err = tx.Exec(ctx,
		`INSERT INTO idempotency_keys (key, user_id, job_id)
		 VALUES ($1, $2, $3)`,
		idempotencyKey, job.UserID, job.ID,
	)
	if err != nil {
		return false, fmt.Errorf("inserting idempotency key: %w", err)
	}

	return true, nil
}

// scanjob loads a full job by id using the given querier (pool or tx).
func (r *JobRepository) scanJob(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}, job *models.Job) error {
	return q.QueryRow(ctx,
		`SELECT id, user_id, type, status, priority, payload,
		        result, error, progress, retry_count, max_retries,
		        created_at, started_at, completed_at
		 FROM jobs WHERE id = $1`,
		job.ID,
	).Scan(
		&job.ID, &job.UserID, &job.Type, &job.Status, &job.Priority, &job.Payload,
		&job.Result, &job.Error, &job.Progress, &job.RetryCount, &job.MaxRetries,
		&job.CreatedAt, &job.StartedAt, &job.CompletedAt,
	)
}

// getbyid returns a job by its id. returns nil if not found.
func (r *JobRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Job, error) {
	var job models.Job
	job.ID = id
	err := r.scanJob(ctx, r.pool, &job)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying job by ID: %w", err)
	}
	return &job, nil
}

// listbyuserid returns a paginated, filtered list of jobs for a user.
func (r *JobRepository) ListByUserID(ctx context.Context, userID uuid.UUID, filters models.JobListFilters) (*models.JobListResult, error) {
	// build where clause dynamically.
	where := []string{"user_id = $1"}
	args := []interface{}{userID}
	argIdx := 2

	if filters.Status != nil {
		where = append(where, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.Type != nil {
		where = append(where, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, *filters.Type)
		argIdx++
	}
	if filters.Priority != nil {
		where = append(where, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *filters.Priority)
		argIdx++
	}
	if filters.CreatedAfter != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filters.CreatedAfter)
		argIdx++
	}
	if filters.CreatedBefore != nil {
		where = append(where, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filters.CreatedBefore)
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")

	// count total matching rows.
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM jobs WHERE %s", whereClause)
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("counting jobs: %w", err)
	}

	// fetch the page.
	offset := (filters.Page - 1) * filters.Limit
	dataQuery := fmt.Sprintf(
		`SELECT id, user_id, type, status, priority, payload,
		        result, error, progress, retry_count, max_retries,
		        created_at, started_at, completed_at
		 FROM jobs
		 WHERE %s
		 ORDER BY priority DESC, created_at DESC
		 LIMIT $%d OFFSET $%d`,
		whereClause, argIdx, argIdx+1,
	)
	args = append(args, filters.Limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("querying jobs: %w", err)
	}
	defer rows.Close()

	jobs := make([]models.Job, 0)
	for rows.Next() {
		var j models.Job
		if err := rows.Scan(
			&j.ID, &j.UserID, &j.Type, &j.Status, &j.Priority, &j.Payload,
			&j.Result, &j.Error, &j.Progress, &j.RetryCount, &j.MaxRetries,
			&j.CreatedAt, &j.StartedAt, &j.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning job row: %w", err)
		}
		jobs = append(jobs, j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating job rows: %w", err)
	}

	totalPages := total / filters.Limit
	if total%filters.Limit > 0 {
		totalPages++
	}

	return &models.JobListResult{
		Jobs:       jobs,
		Total:      total,
		Page:       filters.Page,
		Limit:      filters.Limit,
		TotalPages: totalPages,
	}, nil
}

// updatestatus atomically transitions a job from one status to another.
// uses where status = fromstatus so only one worker can claim a job.
// returns an error if no rows were affected (job was in a different state).
func (r *JobRepository) UpdateStatus(ctx context.Context, jobID uuid.UUID, fromStatus, toStatus string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE jobs SET status = $1 WHERE id = $2 AND status = $3`,
		toStatus, jobID, fromStatus,
	)
	if err != nil {
		return fmt.Errorf("updating job status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("job %s is not in status %q (possibly already claimed)", jobID, fromStatus)
	}
	return nil
}

// markstarted sets the started_at timestamp and transitions to processing.
func (r *JobRepository) MarkStarted(ctx context.Context, jobID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE jobs SET status = $1, started_at = NOW()
		 WHERE id = $2 AND status = $3`,
		models.JobStatusProcessing, jobID, models.JobStatusQueued,
	)
	if err != nil {
		return fmt.Errorf("marking job started: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("job %s is not queued (possibly already claimed)", jobID)
	}
	return nil
}

// markcompleted sets the result, progress to 100, and transitions to completed.
func (r *JobRepository) MarkCompleted(ctx context.Context, jobID uuid.UUID, result []byte) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE jobs SET status = $1, progress = 100, result = $2, completed_at = NOW()
		 WHERE id = $3 AND status = $4`,
		models.JobStatusCompleted, result, jobID, models.JobStatusProcessing,
	)
	if err != nil {
		return fmt.Errorf("marking job completed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("job %s is not processing", jobID)
	}
	return nil
}

// markfailed sets the error message and transitions to failed.
func (r *JobRepository) MarkFailed(ctx context.Context, jobID uuid.UUID, jobErr string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE jobs SET status = $1, error = $2, completed_at = NOW()
		 WHERE id = $3 AND status = $4`,
		models.JobStatusFailed, jobErr, jobID, models.JobStatusProcessing,
	)
	if err != nil {
		return fmt.Errorf("marking job failed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("job %s is not processing", jobID)
	}
	return nil
}

// incrementretry bumps retry_count and transitions from failed → retrying.
func (r *JobRepository) IncrementRetry(ctx context.Context, jobID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE jobs SET status = $1, retry_count = retry_count + 1
		 WHERE id = $2 AND status = $3`,
		models.JobStatusRetrying, jobID, models.JobStatusFailed,
	)
	if err != nil {
		return fmt.Errorf("incrementing retry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("job %s is not failed", jobID)
	}
	return nil
}

// updateprogress sets the progress percentage (0-100).
func (r *JobRepository) UpdateProgress(ctx context.Context, jobID uuid.UUID, progress int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs SET progress = $1 WHERE id = $2`,
		progress, jobID,
	)
	if err != nil {
		return fmt.Errorf("updating progress: %w", err)
	}
	return nil
}

// getstats returns job count aggregates for a user (used by dashboard).
func (r *JobRepository) GetStats(ctx context.Context, userID uuid.UUID) (map[string]int, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT status, COUNT(*)
		 FROM jobs
		 WHERE user_id = $1
		 GROUP BY status`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying job stats: %w", err)
	}
	defer rows.Close()

	stats := map[string]int{
		"total":      0,
		"queued":     0,
		"processing": 0,
		"completed":  0,
		"failed":     0,
		"retrying":   0,
		"cancelled":  0,
	}

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scanning stats row: %w", err)
		}
		stats[status] = count
		stats["total"] += count
	}
	return stats, rows.Err()
}

// pool exposes the underlying pgxpool for transaction management by the service layer.
func (r *JobRepository) Pool() *pgxpool.Pool {
	return r.pool
}
