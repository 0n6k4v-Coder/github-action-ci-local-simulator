package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	wfPath := filepath.Join(tmpDir, "test.yml")

	wfContent := `name: simple-run

on: push

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo hello
      - run: pwd
      - run: ls -la
`

	if err := os.WriteFile(wfPath, []byte(wfContent), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	wf, err := LoadWorkflow(wfPath)
	if err != nil {
		t.Fatalf("LoadWorkflow failed: %v", err)
	}

	if wf.Name != "simple-run" {
		t.Errorf("expected name 'simple-run', got %q", wf.Name)
	}

	if len(wf.Jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(wf.Jobs))
	}

	job, ok := wf.Jobs["test"]
	if !ok {
		t.Fatal("job 'test' not found")
	}

	if job.Name != "" {
		t.Errorf("expected empty job name, got %q", job.Name)
	}
	// Check runs-on (before normalization, it's a string)
		if job.RunsOn != "ubuntu-latest" {
			t.Errorf("expected runs-on 'ubuntu-latest', got %v", job.RunsOn)
		}

	if len(job.Steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(job.Steps))
	}

	expectedSteps := []string{"echo hello", "pwd", "ls -la"}
	for i, step := range job.Steps {
		if step.Run != expectedSteps[i] {
			t.Errorf("step %d: expected run %q, got %q", i+1, expectedSteps[i], step.Run)
		}
		if step.Uses != "" {
			t.Errorf("step %d: expected empty uses, got %q", i+1, step.Uses)
		}
	}
}

func TestLoadWorkflowInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	wfPath := filepath.Join(tmpDir, "invalid.yml")

	wfContent := `name: test
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo hello
  invalid yaml: [
`

	if err := os.WriteFile(wfPath, []byte(wfContent), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	_, err := LoadWorkflow(wfPath)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestLoadWorkflowMissingFile(t *testing.T) {
	_, err := LoadWorkflow("/nonexistent/path/workflow.yml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestValidateNoJobs(t *testing.T) {
	wf := &Workflow{
		Name: "test",
		Jobs: map[string]Job{},
	}

	err := Validate(wf)
	if err == nil {
		t.Error("expected error for empty jobs, got nil")
	}
}

func TestValidateDuplicateJobID(t *testing.T) {
	// The YAML parser rejects duplicate keys
	// This test verifies that loading fails for duplicate job IDs
	tmpDir := t.TempDir()
	wfPath := filepath.Join(tmpDir, "duplicate.yml")

	wfContent := "name: test\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo 1\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo 2\n"

	if err := os.WriteFile(wfPath, []byte(wfContent), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	_, err := LoadWorkflow(wfPath)
	if err == nil {
		t.Error("expected error for duplicate job ID, got nil")
	}
}

func TestValidateMissingRunsOn(t *testing.T) {
	wf := &Workflow{
		Name: "test",
		Jobs: map[string]Job{
			"test": {Steps: []Step{{Run: "echo hello"}}},
		},
	}

	err := Validate(wf)
	if err == nil {
		t.Error("expected error for missing runs-on, got nil")
	}
}

func TestValidateNoSteps(t *testing.T) {
	wf := &Workflow{
		Name: "test",
		Jobs: map[string]Job{
			"test": {RunsOn: []string{"ubuntu-latest"}},
		},
	}

	err := Validate(wf)
	if err == nil {
		t.Error("expected error for no steps, got nil")
	}
}

func TestValidateStepBothRunAndUses(t *testing.T) {
	wf := &Workflow{
		Name: "test",
		Jobs: map[string]Job{
			"test": {
				RunsOn: []string{"ubuntu-latest"},
				Steps:  []Step{{Run: "echo hello", Uses: "actions/checkout@v4"}},
			},
		},
	}

	err := Validate(wf)
	if err == nil {
		t.Error("expected error for step with both run and uses, got nil")
	}
}

func TestValidateStepNeitherRunNorUses(t *testing.T) {
	wf := &Workflow{
		Name: "test",
		Jobs: map[string]Job{
			"test": {
				RunsOn: []string{"ubuntu-latest"},
				Steps:  []Step{{Name: "invalid step"}},
			},
		},
	}

	err := Validate(wf)
	if err == nil {
		t.Error("expected error for step with neither run nor uses, got nil")
	}
}

func TestGenerateDryRunPlan(t *testing.T) {
	wf := &Workflow{
		Name: "simple-run",
		Jobs: map[string]Job{
			"test": {
				Name:   "Test Job",
				RunsOn: []string{"ubuntu-latest"},
				Steps: []Step{
					{ID: "step1", Name: "Echo Hello", Run: "echo hello"},
					{ID: "step2", Name: "Print PWD", Run: "pwd"},
				},
			},
		},
	}

	plan := GenerateDryRunPlan(wf, "test.yml")

	if plan.WorkflowName != "simple-run" {
		t.Errorf("expected workflow name 'simple-run', got %q", plan.WorkflowName)
	}

	if plan.WorkflowPath != "test.yml" {
		t.Errorf("expected workflow path 'test.yml', got %q", plan.WorkflowPath)
	}

	if len(plan.Jobs) != 1 {
		t.Errorf("expected 1 job in plan, got %d", len(plan.Jobs))
	}

	job := plan.Jobs[0]
	if job.ID != "test" {
		t.Errorf("expected job ID 'test', got %q", job.ID)
	}

	if job.Name != "Test Job" {
		t.Errorf("expected job name 'Test Job', got %q", job.Name)
	}

	if job.RunsOn != "ubuntu-latest" {
		t.Errorf("expected runs-on 'ubuntu-latest', got %q", job.RunsOn)
	}

	if len(job.Steps) != 2 {
		t.Errorf("expected 2 steps in plan, got %d", len(job.Steps))
	}

	if job.Steps[0].Run != "echo hello" {
		t.Errorf("expected step 1 run 'echo hello', got %q", job.Steps[0].Run)
	}

	if job.Steps[1].Run != "pwd" {
		t.Errorf("expected step 2 run 'pwd', got %q", job.Steps[1].Run)
	}
}