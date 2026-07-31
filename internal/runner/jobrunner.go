package runner

import (
	"context"
	"fmt"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/dockerx"
	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
	"github.com/docker/docker/client"
)

// JobRunner handles execution of jobs within Docker containers.
type JobRunner struct {
	cli *client.Client
}

// NewJobRunner creates a new job runner.
func NewJobRunner(cli *client.Client) *JobRunner {
	return &JobRunner{
		cli: cli,
	}
}

// JobResult represents the result of executing a job.
type JobResult struct {
	JobID    string
	Steps    []*StepResult
	ExitCode int
	Error    error
}

// RunJob executes a job in a Docker container.
func (jr *JobRunner) RunJob(ctx context.Context, job workflow.Job, jobID string, workflowEnv map[string]string) (*JobResult, error) {
	// Resolve image from runs-on
	runsOn := getRunsOn(job)
	imageName, err := dockerx.ResolveImage(runsOn)
	if err != nil {
		return &JobResult{JobID: jobID, ExitCode: 1, Error: err}, nil
	}

	// Ensure image exists
	if err := dockerx.EnsureImage(ctx, jr.cli, imageName); err != nil {
		return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("ensure image: %w", err)}, nil
	}

	// Create container
	workingDir := "/github/workspace"
	containerID, err := dockerx.CreateContainer(ctx, jr.cli, imageName, workingDir)
	if err != nil {
		return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("create container: %w", err)}, nil
	}

	// Start container
	if err := dockerx.StartContainer(ctx, jr.cli, containerID); err != nil {
		_ = dockerx.RemoveContainer(ctx, jr.cli, containerID)
		return &JobResult{JobID: jobID, ExitCode: 1, Error: fmt.Errorf("start container: %w", err)}, nil
	}

	// Create step runner
	stepRunner := NewStepRunner(jr.cli, containerID, workingDir)

	// Prepare job environment
	jobEnv := make(map[string]string)
	for k, v := range workflowEnv {
		jobEnv[k] = fmt.Sprintf("%v", v)
	}
	for k, v := range job.Env {
		jobEnv[k] = fmt.Sprintf("%v", v)
	}

	// Run steps sequentially
	var stepResults []*StepResult
	exitCode := 0
	var firstError error

	for i, step := range job.Steps {
		// Set step environment
		stepEnv := make(map[string]string)
		for k, v := range jobEnv {
			stepEnv[k] = v
		}
		for k, v := range step.Env {
			stepEnv[k] = fmt.Sprintf("%v", v)
		}

		// Run step
		result, err := stepRunner.RunStep(ctx, step, stepEnv)
		if err != nil {
			// Check if continue-on-error
			if step.ContinueOnError {
				stepResults = append(stepResults, &StepResult{
					ExitCode: 1,
					Stdout:   "",
					Stderr:   err.Error(),
				})
				continue
			}
			firstError = err
			exitCode = 1
			break
		}

		stepResults = append(stepResults, result)
		if result.ExitCode != 0 {
			if step.ContinueOnError {
				continue
			}
			exitCode = result.ExitCode
			firstError = fmt.Errorf("step %d failed with exit code %d", i+1, result.ExitCode)
			break
		}
	}

	// Cleanup container
	if err := dockerx.RemoveContainer(ctx, jr.cli, containerID); err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to remove container: %v\n", err)
	}

	return &JobResult{
		JobID:    jobID,
		Steps:    stepResults,
		ExitCode: exitCode,
		Error:    firstError,
	}, nil
}

// getRunsOn extracts the runs-on value from a job.
func getRunsOn(job workflow.Job) string {
	// runs-on can be string or []string - handle both
	switch v := job.RunsOn.(type) {
	case string:
		return v
	case []interface{}:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return s
			}
		}
	case []string:
		if len(v) > 0 {
			return v[0]
		}
	}
	return "ubuntu-latest" // default
}