package runner

import (
	"context"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
)

// Runner defines the interface for running workflows.
type Runner interface {
	// RunWorkflow executes a workflow and returns the result.
	RunWorkflow(ctx context.Context, wf *workflow.Workflow) (*WorkflowResult, error)

	// RunJob executes a single job and returns the result.
	RunJob(ctx context.Context, job workflow.Job, jobID string, workflowEnv map[string]string, workflowDefaults *workflow.Defaults, wf *workflow.Workflow) (*JobResult, error)
}

// RunConfig holds configuration for a workflow run.
type RunConfig struct {
	WorkflowPath  string
	Job           string
	DryRun        bool
	Parallel      int
	CRLF          string
	Platform      string
	Offline       bool
	TimeoutConfig TimeoutConfig
}