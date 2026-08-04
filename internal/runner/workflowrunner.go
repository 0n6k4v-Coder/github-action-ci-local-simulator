package runner

import (
	"context"
	"fmt"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
)

// WorkflowRunner handles execution of workflows including job dependency ordering.
type WorkflowRunner struct {
	jobRunner *JobRunner
}

// NewWorkflowRunner creates a new WorkflowRunner.
func NewWorkflowRunner(jobRunner *JobRunner) *WorkflowRunner {
	return &WorkflowRunner{
		jobRunner: jobRunner,
	}
}

// RunWorkflow runs a workflow in dependency order.
func (wr *WorkflowRunner) RunWorkflow(
	ctx context.Context,
	wf *workflow.Workflow,
	expandedWf *workflow.Workflow,
	workspacePath string,
	workflowEnv map[string]string,
) (*WorkflowResult, error) {
	// 1. Validate needs graph
	if err := ValidateNeeds(wf); err != nil {
		return nil, err
	}

	// 2. Sort job IDs topologically
	sortedJobIDs, err := TopologicalSort(wf)
	if err != nil {
		return nil, err
	}

	// Track original job states
	jobResults := make(map[string]Status)
	jobOutputs := make(map[string]map[string]string)
	matrixJobs := make(map[string]bool)
	allJobResults := make(map[string]*JobResult)

	// Identify matrix jobs in original workflow
	for id, job := range wf.Jobs {
		if job.HasMatrix() {
			matrixJobs[id] = true
		}
	}

	var jobExecutionOrder []string
	workflowFailed := false

	// 3. Execute jobs in topological order
	for _, origJobID := range sortedJobIDs {
		origJob := wf.Jobs[origJobID]

		// Find expanded job instances belonging to origJobID
		var instanceIDs []string
		for expID := range expandedWf.Jobs {
			if expID == origJobID || isMatrixInstance(expID, origJobID) {
				instanceIDs = append(instanceIDs, expID)
			}
		}

		// Build needs context for this job
		needsCtx := BuildNeedsContext(origJob, jobResults, jobOutputs, matrixJobs)

		// Determine if job should run based on dependencies and job-level if
		shouldRun, skipErr := shouldRunJob(origJob, needsCtx)
		if skipErr != nil {
			return nil, skipErr
		}

		if !shouldRun {
			jobResults[origJobID] = StatusSkipped
			for _, instID := range instanceIDs {
				allJobResults[instID] = &JobResult{
					JobID:    instID,
					ExitCode: 0,
					Status:   StatusSkipped,
				}
				jobExecutionOrder = append(jobExecutionOrder, instID)
			}
			continue
		}

		// Run expanded instances sequentially
		var instanceResults []*JobResult

		for _, instID := range instanceIDs {
			jobExecutionOrder = append(jobExecutionOrder, instID)
			expJob := expandedWf.Jobs[instID]
			res, err := wr.jobRunner.RunJob(ctx, expJob, instID, workflowEnv, wf.Defaults, wf, workspacePath, needsCtx)
			if err != nil {
				return nil, fmt.Errorf("run job %s: %w", instID, err)
			}

			allJobResults[instID] = res
			instanceResults = append(instanceResults, res)
		}

		// Aggregate result for original job ID
		aggStatus := AggregateJobResults(instanceResults)
		jobResults[origJobID] = aggStatus

		if aggStatus == StatusFailure {
			workflowFailed = true
		}

		// Store resolved outputs if non-matrix job
		if !matrixJobs[origJobID] && len(instanceResults) == 1 && instanceResults[0].Status == StatusSuccess {
			if instanceResults[0].Outputs != nil {
				jobOutputs[origJobID] = instanceResults[0].Outputs
			}
		}
	}

	finalStatus := StatusSuccess
	finalExitCode := 0
	if workflowFailed {
		finalStatus = StatusFailure
		finalExitCode = 1
	}

	return &WorkflowResult{
		Jobs:     allJobResults,
		JobOrder: jobExecutionOrder,
		ExitCode: finalExitCode,
		Status:   finalStatus,
	}, nil
}

// isMatrixInstance checks if expanded ID belongs to original job ID.
func isMatrixInstance(expID, origJobID string) bool {
	return len(expID) > len(origJobID) && expID[:len(origJobID)] == origJobID && expID[len(origJobID)] == ' '
}

// shouldRunJob checks if job should run based on dependencies and job-level if condition.
func shouldRunJob(job workflow.Job, needsCtx map[string]JobNeedsData) (bool, error) {
	neededJobIDs := getJobNeedsList(job)

	// If no dependencies and no job-level if, run
	if len(neededJobIDs) == 0 && job.If == "" {
		return true, nil
	}

	// Create temporary ExpressionContext to evaluate job-level if
	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	exprCtx.SetNeedsContext(needsCtx)

	condition := job.If
	if condition == "" {
		// Default behavior: success()
		condition = "success()"
	}

	return evaluateIfConditionStatic(condition, exprCtx)
}
