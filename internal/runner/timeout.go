package runner

import (
	"context"
	"time"
)

// WithTimeout returns a context with a timeout and its cancel function.
// This is a minimal helper for Phase 4B timeout support.
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// TimeoutConfig holds timeout configuration for job/step execution.
type TimeoutConfig struct {
	JobTimeout    time.Duration
	StepTimeout   time.Duration
	DefaultJob    time.Duration
	DefaultStep   time.Duration
}

// DefaultTimeoutConfig returns sensible defaults for timeouts.
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		DefaultJob:  6 * time.Hour, // GitHub Actions default
		DefaultStep: 1 * time.Hour,
	}
}