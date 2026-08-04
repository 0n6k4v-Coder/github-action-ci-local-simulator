package runner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/actions"
	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/dockerx"
	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
	"github.com/docker/docker/client"
)

// StepRunner handles execution of individual steps within a job container.
type StepRunner struct {
	cli         *client.Client
	containerID string
	workingDir  string
	secrets     []string
}

// NewStepRunner creates a new step runner for a job container.
func NewStepRunner(cli *client.Client, containerID, workingDir string) *StepRunner {
	return &StepRunner{
		cli:         cli,
		containerID: containerID,
		workingDir:  workingDir,
	}
}

// SetSecrets sets the secret values to mask in step output.
func (sr *StepRunner) SetSecrets(secrets []string) {
	sr.secrets = secrets
}

// RunStep executes a single step and returns the result.
func (sr *StepRunner) RunStep(ctx context.Context, step workflow.Step, jobEnv map[string]string, shell, workingDir string, githubOutputFile *GitHubOutputFile, exprContext *ExpressionContext, jobTimeout, stepTimeout time.Duration) (*StepResult, error) {
	// Evaluate if condition
	if step.If != "" {
		shouldRun, err := sr.evaluateIfCondition(ctx, step.If, exprContext)
		if err != nil {
			return nil, fmt.Errorf("evaluate if condition: %w", err)
		}
		if !shouldRun {
			return &StepResult{
				ExitCode:        0,
				Stdout:          "",
				Stderr:          "",
				Status:          StatusSkipped,
				Outcome:         StatusSkipped,
				Conclusion:      StatusSkipped,
				ContinueOnError: step.ContinueOnError,
			}, nil
		}
	}

	// Handle action steps (uses)
	if step.Uses != "" {
		ref, err := actions.ParseActionRef(step.Uses)
		if err != nil {
			return nil, NewUnsupportedError(fmt.Sprintf("unsupported action: %s", step.Uses))
		}

		registry := actions.NewRegistry()
		if !registry.IsSupported(ref) {
			return nil, NewUnsupportedError(fmt.Sprintf("unsupported action: %s", ref.ActionName()))
		}

		interpolatedWith, err := exprContext.InterpolateWith(step.With)
		if err != nil {
			return nil, fmt.Errorf("interpolate action inputs: %w", err)
		}

		res, err := registry.Execute(ctx, sr.cli, sr.containerID, workingDir, ref, interpolatedWith)
		if err != nil {
			return nil, err
		}

		if res != nil && res.Env != nil {
			for k, v := range res.Env {
				jobEnv[k] = v
			}
		}

		stdout := ""
		if res != nil {
			stdout = MaskSecrets(res.Stdout, sr.secrets)
		}

		return &StepResult{
			ExitCode:        0,
			Stdout:          stdout,
			Stderr:          "",
			Status:          StatusSuccess,
			Outcome:         StatusSuccess,
			Conclusion:      StatusSuccess,
			ContinueOnError: step.ContinueOnError,
		}, nil
	}

	// Skip if step has neither run nor uses
	if step.Run == "" && step.Uses == "" {
		return nil, fmt.Errorf("step has neither run nor uses")
	}

	// Interpolate expressions in step.Run
	runCommand, err := exprContext.Interpolate(step.Run)
	if err != nil {
		return nil, fmt.Errorf("interpolate step run: %w\n  Hint: Check expression syntax", err)
	}

	// Build the command to execute
	cmd := sr.buildCommand(runCommand, shell)

	// Set working directory
	if workingDir == "" {
		workingDir = sr.workingDir
	}

	// Determine effective timeout for this step
	effectiveTimeout := stepTimeout
	if step.TimeoutMinutes > 0 {
		stepTimeoutDuration, err := ParseTimeoutMinutes(step.TimeoutMinutes)
		if err != nil {
			return nil, fmt.Errorf("parse step timeout-minutes: %w", err)
		}
		effectiveTimeout = stepTimeoutDuration
	} else if jobTimeout > 0 {
		effectiveTimeout = jobTimeout
	}

	// Execute in container with timeout
	var result *dockerx.ExecResult
	if effectiveTimeout > 0 {
		timeoutCtx, cancel := context.WithTimeout(ctx, effectiveTimeout)
		defer cancel()
		result, err = dockerx.ExecCommand(timeoutCtx, sr.cli, sr.containerID, workingDir, cmd, jobEnv)
		if err != nil {
			// Check if it's a timeout error
			if timeoutCtx.Err() == context.DeadlineExceeded {
				return &StepResult{
					ExitCode:        5,
					Stdout:          "",
					Stderr:          MaskSecrets("step timed out", sr.secrets),
					Status:          StatusFailure,
					Outcome:         StatusFailure,
					Conclusion:      StatusFailure,
					ContinueOnError: step.ContinueOnError,
				}, nil
			}
			return nil, fmt.Errorf("execute step: %w\n  Hint: Check Docker container state and network connectivity", err)
		}
	} else {
		result, err = dockerx.ExecCommand(ctx, sr.cli, sr.containerID, workingDir, cmd, jobEnv)
		if err != nil {
			return nil, fmt.Errorf("execute step: %w\n  Hint: Check Docker container state and network connectivity", err)
		}
	}

	// Determine outcome and conclusion
	outcome := StatusSuccess
	if result.ExitCode != 0 {
		outcome = StatusFailure
	}

	conclusion := outcome
	if step.ContinueOnError && outcome == StatusFailure {
		conclusion = StatusSuccess
	}

	return &StepResult{
		ExitCode:        result.ExitCode,
		Stdout:          MaskSecrets(result.Stdout, sr.secrets),
		Stderr:          MaskSecrets(result.Stderr, sr.secrets),
		Status:          conclusion, // Use conclusion as the status
		Outcome:         outcome,
		Conclusion:      conclusion,
		ContinueOnError: step.ContinueOnError,
	}, nil
}

// evaluateIfCondition evaluates the if condition for a step.
func (sr *StepRunner) evaluateIfCondition(ctx context.Context, condition string, exprContext *ExpressionContext) (bool, error) {
	// Interpolate the condition first
	interpolated, err := exprContext.Interpolate(condition)
	if err != nil {
		return false, err
	}

	// Also handle bare expressions (without ${{ }})
	trimmed := strings.TrimSpace(interpolated)
	if !strings.HasPrefix(trimmed, "${{") {
		// Try to evaluate as bare expression
		interpolated = "${{" + trimmed + "}}"
	}

	// Re-interpolate to evaluate functions
	result, err := exprContext.Interpolate(interpolated)
	if err != nil {
		return false, err
	}

	// Parse result as boolean
	switch strings.ToLower(strings.TrimSpace(result)) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	default:
		return false, fmt.Errorf("if condition evaluated to non-boolean: %s", result)
	}
}

// buildCommand builds the command to execute for a run step.
func (sr *StepRunner) buildCommand(runCommand, shell string) []string {
	// Determine shell command format
	var cmd []string
	switch shell {
	case "bash", "":
		cmd = []string{"bash", "-e", "-c", runCommand}
	case "sh":
		cmd = []string{"sh", "-e", "-c", runCommand}
	default:
		// Fallback to bash
		cmd = []string{"bash", "-e", "-c", runCommand}
	}

	return cmd
}