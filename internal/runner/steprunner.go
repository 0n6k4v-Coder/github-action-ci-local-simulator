package runner

import (
	"context"
	"fmt"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/dockerx"
	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
	"github.com/docker/docker/client"
)

// StepRunner handles execution of individual steps within a job container.
type StepRunner struct {
	cli         *client.Client
	containerID string
	workingDir  string
}

// NewStepRunner creates a new step runner for a job container.
func NewStepRunner(cli *client.Client, containerID, workingDir string) *StepRunner {
	return &StepRunner{
		cli:         cli,
		containerID: containerID,
		workingDir:  workingDir,
	}
}

// RunStep executes a single step and returns the result.
func (sr *StepRunner) RunStep(ctx context.Context, step workflow.Step, jobEnv map[string]string, shell, workingDir string, githubOutputFile *GitHubOutputFile, exprContext *ExpressionContext) (*StepResult, error) {
	// Skip if step has 'uses' (actions not supported yet)
	if step.Uses != "" {
		return nil, fmt.Errorf("actions (uses) not yet supported: %s", step.Uses)
	}

	// Skip if step has neither run nor uses
	if step.Run == "" && step.Uses == "" {
		return nil, fmt.Errorf("step has neither run nor uses")
	}

	// Interpolate expressions in step.Run
	runCommand, err := exprContext.Interpolate(step.Run)
	if err != nil {
		return nil, fmt.Errorf("interpolate step run: %w", err)
	}

	// Build the command to execute
	cmd := sr.buildCommand(runCommand, shell)

	// Set working directory
	if workingDir == "" {
		workingDir = sr.workingDir
	}

	// Execute in container
	result, err := dockerx.ExecCommand(ctx, sr.cli, sr.containerID, workingDir, cmd, jobEnv)
	if err != nil {
		return nil, fmt.Errorf("execute step: %w", err)
	}

	return &StepResult{
		ExitCode: result.ExitCode,
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
	}, nil
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