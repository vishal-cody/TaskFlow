package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/vishalyadav/jobplatform/internal/metrics"
	"github.com/vishalyadav/jobplatform/internal/models"
	"github.com/vishalyadav/jobplatform/internal/queue"
	"github.com/vishalyadav/jobplatform/internal/repository"
)

// consumer manages pulling messages from rabbitmq and routing them to processors.
type Consumer struct {
	amqpConn    *queue.Connection
	jobRepo     *repository.JobRepository
	jobLogRepo  *repository.JobLogRepository
	registry    *ProcessorRegistry
	logger      *slog.Logger
	concurrency int
}

func NewConsumer(
	amqpConn *queue.Connection,
	jobRepo *repository.JobRepository,
	jobLogRepo *repository.JobLogRepository,
	registry *ProcessorRegistry,
	logger *slog.Logger,
	concurrency int,
) *Consumer {
	return &Consumer{
		amqpConn:    amqpConn,
		jobRepo:     jobRepo,
		jobLogRepo:  jobLogRepo,
		registry:    registry,
		logger:      logger.With("component", "worker"),
		concurrency: concurrency,
	}
}

// start begins consuming messages from the queue. it blocks until ctx is canceled.
func (c *Consumer) Start(ctx context.Context) error {
	ch := c.amqpConn.Channel()

	// set qos (prefetch) to control concurrency.
	if err := ch.Qos(c.concurrency, 0, false); err != nil {
		return err
	}

	msgs, err := ch.Consume(
		queue.QueueProcess, // queue
		"",                 // consumer
		false,              // auto-ack
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,                // args
	)
	if err != nil {
		return err
	}

	c.logger.Info("worker started consuming", "concurrency", c.concurrency)

	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("shutting down worker, waiting for active jobs to finish")
			wg.Wait()
			return nil
		case msg, ok := <-msgs:
			if !ok {
				c.logger.Info("rabbitmq channel closed, shutting down worker")
				wg.Wait()
				return nil
			}

			wg.Add(1)
			go func(m amqp.Delivery) {
				defer wg.Done()
				c.handleMessage(ctx, m)
			}(msg)
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg amqp.Delivery) {
	// 1. parse payload.
	var payload queue.MessagePayload
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		c.logger.Error("failed to parse message body, rejecting", "error", err)
		msg.Reject(false) // dlq
		return
	}

	jobID, err := uuid.Parse(payload.JobID)
	if err != nil {
		c.logger.Error("invalid job ID, rejecting", "error", err, "job_id", payload.JobID)
		msg.Reject(false) // dlq
		return
	}

	logger := c.logger.With("job_id", jobID)

	// 2. fetch the job from db.
	job, err := c.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		logger.Error("failed to fetch job", "error", err)
		msg.Nack(false, true) // transient db error, requeue
		return
	}
	if job == nil {
		logger.Warn("job not found, acknowledging and dropping")
		msg.Ack(false)
		return
	}

	// 3. mark started (state machine check).
	// if the job isn't queued (or retrying), someone else picked it up or it was cancelled.
	if job.Status == models.JobStatusQueued || job.Status == models.JobStatusRetrying {
		err = c.jobRepo.MarkStarted(ctx, job.ID)
		if err != nil {
			logger.Warn("could not mark job as started, dropping message", "error", err)
			msg.Ack(false) // job already processing or cancelled
			return
		}
		// write log
		_ = c.jobLogRepo.Create(ctx, &models.JobLog{
			JobID: jobID, Level: "INFO", Message: "Job started processing",
		})
	} else {
		// not in a state we can process
		msg.Ack(false)
		return
	}

	// re-fetch job to have up-to-date state (like started_at)
	job, _ = c.jobRepo.GetByID(ctx, jobID)

	// 4. find processor.
	processor, err := c.registry.Get(job.Type)
	if err != nil {
		logger.Error("no processor found for job type", "type", job.Type)
		c.failJob(ctx, job, msg, err.Error())
		return
	}

	// 5. execute.
	// create a context with a timeout for the job execution.
	jobCtx, cancelJob := context.WithTimeout(ctx, 30*time.Minute)
	defer cancelJob()

	// spawn a goroutine to poll for cancellation from the database.
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-jobCtx.Done():
				return // job finished or timeout
			case <-ticker.C:
				currentJob, err := c.jobRepo.GetByID(context.Background(), job.ID)
				if err == nil && currentJob != nil && currentJob.Status == models.JobStatusCancelled {
					logger.Info("job was cancelled via API, aborting execution")
					cancelJob()
					return
				}
			}
		}
	}()

	execJobCtx := &JobContext{
		Context: jobCtx,
		Job:     job,
		UpdateProgress: func(updateCtx context.Context, progress int) error {
			return c.jobRepo.UpdateProgress(updateCtx, job.ID, progress)
		},
	}

	metrics.JobsInFlight.Inc()
	start := time.Now()

	// use a panic recovery wrapper for execution.
	result, execErr := c.executeSafe(execJobCtx, processor)

	duration := time.Since(start).Seconds()
	metrics.JobsInFlight.Dec()
	metrics.JobProcessingDuration.WithLabelValues(job.Type).Observe(duration)

	// 6. handle result.
	if execErr != nil {
		c.handleFailure(ctx, job, msg, execErr.Error(), duration)
	} else {
		c.completeJob(ctx, job, msg, result, duration)
	}
}

func (c *Consumer) executeSafe(jobCtx *JobContext, p JobProcessor) (result []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during job execution: %v", r)
		}
	}()
	return p.Process(jobCtx)
}

func (c *Consumer) completeJob(ctx context.Context, job *models.Job, msg amqp.Delivery, result []byte, durationSec float64) {
	logger := c.logger.With(
		"job_id", job.ID,
		"type", job.Type,
		"duration_sec", durationSec,
	)

	if err := c.jobRepo.MarkCompleted(ctx, job.ID, result); err != nil {
		logger.Error("failed to mark job completed", "error", err)
		msg.Nack(false, true) // requeue
		return
	}

	_ = c.jobLogRepo.Create(ctx, &models.JobLog{
		JobID: job.ID, Level: "INFO", Message: fmt.Sprintf("Job completed successfully in %.2fs", durationSec),
	})

	metrics.JobsProcessedTotal.WithLabelValues(job.Type, "completed").Inc()

	logger.Info("job execution finished successfully")
	msg.Ack(false)
}

func (c *Consumer) handleFailure(ctx context.Context, job *models.Job, msg amqp.Delivery, errMsg string, durationSec float64) {
	logger := c.logger.With(
		"job_id", job.ID,
		"type", job.Type,
		"duration_sec", durationSec,
		"retry_count", job.RetryCount,
	)
	logger.Warn("job execution failed", "error", errMsg)

	// log the failure
	_ = c.jobLogRepo.Create(ctx, &models.JobLog{
		JobID: job.ID, Level: "ERROR", Message: fmt.Sprintf("Execution failed: %s", errMsg),
	})

	// check retries
	if job.RetryCount < job.MaxRetries {
		if err := c.jobRepo.IncrementRetry(ctx, job.ID); err != nil {
			logger.Error("failed to increment retry", "error", err)
			msg.Nack(false, true) // requeue to try updating state again
			return
		}

		_ = c.jobLogRepo.Create(ctx, &models.JobLog{
			JobID: job.ID, Level: "WARN",
			Message: fmt.Sprintf("Scheduling retry %d of %d", job.RetryCount+1, job.MaxRetries),
		})

		// publish to retry queue
		err := c.amqpConn.Publish(ctx, queue.RoutingKeyRetry, msg.Body)
		if err != nil {
			logger.Error("failed to publish to retry queue", "error", err)
			msg.Nack(false, true) // requeue locally
			return
		}

		logger.Info("job scheduled for retry")
		msg.Ack(false) // we successfully forwarded it to the retry queue
	} else {
		// max retries exceeded
		c.failJob(ctx, job, msg, errMsg)
	}
}

func (c *Consumer) failJob(ctx context.Context, job *models.Job, msg amqp.Delivery, errMsg string) {
	logger := c.logger.With("job_id", job.ID)

	if err := c.jobRepo.MarkFailed(ctx, job.ID, errMsg); err != nil {
		logger.Error("failed to mark job failed", "error", err)
		msg.Nack(false, true)
		return
	}

	_ = c.jobLogRepo.Create(ctx, &models.JobLog{
		JobID: job.ID, Level: "ERROR", Message: "Job failed permanently (max retries exceeded)",
	})

	metrics.JobsProcessedTotal.WithLabelValues(job.Type, "failed").Inc()

	logger.Info("job failed permanently")

	// rejecting without requeue routes the message to the dlq via dlx configuration.
	msg.Reject(false)
}
