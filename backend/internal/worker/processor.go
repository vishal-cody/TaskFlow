package worker

import (
	"context"
	"fmt"

	"github.com/vishalyadav/jobplatform/internal/models"
)

// jobcontext provides the execution context and utilities for a running job.
type JobContext struct {
	context.Context
	Job *models.Job

	// updateprogress allows the processor to report progress (0-100).
	UpdateProgress func(ctx context.Context, progress int) error
}

// jobprocessor defines the interface that all specific job type handlers must implement.
type JobProcessor interface {
	// process executes the job. it should be idempotent.
	// if it returns an error, the job will be retried (or sent to dlq if max retries exceeded).
	// if it returns a non-nil byte slice, that is stored as the job result.
	Process(ctx *JobContext) ([]byte, error)
}

// processorregistry holds a mapping of job types to their respective processors.
type ProcessorRegistry struct {
	processors map[string]JobProcessor
}

func NewProcessorRegistry() *ProcessorRegistry {
	return &ProcessorRegistry{
		processors: make(map[string]JobProcessor),
	}
}

// register adds a processor for a specific job type.
func (r *ProcessorRegistry) Register(jobType string, p JobProcessor) {
	r.processors[jobType] = p
}

// get returns the processor for a job type, or an error if not found.
func (r *ProcessorRegistry) Get(jobType string) (JobProcessor, error) {
	p, ok := r.processors[jobType]
	if !ok {
		return nil, fmt.Errorf("no processor registered for job type: %s", jobType)
	}
	return p, nil
}
