package worker

import (
	"encoding/json"
	"fmt"
	"time"
)

// reportgenerationprocessor simulates generating a report.
type ReportGenerationProcessor struct{}

type reportPayload struct {
	ReportType string `json:"report_type"`
	DateRange  string `json:"date_range"`
}

func (p *ReportGenerationProcessor) Process(ctx *JobContext) ([]byte, error) {
	// parse payload
	var payload reportPayload
	if len(ctx.Job.Payload) > 0 {
		if err := json.Unmarshal(ctx.Job.Payload, &payload); err != nil {
			return nil, fmt.Errorf("invalid payload: %w", err)
		}
	}

	// simulate work and report progress
	steps := 5
	for i := 1; i <= steps; i++ {
		// check for cancellation or timeout
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("job cancelled: %w", err)
		}

		time.Sleep(1 * time.Second) // simulate expensive work

		progress := (i * 100) / steps
		if err := ctx.UpdateProgress(ctx.Context, progress); err != nil {
			// non-fatal, just log it and continue
			// in a real app we'd inject a logger here, but for now we ignore
		}
	}

	// produce result
	result := map[string]string{
		"report_url": fmt.Sprintf("https://storage.example.com/reports/%s.pdf", ctx.Job.ID),
		"status":     "generated",
	}
	return json.Marshal(result)
}
