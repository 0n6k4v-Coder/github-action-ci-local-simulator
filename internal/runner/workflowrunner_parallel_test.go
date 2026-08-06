package runner

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
)

func TestRunWorkflow_Parallel_LimitsConcurrency(t *testing.T) {
	// Create workflow with 5 independent jobs
	wf := &workflow.Workflow{
		Name: "test",
		Jobs: map[string]workflow.Job{
			"job1": {RunsOn: "ubuntu-latest", Steps: []workflow.Step{{Run: "sleep 1"}}},
			"job2": {RunsOn: "ubuntu-latest", Steps: []workflow.Step{{Run: "sleep 1"}}},
			"job3": {RunsOn: "ubuntu-latest", Steps: []workflow.Step{{Run: "sleep 1"}}},
			"job4": {RunsOn: "ubuntu-latest", Steps: []workflow.Step{{Run: "sleep 1"}}},
			"job5": {RunsOn: "ubuntu-latest", Steps: []workflow.Step{{Run: "sleep 1"}}},
		},
	}

	// Track concurrent execution
	var maxConcurrent int32
	var currentConcurrent int32
	var mu sync.Mutex

	mockRunner := &mockParallelRunner{
		onRun: func() {
			// Increment concurrent count
			current := atomic.AddInt32(&currentConcurrent, 1)

			mu.Lock()
			if current > maxConcurrent {
				maxConcurrent = current
			}
			mu.Unlock()

			// Simulate work
			time.Sleep(100 * time.Millisecond)

			// Decrement concurrent count
			atomic.AddInt32(&currentConcurrent, -1)
		},
	}

	runner := newWorkflowRunnerWithInterface(mockRunner)
	ctx := context.Background()

	// Run with parallel=2 (max 2 concurrent jobs)
	result, err := runner.RunWorkflow(ctx, wf, wf, "/workspace", nil, "", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != StatusSuccess {
		t.Errorf("expected success, got %s", result.Status)
	}

	// Verify max concurrent was limited to 2
	if maxConcurrent > 2 {
		t.Errorf("expected max concurrent ≤ 2, got %d", maxConcurrent)
	}

	// Verify all 5 jobs ran
	if len(mockRunner.executedJobs) != 5 {
		t.Errorf("expected 5 jobs, got %d: %v", len(mockRunner.executedJobs), mockRunner.executedJobs)
	}
}

func TestRunWorkflow_Parallel_ZeroMeansUnlimited(t *testing.T) {
	// Create workflow with 3 independent jobs
	wf := &workflow.Workflow{
		Name: "test",
		Jobs: map[string]workflow.Job{
			"job1": {RunsOn: "ubuntu-latest", Steps: []workflow.Step{{Run: "sleep 1"}}},
			"job2": {RunsOn: "ubuntu-latest", Steps: []workflow.Step{{Run: "sleep 1"}}},
			"job3": {RunsOn: "ubuntu-latest", Steps: []workflow.Step{{Run: "sleep 1"}}},
		},
	}

	var maxConcurrent int32
	var currentConcurrent int32
	var mu sync.Mutex

	mockRunner := &mockParallelRunner{
		onRun: func() {
			current := atomic.AddInt32(&currentConcurrent, 1)

			mu.Lock()
			if current > maxConcurrent {
				maxConcurrent = current
			}
			mu.Unlock()

			time.Sleep(100 * time.Millisecond)
			atomic.AddInt32(&currentConcurrent, -1)
		},
	}

	runner := newWorkflowRunnerWithInterface(mockRunner)
	ctx := context.Background()

	// Run with parallel=0 (unlimited)
	result, err := runner.RunWorkflow(ctx, wf, wf, "/workspace", nil, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != StatusSuccess {
		t.Errorf("expected success, got %s", result.Status)
	}

	// With unlimited parallelism, all 3 should run concurrently
	if maxConcurrent < 3 {
		t.Errorf("expected max concurrent = 3 (unlimited), got %d", maxConcurrent)
	}
}

// mockParallelRunner tracks job execution and concurrency
type mockParallelRunner struct {
	executedJobs []string
	onRun        func()
	mu           sync.Mutex
}

func (m *mockParallelRunner) RunJob(ctx context.Context, job workflow.Job, jobID string, workflowEnv map[string]string, workflowDefaults *workflow.Defaults, wf *workflow.Workflow, workspacePath string, needsCtx map[string]JobNeedsData) (*JobResult, error) {
	m.mu.Lock()
	m.executedJobs = append(m.executedJobs, jobID)
	m.mu.Unlock()

	if m.onRun != nil {
		m.onRun()
	}

	return &JobResult{JobID: jobID, Status: StatusSuccess}, nil
}
