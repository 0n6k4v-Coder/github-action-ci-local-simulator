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
