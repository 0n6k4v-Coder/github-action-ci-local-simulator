package runner

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
)

type mockJobRunner struct {
	mu           sync.Mutex
	executedJobs []string
	jobTimes     map[string]time.Time
	jobDelay     time.Duration
}

func (m *mockJobRunner) RunJob(
	ctx context.Context,
	job workflow.Job,
	jobID string,
	workflowEnv map[string]string,
	workflowDefaults *workflow.Defaults,
	wf *workflow.Workflow,
	workspacePath string,
	needsCtx map[string]JobNeedsData,
) (*JobResult, error) {
	m.mu.Lock()
	m.executedJobs = append(m.executedJobs, jobID)
	if m.jobTimes == nil {
		m.jobTimes = make(map[string]time.Time)
	}
	m.jobTimes[jobID] = time.Now()
	delay := m.jobDelay
	m.mu.Unlock()

	if delay > 0 {
		time.Sleep(delay)
	}

	return &JobResult{
		JobID:    jobID,
		ExitCode: 0,
		Status:   StatusSuccess,
	}, nil
}

// TestParallelExecution_IndependentJobs verifies two independent jobs run in parallel.
func TestParallelExecution_IndependentJobs(t *testing.T) {
	mock := &mockJobRunner{
		jobDelay: 300 * time.Millisecond,
	}
	wr := newWorkflowRunnerWithInterface(mock)

	wf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"job1": {
				RunsOn: "ubuntu-latest",
				Steps:  []workflow.Step{{Run: "echo job1"}},
			},
			"job2": {
				RunsOn: "ubuntu-latest",
				Steps:  []workflow.Step{{Run: "echo job2"}},
			},
		},
	}

	expandedWf := wf

	start := time.Now()
	res, err := wr.RunWorkflow(context.Background(), wf, expandedWf, "/tmp", nil)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected StatusSuccess, got %v", res.Status)
	}

	// Two jobs running in parallel with 300ms delay each should take < 550ms, whereas sequential would take >= 600ms
	if duration >= 550*time.Millisecond {
		t.Errorf("expected parallel execution (<550ms), but took %v", duration)
	}
}

// TestParallelExecution_MatrixDependency verifies dependent job waits for all matrix instances.
func TestParallelExecution_MatrixDependency(t *testing.T) {
	var mu sync.Mutex
	var eventLog []string

	trackingRunner := &trackingMockJobRunner{
		onJob: func(jobID string) {
			mu.Lock()
			eventLog = append(eventLog, jobID)
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		},
	}

	wr := newWorkflowRunnerWithInterface(trackingRunner)

	wf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"matrix_job": {
				RunsOn: "ubuntu-latest",
				Strategy: &workflow.Strategy{
					Matrix: map[string]interface{}{
						"os": []interface{}{"ubuntu-latest", "windows-latest"},
					},
				},
				Steps: []workflow.Step{{Run: "echo matrix"}},
			},
			"dependent_job": {
				RunsOn: "ubuntu-latest",
				Needs:  "matrix_job",
				Steps:  []workflow.Step{{Run: "echo dep"}},
			},
		},
	}

	expandedWf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"matrix_job (ubuntu-latest)": {
				RunsOn: "ubuntu-latest",
				Steps:  []workflow.Step{{Run: "echo matrix"}},
			},
			"matrix_job (windows-latest)": {
				RunsOn: "windows-latest",
				Steps:  []workflow.Step{{Run: "echo matrix"}},
			},
			"dependent_job": {
				RunsOn: "ubuntu-latest",
				Needs:  "matrix_job",
				Steps:  []workflow.Step{{Run: "echo dep"}},
			},
		},
	}

	res, err := wr.RunWorkflow(context.Background(), wf, expandedWf, "/tmp", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected StatusSuccess, got %v", res.Status)
	}

	mu.Lock()
	defer mu.Unlock()

	// dependent_job must be executed last (after both matrix_job instances)
	if len(eventLog) != 3 {
		t.Fatalf("expected 3 executed jobs, got %d", len(eventLog))
	}
	if eventLog[2] != "dependent_job" {
		t.Errorf("expected dependent_job to run last, execution log: %v", eventLog)
	}
}

// TestParallelExecution_ThreadSafety verifies 10 independent jobs run without data races.
func TestParallelExecution_ThreadSafety(t *testing.T) {
	mock := &mockJobRunner{}
	wr := newWorkflowRunnerWithInterface(mock)

	jobs := make(map[string]workflow.Job)
	for i := 1; i <= 10; i++ {
		jobID := fmt.Sprintf("job%d", i)
		jobs[jobID] = workflow.Job{
			RunsOn: "ubuntu-latest",
			Steps:  []workflow.Step{{Run: fmt.Sprintf("echo %s", jobID)}},
		}
	}

	wf := &workflow.Workflow{Jobs: jobs}
	res, err := wr.RunWorkflow(context.Background(), wf, wf, "/tmp", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected StatusSuccess, got %v", res.Status)
	}
	if len(res.Jobs) != 10 {
		t.Errorf("expected 10 job results, got %d", len(res.Jobs))
	}
}

type trackingMockJobRunner struct {
	onJob func(jobID string)
}

func (t *trackingMockJobRunner) RunJob(
	ctx context.Context,
	job workflow.Job,
	jobID string,
	workflowEnv map[string]string,
	workflowDefaults *workflow.Defaults,
	wf *workflow.Workflow,
	workspacePath string,
	needsCtx map[string]JobNeedsData,
) (*JobResult, error) {
	if t.onJob != nil {
		t.onJob(jobID)
	}
	return &JobResult{
		JobID:    jobID,
		ExitCode: 0,
		Status:   StatusSuccess,
	}, nil
}
