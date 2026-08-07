package runner

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// WithTimeout returns a context with a timeout and its cancel function.
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// ParseTimeoutMinutes parses timeout-minutes value which can be integer or float.
func ParseTimeoutMinutes(value any) (time.Duration, error) {
	if value == nil {
		return 0, nil
	}

	switch v := value.(type) {
	case int:
		return time.Duration(v) * time.Minute, nil
	case float64:
		return time.Duration(v * float64(time.Minute)), nil
	case string:
		// Try parsing as float
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(f * float64(time.Minute)), nil
	default:
		return 0, fmt.Errorf("timeout-minutes must be a number, got %T", value)
	}
}

// TimeoutConfig holds timeout configuration for job/step execution.
type TimeoutConfig struct {
	JobTimeout  time.Duration
	StepTimeout time.Duration
	DefaultJob  time.Duration
	DefaultStep time.Duration
}

// DefaultTimeoutConfig returns sensible defaults for timeouts.
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		DefaultJob:  6 * time.Hour, // GitHub Actions default (360 minutes)
		DefaultStep: 1 * time.Hour,
	}
}

// GetEffectiveStepTimeout returns the effective timeout for a step.
// Step timeout overrides job timeout. If neither is set, uses default.
func (tc *TimeoutConfig) GetEffectiveStepTimeout(jobTimeout, stepTimeout time.Duration) time.Duration {
	if stepTimeout > 0 {
		return stepTimeout
	}
	if jobTimeout > 0 {
		return jobTimeout
	}
	return tc.DefaultStep
}

// GetEffectiveJobTimeout returns the effective timeout for a job.
// Uses default if not set.
func (tc *TimeoutConfig) GetEffectiveJobTimeout(jobTimeout time.Duration) time.Duration {
	if jobTimeout > 0 {
		return jobTimeout
	}
	return tc.DefaultJob
}
