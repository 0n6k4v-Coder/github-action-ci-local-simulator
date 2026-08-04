package runner

import (
	"testing"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
)

func TestValidateNeeds_Valid(t *testing.T) {
	wf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"build": {
				RunsOn: "ubuntu-latest",
				Steps:  []workflow.Step{{Run: "echo build"}},
			},
			"test": {
				RunsOn: "ubuntu-latest",
				Needs:  "build",
				Steps:  []workflow.Step{{Run: "echo test"}},
			},
			"deploy": {
				RunsOn: "ubuntu-latest",
				Needs:  []string{"build", "test"},
				Steps:  []workflow.Step{{Run: "echo deploy"}},
			},
		},
	}

	err := ValidateNeeds(wf)
	if err != nil {
		t.Fatalf("expected valid needs graph to pass validation, got: %v", err)
	}
}

func TestValidateNeeds_MissingDependency(t *testing.T) {
	wf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"test": {
				RunsOn: "ubuntu-latest",
				Needs:  "nonexistent",
				Steps:  []workflow.Step{{Run: "echo test"}},
			},
		},
	}

	err := ValidateNeeds(wf)
	if err == nil {
		t.Fatalf("expected error for missing dependency, got nil")
	}

	verr, ok := err.(*workflow.ValidationErrorWithCode)
	if !ok {
		t.Fatalf("expected ValidationErrorWithCode, got %T: %v", err, err)
	}
	if verr.ExitCode != 2 {
		t.Errorf("expected exit code 2, got %d", verr.ExitCode)
	}
}

func TestValidateNeeds_CycleDetection(t *testing.T) {
	wf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"a": {
				RunsOn: "ubuntu-latest",
				Needs:  "b",
				Steps:  []workflow.Step{{Run: "echo a"}},
			},
			"b": {
				RunsOn: "ubuntu-latest",
				Needs:  "a",
				Steps:  []workflow.Step{{Run: "echo b"}},
			},
		},
	}

	err := ValidateNeeds(wf)
	if err == nil {
		t.Fatalf("expected error for dependency cycle, got nil")
	}

	verr, ok := err.(*workflow.ValidationErrorWithCode)
	if !ok {
		t.Fatalf("expected ValidationErrorWithCode, got %T: %v", err, err)
	}
	if verr.ExitCode != 2 {
		t.Errorf("expected exit code 2, got %d", verr.ExitCode)
	}
}

func TestTopologicalSort(t *testing.T) {
	wf := &workflow.Workflow{
		Jobs: map[string]workflow.Job{
			"deploy": {
				RunsOn: "ubuntu-latest",
				Needs:  []string{"test", "build"},
				Steps:  []workflow.Step{{Run: "echo deploy"}},
			},
			"build": {
				RunsOn: "ubuntu-latest",
				Steps:  []workflow.Step{{Run: "echo build"}},
			},
			"test": {
				RunsOn: "ubuntu-latest",
				Needs:  "build",
				Steps:  []workflow.Step{{Run: "echo test"}},
			},
		},
	}

	order, err := TopologicalSort(wf)
	if err != nil {
		t.Fatalf("unexpected topological sort error: %v", err)
	}

	expected := []string{"build", "test", "deploy"}
	if len(order) != len(expected) {
		t.Fatalf("expected order length %d, got %d", len(expected), len(order))
	}
	for i, id := range expected {
		if order[i] != id {
			t.Errorf("order[%d]: expected %s, got %s", i, id, order[i])
		}
	}
}

func TestAggregateJobResults_Success(t *testing.T) {
	results := []*JobResult{
		{JobID: "build-1", Status: StatusSuccess, ExitCode: 0},
		{JobID: "build-2", Status: StatusSuccess, ExitCode: 0},
	}

	status := AggregateJobResults(results)
	if status != StatusSuccess {
		t.Errorf("expected StatusSuccess, got %s", status)
	}
}

func TestAggregateJobResults_Failure(t *testing.T) {
	results := []*JobResult{
		{JobID: "build-1", Status: StatusSuccess, ExitCode: 0},
		{JobID: "build-2", Status: StatusFailure, ExitCode: 1},
	}

	status := AggregateJobResults(results)
	if status != StatusFailure {
		t.Errorf("expected StatusFailure, got %s", status)
	}
}

func TestAggregateJobResults_Skipped(t *testing.T) {
	results := []*JobResult{
		{JobID: "build-1", Status: StatusSkipped, ExitCode: 0},
		{JobID: "build-2", Status: StatusSkipped, ExitCode: 0},
	}

	status := AggregateJobResults(results)
	if status != StatusSkipped {
		t.Errorf("expected StatusSkipped, got %s", status)
	}
}

func TestNeedsContext_ResultValues(t *testing.T) {
	job := workflow.Job{
		Needs: []string{"build", "test"},
	}
	jobResults := map[string]Status{
		"build": StatusSuccess,
		"test":  StatusFailure,
	}
	jobOutputs := map[string]map[string]string{}
	matrixJobs := map[string]bool{}

	needsCtx := BuildNeedsContext(job, jobResults, jobOutputs, matrixJobs)

	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	exprCtx.SetNeedsContext(needsCtx)

	res1, err := exprCtx.Interpolate("${{ needs.build.result }}")
	if err != nil {
		t.Fatalf("unexpected interpolation error: %v", err)
	}
	if res1 != "success" {
		t.Errorf("expected 'success', got %q", res1)
	}

	res2, err := exprCtx.Interpolate("${{ needs.test.result }}")
	if err != nil {
		t.Fatalf("unexpected interpolation error: %v", err)
	}
	if res2 != "failure" {
		t.Errorf("expected 'failure', got %q", res2)
	}
}

func TestNeedsContext_OutputsNonMatrix(t *testing.T) {
	job := workflow.Job{
		Needs: "build",
	}
	jobResults := map[string]Status{
		"build": StatusSuccess,
	}
	jobOutputs := map[string]map[string]string{
		"build": {
			"version": "1.2.3",
		},
	}
	matrixJobs := map[string]bool{
		"build": false,
	}

	needsCtx := BuildNeedsContext(job, jobResults, jobOutputs, matrixJobs)

	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	exprCtx.SetNeedsContext(needsCtx)

	val, err := exprCtx.Interpolate("${{ needs.build.outputs.version }}")
	if err != nil {
		t.Fatalf("unexpected interpolation error: %v", err)
	}
	if val != "1.2.3" {
		t.Errorf("expected '1.2.3', got %q", val)
	}
}

func TestNeedsContext_OutputsMatrixUnsupported(t *testing.T) {
	job := workflow.Job{
		Needs: "build",
	}
	jobResults := map[string]Status{
		"build": StatusSuccess,
	}
	jobOutputs := map[string]map[string]string{}
	matrixJobs := map[string]bool{
		"build": true,
	}

	needsCtx := BuildNeedsContext(job, jobResults, jobOutputs, matrixJobs)

	exprCtx := NewExpressionContext(nil, nil, nil, NewStepOutputs())
	exprCtx.SetNeedsContext(needsCtx)

	_, err := exprCtx.Interpolate("${{ needs.build.outputs.version }}")
	if err == nil {
		t.Fatalf("expected error referencing matrix job outputs via needs, got nil")
	}
}
