package runner

import (
	"testing"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
)

func TestFindSetupPythonStep_Found(t *testing.T) {
	job := workflow.Job{
		Steps: []workflow.Step{
			{Uses: "actions/checkout@v4"},
			{Uses: "actions/setup-python@v5", With: map[string]any{"python-version": "3.12"}},
			{Run: "pip install ruff"},
		},
	}

	step := findSetupPythonStep(job)
	if step == nil {
		t.Fatal("expected to find setup-python step")
	}
}

func TestFindSetupPythonStep_NotFound(t *testing.T) {
	job := workflow.Job{
		Steps: []workflow.Step{
			{Uses: "actions/checkout@v4"},
			{Run: "echo hello"},
		},
	}

	step := findSetupPythonStep(job)
	if step != nil {
		t.Fatal("expected no setup-python step")
	}
}

func TestExtractPythonVersion_FromWith(t *testing.T) {
	step := &workflow.Step{
		Uses: "actions/setup-python@v5",
		With: map[string]any{"python-version": "3.11"},
	}

	version := extractPythonVersion(step)
	if version != "3.11" {
		t.Errorf("expected 3.11, got %s", version)
	}
}

func TestExtractPythonVersion_Default(t *testing.T) {
	step := &workflow.Step{
		Uses: "actions/setup-python@v5",
	}

	version := extractPythonVersion(step)
	if version != "3.12" {
		t.Errorf("expected default 3.12, got %s", version)
	}
}

func TestExtractPythonVersion_MatrixVariable(t *testing.T) {
	step := &workflow.Step{
		Uses: "actions/setup-python@v5",
		With: map[string]any{"python-version": "${{ matrix.python-version }}"},
	}

	version := extractPythonVersion(step)
	// Matrix variables not yet interpolated at this point
	// Should return default
	if version != "3.12" {
		t.Errorf("expected default 3.12 for matrix variable, got %s", version)
	}
}

func TestExtractPythonVersion_EmptyString(t *testing.T) {
	step := &workflow.Step{
		Uses: "actions/setup-python@v5",
		With: map[string]any{"python-version": ""},
	}

	version := extractPythonVersion(step)
	if version != "3.12" {
		t.Errorf("expected default 3.12 for empty string, got %s", version)
	}
}

// Image selection tests
func TestDetermineJobImage_Default(t *testing.T) {
	job := workflow.Job{RunsOn: "ubuntu-latest"}
	runsOn := getRunsOn(job)
	if runsOn != "ubuntu-latest" {
		t.Errorf("expected ubuntu-latest, got %s", runsOn)
	}
}

func TestDetermineJobImage_CustomImage(t *testing.T) {
	job := workflow.Job{RunsOn: "ubuntu-22.04"}
	runsOn := getRunsOn(job)
	if runsOn != "ubuntu-22.04" {
		t.Errorf("expected ubuntu-22.04, got %s", runsOn)
	}
}

func TestDetermineJobImage_WithDefaults(t *testing.T) {
	job := workflow.Job{}
	runsOn := getRunsOn(job)
	if runsOn != "ubuntu-latest" {
		t.Errorf("expected default ubuntu-latest, got %s", runsOn)
	}
}

func TestJobImageSelection_WithSetupPython(t *testing.T) {
	job := workflow.Job{
		RunsOn: "ubuntu-latest",
		Steps: []workflow.Step{
			{Uses: "actions/setup-python@v5", With: map[string]any{"python-version": "3.10"}},
		},
	}
	step := findSetupPythonStep(job)
	if step == nil {
		t.Fatal("expected to find setup-python step")
	}
	ver := extractPythonVersion(step)
	if ver != "3.10" {
		t.Errorf("expected python version 3.10, got %s", ver)
	}
}

func TestJobImageSelection_WithoutSetupPython(t *testing.T) {
	job := workflow.Job{
		RunsOn: "ubuntu-latest",
		Steps: []workflow.Step{
			{Run: "echo no python step"},
		},
	}
	step := findSetupPythonStep(job)
	if step != nil {
		t.Errorf("expected nil setup-python step, got %v", step)
	}
}

