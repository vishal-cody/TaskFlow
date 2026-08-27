package worker

import (
	"encoding/json"
	"fmt"
	"time"
)

// dataprocessingprocessor simulates processing a batch of data.
// it randomly fails to demonstrate the retry mechanism.
type DataProcessingProcessor struct{}

func (p *DataProcessingProcessor) Process(ctx *JobContext) ([]byte, error) {
	// simulate checking data size
	if err := ctx.UpdateProgress(ctx.Context, 10); err != nil {
	}

	time.Sleep(500 * time.Millisecond)

	// simulate a transient error based on time (e.g., external api rate limit)
	// for demonstration, we'll fail if the current second is even.
	// this ensures some jobs fail and get retried, eventually passing.
	if time.Now().Second()%2 == 0 {
		return nil, fmt.Errorf("external API rate limit exceeded (simulated transient error)")
	}

	if err := ctx.UpdateProgress(ctx.Context, 50); err != nil {
	}

	time.Sleep(1 * time.Second)

	if err := ctx.UpdateProgress(ctx.Context, 90); err != nil {
	}

	time.Sleep(500 * time.Millisecond)

	result := map[string]interface{}{
		"records_processed": 1542,
		"errors":            0,
	}
	return json.Marshal(result)
}
