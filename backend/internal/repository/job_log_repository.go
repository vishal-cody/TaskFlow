package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vishalyadav/jobplatform/internal/models"
)

type JobLogRepository struct {
	pool *pgxpool.Pool
}

func NewJobLogRepository(pool *pgxpool.Pool) *JobLogRepository {
	return &JobLogRepository{pool: pool}
}

// create inserts a new job log entry.
func (r *JobLogRepository) Create(ctx context.Context, log *models.JobLog) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO job_logs (job_id, level, message)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		log.JobID, log.Level, log.Message,
	).Scan(&log.ID, &log.CreatedAt)

	if err != nil {
		return fmt.Errorf("inserting job log: %w", err)
	}
	return nil
}

// listbyjobid returns all log entries for a job, ordered chronologically.
func (r *JobLogRepository) ListByJobID(ctx context.Context, jobID uuid.UUID) ([]models.JobLog, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, job_id, level, message, created_at
		 FROM job_logs
		 WHERE job_id = $1
		 ORDER BY created_at ASC`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying job logs: %w", err)
	}
	defer rows.Close()

	logs := make([]models.JobLog, 0)
	for rows.Next() {
		var l models.JobLog
		if err := rows.Scan(&l.ID, &l.JobID, &l.Level, &l.Message, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning job log row: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}
