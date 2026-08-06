package runner

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/actions"
	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
)

type jobRunnerInterface interface {
	RunJob(ctx context.Context, job workflow.Job, jobID string, workflowEnv map[string]string, workflowDefaults *workflow.Defaults, wf *workflow.Workflow, workspacePath string, needsCtx map[string]JobNeedsData) (*JobResult, error)
}

// WorkflowRunner handles execution of workflows including job dependency ordering.
type WorkflowRunner struct {
	jobRunner jobRunnerInterface
}

// NewWorkflowRunner creates a new WorkflowRunner.
func NewWorkflowRunner(jobRunner *JobRunner) *WorkflowRunner {
	return &WorkflowRunner{
		jobRunner: jobRunner,
	}
}

// newWorkflowRunnerWithInterface creates a WorkflowRunner with a custom jobRunner interface (used for testing).
func newWorkflowRunnerWithInterface(jobRunner jobRunnerInterface) *WorkflowRunner {
	return &WorkflowRunner{
		jobRunner: jobRunner,
	}
}

// RunWorkflow runs a workflow with parallel job execution for unblocked jobs.
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

	if wf == nil || len(wf.Jobs) == 0 {
		return &WorkflowResult{
			Jobs:     make(map[string]*JobResult),
			ExitCode: 0,
			Status:   StatusSuccess,
		}, nil
	}

	// Setup host cache and artifacts directories
	cacheDir := ".gacils-cache"
	artifactsDir := ".gacils-artifacts"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create host cache dir: %w", err)
	}
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		return nil, fmt.Errorf("create host artifacts dir: %w", err)
	}
	ctx = actions.WithHostDirs(ctx, cacheDir, artifactsDir)

	// 2. Track original job states
	var mu sync.Mutex
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

	// Calculate in-degree and dependents map for dependency tracking
	inDegree := make(map[string]int)
	dependents := make(map[string][]string)

	for id, job := range wf.Jobs {
		needs := getJobNeedsList(job)
		inDegree[id] = len(needs)
		for _, needID := range needs {
			dependents[needID] = append(dependents[needID], id)
		}
	}

	var jobExecutionOrder []string
	workflowFailed := false

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var runErr error
	var errOnce sync.Once
	var wg sync.WaitGroup
	wg.Add(len(wf.Jobs))

	var launchJob func(origJobID string)
	launchJob = func(origJobID string) {
		go func() {
			defer wg.Done()

			if ctx.Err() != nil {
				return
			}

			origJob := wf.Jobs[origJobID]

			// Find expanded job instances belonging to origJobID
			var instanceIDs []string
			for expID := range expandedWf.Jobs {
				if expID == origJobID || isMatrixInstance(expID, origJobID) {
					instanceIDs = append(instanceIDs, expID)
				}
			}
			sort.Strings(instanceIDs)

			// Build needs context for this job under lock
			mu.Lock()
			needsCtx := BuildNeedsContext(origJob, jobResults, jobOutputs, matrixJobs)
			mu.Unlock()

			// Determine if job should run based on dependencies and job-level if
			shouldRun, skipErr := shouldRunJob(origJob, needsCtx)
			if skipErr != nil {
				errOnce.Do(func() {
					runErr = skipErr
					cancel()
				})
				return
			}

			if !shouldRun {
				mu.Lock()
				jobResults[origJobID] = StatusSkipped
				for _, instID := range instanceIDs {
					allJobResults[instID] = &JobResult{
						JobID:    instID,
						ExitCode: 0,
						Status:   StatusSkipped,
					}
					jobExecutionOrder = append(jobExecutionOrder, instID)
				}
				mu.Unlock()
			} else {
				// Run expanded instances concurrently
				instanceResults := make([]*JobResult, len(instanceIDs))
				var instWg sync.WaitGroup
				var instErr error
				var instErrOnce sync.Once

				for i, instID := range instanceIDs {
					instWg.Add(1)
					go func(idx int, id string) {
						defer instWg.Done()

						if ctx.Err() != nil {
							return
						}

						expJob := expandedWf.Jobs[id]
						jobCtx := actions.WithJobID(ctx, id)
						res, err := wr.jobRunner.RunJob(jobCtx, expJob, id, workflowEnv, wf.Defaults, wf, workspacePath, needsCtx)
						if err != nil {
							instErrOnce.Do(func() {
								instErr = fmt.Errorf("run job %s: %w", id, err)
								cancel()
							})
							return
						}

						mu.Lock()
						allJobResults[id] = res
						jobExecutionOrder = append(jobExecutionOrder, id)
						mu.Unlock()

						instanceResults[idx] = res
					}(i, instID)
				}
				instWg.Wait()

				if instErr != nil {
					errOnce.Do(func() {
						runErr = instErr
					})
					return
				}

				// Aggregate result for original job ID
				aggStatus := AggregateJobResults(instanceResults)

				mu.Lock()
				jobResults[origJobID] = aggStatus
				if aggStatus == StatusFailure {
					workflowFailed = true
				}

				// Store resolved outputs if non-matrix job and success
				if !matrixJobs[origJobID] && len(instanceResults) == 1 && instanceResults[0] != nil && instanceResults[0].Status == StatusSuccess {
					if instanceResults[0].Outputs != nil {
						jobOutputs[origJobID] = instanceResults[0].Outputs
					}
				}
				mu.Unlock()
			}

			// Unlock dependent jobs
			var readyDeps []string
			mu.Lock()
			deps := dependents[origJobID]
			for _, depID := range deps {
				inDegree[depID]--
				if inDegree[depID] == 0 {
					readyDeps = append(readyDeps, depID)
				}
			}
			mu.Unlock()

			for _, depID := range readyDeps {
				launchJob(depID)
			}
		}()
	}

	// 3. Find and launch all initial jobs with in-degree 0
	jobIDs := make([]string, 0, len(wf.Jobs))
	for id := range wf.Jobs {
		jobIDs = append(jobIDs, id)
	}
	sort.Strings(jobIDs)

	var initialJobs []string
	for _, id := range jobIDs {
		if inDegree[id] == 0 {
			initialJobs = append(initialJobs, id)
		}
	}

	for _, id := range initialJobs {
		launchJob(id)
	}

	wg.Wait()

	if runErr != nil {
		return nil, runErr
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
	return len(expID) > len(origJobID) && expID[:len(origJobID)] == origJobID && (expID[len(origJobID)] == ' ' || expID[len(origJobID)] == '-')
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
