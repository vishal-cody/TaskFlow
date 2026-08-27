package models

import (
	"testing"
)

func TestValidateTransition_AllowedTransitions(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{"queued to processing", JobStatusQueued, JobStatusProcessing},
		{"queued to cancelled", JobStatusQueued, JobStatusCancelled},
		{"processing to completed", JobStatusProcessing, JobStatusCompleted},
		{"processing to failed", JobStatusProcessing, JobStatusFailed},
		{"failed to retrying", JobStatusFailed, JobStatusRetrying},
		{"retrying to processing", JobStatusRetrying, JobStatusProcessing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateTransition(tt.from, tt.to); err != nil {
				t.Errorf("expected transition %s → %s to be allowed, got error: %v",
					tt.from, tt.to, err)
			}
		})
	}
}

func TestValidateTransition_RejectedTransitions(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{"completed is terminal", JobStatusCompleted, JobStatusProcessing},
		{"cancelled is terminal", JobStatusCancelled, JobStatusQueued},
		{"queued to failed", JobStatusQueued, JobStatusFailed},
		{"queued to completed", JobStatusQueued, JobStatusCompleted},
		{"processing to queued", JobStatusProcessing, JobStatusQueued},
		{"processing to cancelled", JobStatusProcessing, JobStatusCancelled},
		{"failed to completed", JobStatusFailed, JobStatusCompleted},
		{"failed to queued", JobStatusFailed, JobStatusQueued},
		{"retrying to failed", JobStatusRetrying, JobStatusFailed},
		{"retrying to completed", JobStatusRetrying, JobStatusCompleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateTransition(tt.from, tt.to); err == nil {
				t.Errorf("expected transition %s → %s to be rejected, but it was allowed",
					tt.from, tt.to)
			}
		})
	}
}

func TestIsValidJobType(t *testing.T) {
	valid := []string{
		JobTypeReportGeneration,
		JobTypeDataProcessing,
		JobTypeImageProcessing,
		JobTypeNotification,
	}
	for _, jt := range valid {
		if !IsValidJobType(jt) {
			t.Errorf("expected %q to be valid job type", jt)
		}
	}

	invalid := []string{"unknown", "", "pdf_generation", "REPORT_GENERATION"}
	for _, jt := range invalid {
		if IsValidJobType(jt) {
			t.Errorf("expected %q to be invalid job type", jt)
		}
	}
}

func TestValidatePriority(t *testing.T) {
	// valid range
	for i := 1; i <= 10; i++ {
		if err := ValidatePriority(i); err != nil {
			t.Errorf("expected priority %d to be valid, got error: %v", i, err)
		}
	}

	// invalid
	invalid := []int{0, -1, 11, 100}
	for _, p := range invalid {
		if err := ValidatePriority(p); err == nil {
			t.Errorf("expected priority %d to be invalid", p)
		}
	}
}

func TestIsCancellable(t *testing.T) {
	if !IsCancellable(JobStatusQueued) {
		t.Error("queued jobs should be cancellable")
	}

	nonCancellable := []string{
		JobStatusProcessing, JobStatusCompleted,
		JobStatusFailed, JobStatusRetrying, JobStatusCancelled,
	}
	for _, s := range nonCancellable {
		if IsCancellable(s) {
			t.Errorf("jobs in status %q should not be cancellable", s)
		}
	}
}

func TestIsTerminalStatus(t *testing.T) {
	if !IsTerminalStatus(JobStatusCompleted) {
		t.Error("completed should be terminal")
	}
	if !IsTerminalStatus(JobStatusCancelled) {
		t.Error("cancelled should be terminal")
	}

	nonTerminal := []string{
		JobStatusQueued, JobStatusProcessing,
		JobStatusFailed, JobStatusRetrying,
	}
	for _, s := range nonTerminal {
		if IsTerminalStatus(s) {
			t.Errorf("status %q should not be terminal", s)
		}
	}
}
