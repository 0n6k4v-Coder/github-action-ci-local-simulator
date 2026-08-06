package runner

import (
	"context"
	"testing"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
)

func TestRunWorkflow_JobFilter_SingleJob(t *testing.T) {
	wf := &workflow.Workflow{
		Name: "test",
		Jobs: map[string]workflow.Job{
			"lint": {
				RunsOn: "ubuntu-latest",
				Steps: []workflow.Step{
					{Run: "echo lint"},
				},
			},
			"test": {
				RunsOn: "ubuntu-latest",
				Steps: []workflow.Step{
					{Run: "echo test"},
				},
			},
			"docker": {
				RunsOn: "ubuntu-latest",
				Steps: []workflow.Step{
					{Run: "echo docker"},
				},
			},
		},
	}

	// Use mock job runner to track which jobs are executed
	mockRunner := &mockJobRunnerForFilter{}
	runner := newWorkflowRunnerWithInterface(mockRunner)

	ctx := context.Background()
	result, err := runner.RunWorkflow(ctx, wf, wf, "/workspace", nil, "lint")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify only "lint" job was executed
	if len(mockRunner.executedJobs) != 1 {
		t.Errorf("expected 1 job executed, got %d: %v", len(mockRunner.executedJobs), mockRunner.executedJobs)
	}
	if len(mockRunner.executedJobs) > 0 && mockRunner.executedJobs[0] != "lint" {
		t.Errorf("expected 'lint' job, got %v", mockRunner.executedJobs)
	}
	if result.Status != StatusSuccess {
		t.Errorf("expected success, got %s", result.Status)
	}
}

func TestRunWorkflow_JobFilter_NoFilter(t *testing.T) {
	wf := &workflow.Workflow{
		Name: "test",
		Jobs: map[string]workflow.Job{
			"lint": {
				RunsOn: "ubuntu-latest",
				Steps: []workflow.Step{
					{Run: "echo lint"},
				},
			},
			"test": {
				RunsOn: "ubuntu-latest",
				Steps: []workflow.Step{
					{Run: "echo test"},
				},
			},
		},
	}

	mockRunner := &mockJobRunnerForFilter{}
	runner := newWorkflowRunnerWithInterface(mockRunner)

	ctx := context.Background()
	result, err := runner.RunWorkflow(ctx, wf, wf, "/workspace", nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify ALL jobs were executed (no filter)
	if len(mockRunner.executedJobs) != 2 {
		t.Errorf("expected 2 jobs executed, got %d: %v", len(mockRunner.executedJobs), mockRunner.executedJobs)
	}
	if result.Status != StatusSuccess {
		t.Errorf("expected success, got %s", result.Status)
	}
}

func TestRunWorkflow_JobFilter_NonExistentJob(t *testing.T) {
	wf := &workflow.Workflow{
		Name: "test",
		Jobs: map[string]workflow.Job{
			"lint": {
				RunsOn: "ubuntu-latest",
				Steps: []workflow.Step{
					{Run: "echo lint"},
				},
			},
		},
	}

	mockRunner := &mockJobRunnerForFilter{}
	runner := newWorkflowRunnerWithInterface(mockRunner)

	ctx := context.Background()
	_, err := runner.RunWorkflow(ctx, wf, wf, "/workspace", nil, "nonexistent")

	// Should return error for non-existent job
	if err == nil {
		t.Error("expected error for non-existent job filter, got nil")
	}
}

// mockJobRunnerForFilter tracks which jobs are executed
type mockJobRunnerForFilter struct {
	executedJobs []string
}

func (m *mockJobRunnerForFilter) RunJob(ctx context.Context, job workflow.Job, jobID string, workflowEnv map[string]string, workflowDefaults *workflow.Defaults, wf *workflow.Workflow, workspacePath string, needsCtx map[string]JobNeedsData) (*JobResult, error) {
	m.executedJobs = append(m.executedJobs, jobID)
	return &JobResult{JobID: jobID, Status: StatusSuccess}, nil
}

func TestRunWorkflow_JobFilter_WithDependencies(t *testing.T) {
	wf := &workflow.Workflow{
		Name: "test",
		Jobs: map[string]workflow.Job{
			"build": {
				RunsOn: "ubuntu-latest",
				Steps: []workflow.Step{
					{Run: "echo build"},
				},
			},
			"test": {
				RunsOn: "ubuntu-latest",
				Needs:  []string{"build"},
				Steps: []workflow.Step{
					{Run: "echo test"},
				},
			},
			"deploy": {
				RunsOn: "ubuntu-latest",
				Needs:  []string{"test"},
				Steps: []workflow.Step{
					{Run: "echo deploy"},
				},
			},
		},
	}

	mockRunner := &mockJobRunnerForFilter{}
	runner := newWorkflowRunnerWithInterface(mockRunner)

	ctx := context.Background()
	result, err := runner.RunWorkflow(ctx, wf, wf, "/workspace", nil, "build")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only "build" should run (test and deploy depend on it but are filtered out)
	if len(mockRunner.executedJobs) != 1 {
		t.Errorf("expected 1 job, got %d: %v", len(mockRunner.executedJobs), mockRunner.executedJobs)
	}
	if result.Status != StatusSuccess {
		t.Errorf("expected success, got %s", result.Status)
	}
}

func TestRunWorkflow_JobFilter_MatrixJob(t *testing.T) {
	wf := &workflow.Workflow{
		Name: "test",
		Jobs: map[string]workflow.Job{
			"lint": {
				RunsOn: "ubuntu-latest",
				Steps: []workflow.Step{
					{Run: "echo lint"},
				},
			},
			"test": {
				RunsOn: "ubuntu-latest",
				Strategy: &workflow.Strategy{
					Matrix: map[string]any{
						"python": []any{"3.11", "3.12"},
					},
				},
				Steps: []workflow.Step{
					{Run: "echo test"},
				},
			},
		},
	}

	mockRunner := &mockJobRunnerForFilter{}
	runner := newWorkflowRunnerWithInterface(mockRunner)

	ctx := context.Background()
	result, err := runner.RunWorkflow(ctx, wf, wf, "/workspace", nil, "lint")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only "lint" should run, not matrix instances of "test"
	if len(mockRunner.executedJobs) != 1 {
		t.Errorf("expected 1 job, got %d: %v", len(mockRunner.executedJobs), mockRunner.executedJobs)
	}
	if result.Status != StatusSuccess {
		t.Errorf("expected success, got %s", result.Status)
	}
}
