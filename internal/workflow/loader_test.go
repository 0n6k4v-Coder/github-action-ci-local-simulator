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

func TestLoadWorkflowsFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	wfPath := filepath.Join(tmpDir, "test.yml")

	wfContent := `name: simple-run

on: push

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo hello
`

	if err := os.WriteFile(wfPath, []byte(wfContent), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	workflows, err := LoadWorkflows(wfPath)
	if err != nil {
		t.Fatalf("LoadWorkflows failed: %v", err)
	}

	if len(workflows) != 1 {
		t.Errorf("expected 1 workflow, got %d", len(workflows))
	}

	if workflows[0].Name != "simple-run" {
		t.Errorf("expected name 'simple-run', got %q", workflows[0].Name)
	}
}

func TestLoadWorkflowsFromDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first workflow
	wf1Content := `name: workflow-one
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo one
`
	wf1Path := filepath.Join(tmpDir, "one.yml")
	if err := os.WriteFile(wf1Path, []byte(wf1Content), 0644); err != nil {
		t.Fatalf("write wf1: %v", err)
	}

	// Create second workflow
	wf2Content := `name: workflow-two
on: pull_request
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo two
`
	wf2Path := filepath.Join(tmpDir, "two.yaml")
	if err := os.WriteFile(wf2Path, []byte(wf2Content), 0644); err != nil {
		t.Fatalf("write wf2: %v", err)
	}

	// Create a non-workflow file (should be ignored)
	txtPath := filepath.Join(tmpDir, "readme.txt")
	if err := os.WriteFile(txtPath, []byte("ignore me"), 0644); err != nil {
		t.Fatalf("write txt: %v", err)
	}

	workflows, err := LoadWorkflows(tmpDir)
	if err != nil {
		t.Fatalf("LoadWorkflows failed: %v", err)
	}

	if len(workflows) != 2 {
		t.Errorf("expected 2 workflows, got %d", len(workflows))
	}

	// Check names (order is filesystem-dependent, so check both)
	names := map[string]bool{}
	for _, wf := range workflows {
		names[wf.Name] = true
	}
	if !names["workflow-one"] {
		t.Error("expected workflow-one in results")
	}
	if !names["workflow-two"] {
		t.Error("expected workflow-two in results")
	}
}

func TestLoadWorkflowsEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := LoadWorkflows(tmpDir)
	if err == nil {
		t.Error("expected error for empty directory, got nil")
	}
}

func TestLoadWorkflowsNonexistent(t *testing.T) {
	_, err := LoadWorkflows("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent path, got nil")
	}
}

func TestDefaultWorkflowDir(t *testing.T) {
	dir := DefaultWorkflowDir()
	if dir != ".github/workflows" {
		t.Errorf("expected '.github/workflows', got %q", dir)
	}
}

func TestGenerateDryRunPlanSet(t *testing.T) {
	wf1 := &Workflow{
		Name: "workflow-one",
		Jobs: map[string]Job{
			"test": {
				Name:   "Test Job",
				RunsOn: []string{"ubuntu-latest"},
				Steps:  []Step{{ID: "step1", Run: "echo one"}},
			},
		},
	}

	wf2 := &Workflow{
		Name: "workflow-two",
		Jobs: map[string]Job{
			"build": {
				Name:   "Build Job",
				RunsOn: []string{"ubuntu-latest"},
				Steps:  []Step{{ID: "step1", Run: "echo two"}},
			},
		},
	}

	planSet := GenerateDryRunPlanSet([]*Workflow{wf1, wf2}, []string{"one.yml", "two.yml"})

	if len(planSet.Plans) != 2 {
		t.Errorf("expected 2 plans, got %d", len(planSet.Plans))
	}

	if planSet.Plans[0].WorkflowName != "workflow-one" {
		t.Errorf("expected first plan name 'workflow-one', got %q", planSet.Plans[0].WorkflowName)
	}

	if planSet.Plans[1].WorkflowName != "workflow-two" {
		t.Errorf("expected second plan name 'workflow-two', got %q", planSet.Plans[1].WorkflowName)
	}

	if planSet.Plans[0].WorkflowPath != "one.yml" {
		t.Errorf("expected first path 'one.yml', got %q", planSet.Plans[0].WorkflowPath)
	}

	if planSet.Plans[1].WorkflowPath != "two.yml" {
		t.Errorf("expected second path 'two.yml', got %q", planSet.Plans[1].WorkflowPath)
	}
}
