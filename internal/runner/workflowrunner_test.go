package runner

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
)

type mockJobRunner struct {
	mu        sync.Mutex
	jobs      []string
	runErrMap map[string]error
	failJobs  map[string]bool
	skipJobs  map[string]bool
	outputs   map[string]map[string]string
}

func newMockJobRunner() *mockJobRunner {
	return &mockJobRunner{
		runErrMap: make(map[string]error),
		failJobs:  make(map[string]bool),
		skipJobs:  make(map[string]bool),
		outputs:   make(map[string]map[string]string),
	}
}

func (m *mockJobRunner) RunJob(ctx context.Context, job workflow.Job, jobID string, workflowEnv map[string]string, workflowDefaults *workflow.Defaults, wf *workflow.Workflow, workspacePath string, needsCtx map[string]JobNeedsData) (*JobResult, error) {
	m.mu.Lock()
	if err, ok := m.runErrMap[jobID]; ok && err != nil {
		m.mu.Unlock()
		return nil, err
	}
	m.jobs = append(m.jobs, jobID)
	skip := m.skipJobs[jobID]
	fail := m.failJobs[jobID]
	outputs := m.outputs[jobID]
	m.mu.Unlock()

	if skip {
		return &JobResult{JobID: jobID, Status: StatusSkipped, ExitCode: 0}, nil
	}
	if fail {
		return &JobResult{JobID: jobID, Status: StatusFailure, ExitCode: 1}, nil
	}
	return &JobResult{
		JobID:    jobID,
		Status:   StatusSuccess,
		ExitCode: 0,
		Outputs:  outputs,
	}, nil
}

// Basic execution tests
func TestRunWorkflow_EmptyJobs(t *testing.T) {
	mock := newMockJobRunner()
	wr := newWorkflowRunnerWithInterface(mock)

	wf := &workflow.Workflow{Jobs: map[string]workflow.Job{}}
	res, err := wr.RunWorkflow(context.Background(), wf, wf, "/workspace", nil, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSuccess || len(res.Jobs) != 0 {
		t.Errorf("expected empty success result, got %v", res)
	}
}

func TestRunWorkflow_NilWorkflow(t *testing.T) {
	mock := newMockJobRunner()
	wr := newWorkflowRunnerWithInterface(mock)

	res, err := wr.RunWorkflow(context.Background(), nil, nil, "/workspace", nil, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected StatusSuccess for nil workflow, got %s", res.Status)
	}
}

func TestRunWorkflow_SingleJob(t *testing.T) {
	mock := newMockJobRunner()
	wr := newWorkflowRunnerWithInterface(mock)

	wf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"build": {RunsOn: "ubuntu-latest"},
		},
	}
	res, err := wr.RunWorkflow(context.Background(), wf, wf, "/workspace", nil, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected StatusSuccess, got %s", res.Status)
	}
	if len(mock.jobs) != 1 || mock.jobs[0] != "build" {
		t.Errorf("expected executed jobs [build], got %v", mock.jobs)
	}
}

func TestRunWorkflow_MultipleJobs(t *testing.T) {
	mock := newMockJobRunner()
	wr := newWorkflowRunnerWithInterface(mock)

	wf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"job1": {RunsOn: "ubuntu-latest"},
			"job2": {RunsOn: "ubuntu-latest"},
		},
	}
	res, err := wr.RunWorkflow(context.Background(), wf, wf, "/workspace", nil, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected StatusSuccess, got %s", res.Status)
	}
	if len(mock.jobs) != 2 {
		t.Errorf("expected 2 executed jobs, got %v", mock.jobs)
	}
}

// Dependencies tests
func TestRunWorkflow_LinearDependencies(t *testing.T) {
	mock := newMockJobRunner()
	wr := newWorkflowRunnerWithInterface(mock)

	wf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"build":  {RunsOn: "ubuntu-latest"},
			"test":   {RunsOn: "ubuntu-latest", Needs: []any{"build"}},
			"deploy": {RunsOn: "ubuntu-latest", Needs: []any{"test"}},
		},
	}
	res, err := wr.RunWorkflow(context.Background(), wf, wf, "/workspace", nil, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected StatusSuccess, got %s", res.Status)
	}
	if len(mock.jobs) != 3 {
		t.Fatalf("expected 3 jobs, got %v", mock.jobs)
	}
	// Verify topological order
	buildIdx, testIdx, deployIdx := -1, -1, -1
	for i, j := range mock.jobs {
		if j == "build" {
			buildIdx = i
		}
		if j == "test" {
			testIdx = i
		}
		if j == "deploy" {
			deployIdx = i
		}
	}
	if !(buildIdx < testIdx && testIdx < deployIdx) {
		t.Errorf("expected order build -> test -> deploy, got execution order %v", mock.jobs)
	}
}

func TestRunWorkflow_DiamondDependencies(t *testing.T) {
	mock := newMockJobRunner()
	wr := newWorkflowRunnerWithInterface(mock)

	wf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"setup":  {RunsOn: "ubuntu-latest"},
			"test1":  {RunsOn: "ubuntu-latest", Needs: []any{"setup"}},
			"test2":  {RunsOn: "ubuntu-latest", Needs: []any{"setup"}},
			"report": {RunsOn: "ubuntu-latest", Needs: []any{"test1", "test2"}},
		},
	}
	res, err := wr.RunWorkflow(context.Background(), wf, wf, "/workspace", nil, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected StatusSuccess, got %s", res.Status)
	}
	if len(mock.jobs) != 4 {
		t.Fatalf("expected 4 jobs, got %v", mock.jobs)
	}
	if mock.jobs[0] != "setup" || mock.jobs[3] != "report" {
		t.Errorf("expected setup first and report last, got %v", mock.jobs)
	}
}

func TestRunWorkflow_ParallelIndependent(t *testing.T) {
	mock := newMockJobRunner()
	wr := newWorkflowRunnerWithInterface(mock)

	wf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"lint":    {RunsOn: "ubuntu-latest"},
			"unit":    {RunsOn: "ubuntu-latest"},
			"compile": {RunsOn: "ubuntu-latest"},
		},
	}
	res, err := wr.RunWorkflow(context.Background(), wf, wf, "/workspace", nil, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected StatusSuccess, got %s", res.Status)
	}
	if len(mock.jobs) != 3 {
		t.Errorf("expected 3 parallel jobs executed, got %v", mock.jobs)
	}
}

// Matrix tests
func TestRunWorkflow_MatrixExpansion(t *testing.T) {
	mock := newMockJobRunner()
	wr := newWorkflowRunnerWithInterface(mock)

	origWf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"test": {
				RunsOn: "ubuntu-latest",
				Strategy: &workflow.Strategy{
					Matrix: map[string]any{
						"os": []any{"ubuntu", "alpine"},
					},
				},
			},
		},
	}

	expandedWf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"test (os: ubuntu)": {RunsOn: "ubuntu-latest"},
			"test (os: alpine)": {RunsOn: "ubuntu-latest"},
		},
	}

	res, err := wr.RunWorkflow(context.Background(), origWf, expandedWf, "/workspace", nil, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected StatusSuccess, got %s", res.Status)
	}
	if len(mock.jobs) != 2 {
		t.Errorf("expected 2 expanded matrix jobs executed, got %v", mock.jobs)
	}
}

func TestRunWorkflow_MatrixWithInclude(t *testing.T) {
	mock := newMockJobRunner()
	wr := newWorkflowRunnerWithInterface(mock)

	origWf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"test": {
				RunsOn: "ubuntu-latest",
				Strategy: &workflow.Strategy{
					Matrix: map[string]any{
						"version": []any{"3.11"},
						"include": []any{
							map[string]any{"version": "3.12"},
						},
					},
				},
			},
		},
	}

	expandedWf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"test (version: 3.11)": {RunsOn: "ubuntu-latest"},
			"test (version: 3.12)": {RunsOn: "ubuntu-latest"},
		},
	}

	res, err := wr.RunWorkflow(context.Background(), origWf, expandedWf, "/workspace", nil, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected StatusSuccess, got %s", res.Status)
	}
	if len(mock.jobs) != 2 {
		t.Errorf("expected 2 expanded matrix jobs executed, got %v", mock.jobs)
	}
}

func TestRunWorkflow_MatrixWithExclude(t *testing.T) {
	mock := newMockJobRunner()
	wr := newWorkflowRunnerWithInterface(mock)

	origWf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"test": {
				RunsOn: "ubuntu-latest",
				Strategy: &workflow.Strategy{
					Matrix: map[string]any{
						"os": []any{"ubuntu", "windows"},
					},
				},
			},
		},
	}

	expandedWf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"test (os: ubuntu)": {RunsOn: "ubuntu-latest"},
		},
	}

	res, err := wr.RunWorkflow(context.Background(), origWf, expandedWf, "/workspace", nil, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusSuccess {
		t.Errorf("expected StatusSuccess, got %s", res.Status)
	}
	if len(mock.jobs) != 1 || mock.jobs[0] != "test (os: ubuntu)" {
		t.Errorf("expected 1 remaining matrix job, got %v", mock.jobs)
	}
}

// Error handling tests
func TestRunWorkflow_JobFailure(t *testing.T) {
	mock := newMockJobRunner()
	mock.failJobs["test"] = true
	wr := newWorkflowRunnerWithInterface(mock)

	wf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"test": {RunsOn: "ubuntu-latest"},
		},
	}
	res, err := wr.RunWorkflow(context.Background(), wf, wf, "/workspace", nil, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusFailure || res.ExitCode != 1 {
		t.Errorf("expected StatusFailure and exit code 1, got status %s exit %d", res.Status, res.ExitCode)
	}
}

func TestRunWorkflow_JobError(t *testing.T) {
	mock := newMockJobRunner()
	mock.runErrMap["test"] = errors.New("container launch error")
	wr := newWorkflowRunnerWithInterface(mock)

	wf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"test": {RunsOn: "ubuntu-latest"},
		},
	}
	_, err := wr.RunWorkflow(context.Background(), wf, wf, "/workspace", nil, "", 0)
	if err == nil || !strings.Contains(err.Error(), "container launch error") {
		t.Fatalf("expected container launch error, got %v", err)
	}
}

func TestRunWorkflow_ContinueOnError(t *testing.T) {
	mock := newMockJobRunner()
	mock.failJobs["build"] = true
	wr := newWorkflowRunnerWithInterface(mock)

	wf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"build": {RunsOn: "ubuntu-latest"},
			"test":  {RunsOn: "ubuntu-latest", Needs: []any{"build"}, If: "always()"},
		},
	}
	res, err := wr.RunWorkflow(context.Background(), wf, wf, "/workspace", nil, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusFailure {
		t.Errorf("expected workflow status failure overall when a job fails, got %s", res.Status)
	}
	if len(mock.jobs) != 2 {
		t.Errorf("expected both jobs executed due to always() if condition, got %v", mock.jobs)
	}
}

// Conditions tests
func TestRunWorkflow_JobIfCondition(t *testing.T) {
	mock := newMockJobRunner()
	wr := newWorkflowRunnerWithInterface(mock)

	wf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"build": {RunsOn: "ubuntu-latest", If: "false"},
		},
	}
	res, err := wr.RunWorkflow(context.Background(), wf, wf, "/workspace", nil, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Jobs["build"].Status != StatusSkipped {
		t.Errorf("expected job build to be skipped, got status %s", res.Jobs["build"].Status)
	}
	if len(mock.jobs) != 0 {
		t.Errorf("expected 0 executed jobs in mock runner, got %v", mock.jobs)
	}
}

func TestNewWorkflowRunner(t *testing.T) {
	wr := NewWorkflowRunner(nil, false)
	if wr == nil {
		t.Error("NewWorkflowRunner returned nil")
	}
}

func TestRunWorkflow_CycleDependencyError(t *testing.T) {
	mock := newMockJobRunner()
	wr := newWorkflowRunnerWithInterface(mock)

	wf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"jobA": {RunsOn: "ubuntu-latest", Needs: []any{"jobB"}},
			"jobB": {RunsOn: "ubuntu-latest", Needs: []any{"jobA"}},
		},
	}
	_, err := wr.RunWorkflow(context.Background(), wf, wf, "/workspace", nil, "", 0)
	if err == nil {
		t.Fatal("expected dependency cycle error")
	}
}
