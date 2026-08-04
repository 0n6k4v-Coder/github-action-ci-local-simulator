package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitHubEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	jobInstanceID := "test-job-123"
	file := NewGitHubEnvFile(jobInstanceID)
	// Override path for testing
	file.path = filepath.Join(tmpDir, "github_env")

	// Test Write
	err := file.Write("KEY1", "value1")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	err = file.Write("KEY2", "value2")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Test ReadAndParse
	vars, err := file.ReadAndParse()
	if err != nil {
		t.Fatalf("ReadAndParse failed: %v", err)
	}

	if vars["KEY1"] != "value1" {
		t.Errorf("Expected KEY1=value1, got %s", vars["KEY1"])
	}
	if vars["KEY2"] != "value2" {
		t.Errorf("Expected KEY2=value2, got %s", vars["KEY2"])
	}

	// Test file is truncated (empty)
	data, err := os.ReadFile(file.path)
	if err != nil {
		t.Fatalf("Read file failed: %v", err)
	}
	if string(data) != "" {
		t.Errorf("Expected empty file after ReadAndParse, got: %s", string(data))
	}
}

func TestGitHubEnvFileMultiline(t *testing.T) {
	tmpDir := t.TempDir()
	file := NewGitHubEnvFile("test")
	file.path = filepath.Join(tmpDir, "github_env")

	// Write multiline value
	err := file.Write("MULTILINE", "line1\nline2\nline3")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	vars, err := file.ReadAndParse()
	if err != nil {
		t.Fatalf("ReadAndParse failed: %v", err)
	}

	expected := "line1\nline2\nline3"
	if vars["MULTILINE"] != expected {
		t.Errorf("Expected multiline value, got: %q", vars["MULTILINE"])
	}
}

func TestGitHubPathFile(t *testing.T) {
	tmpDir := t.TempDir()
	file := NewGitHubPathFile("test")
	file.path = filepath.Join(tmpDir, "github_path")

	// Test Write
	err := file.Write("/path/one")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	err = file.Write("/path/two")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Test ReadAndParse
	paths, err := file.ReadAndParse()
	if err != nil {
		t.Fatalf("ReadAndParse failed: %v", err)
	}

	if len(paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(paths))
	}
	if paths[0] != "/path/one" {
		t.Errorf("Expected /path/one, got %s", paths[0])
	}
	if paths[1] != "/path/two" {
		t.Errorf("Expected /path/two, got %s", paths[1])
	}

	// Test file is truncated
	data, err := os.ReadFile(file.path)
	if err != nil {
		t.Fatalf("Read file failed: %v", err)
	}
	if string(data) != "" {
		t.Errorf("Expected empty file after ReadAndParse, got: %s", string(data))
	}
}

func TestGitHubOutputFile(t *testing.T) {
	tmpDir := t.TempDir()
	file := NewGitHubOutputFile("job-123", "step-1")
	file.path = filepath.Join(tmpDir, "github_output")

	// Test single-line write
	err := file.Write("version", "1.2.3")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Test multiline write
	err = file.Write("body", "line1\nline2")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Test ReadAndParse
	outputs, err := file.ReadAndParse()
	if err != nil {
		t.Fatalf("ReadAndParse failed: %v", err)
	}

	if outputs["version"] != "1.2.3" {
		t.Errorf("Expected version=1.2.3, got %s", outputs["version"])
	}
	if outputs["body"] != "line1\nline2" {
		t.Errorf("Expected multiline body, got %q", outputs["body"])
	}
}

func TestStepOutputs(t *testing.T) {
	so := NewStepOutputs()

	// Test SetOutputs and GetOutputs
	outputs := map[string]string{
		"version": "1.0.0",
		"sha":     "abc123",
	}
	so.SetOutputs("step1", outputs)

	retrieved := so.GetOutputs("step1")
	if retrieved["version"] != "1.0.0" {
		t.Errorf("Expected version=1.0.0, got %s", retrieved["version"])
	}

	// Test GetOutput
	val, ok := so.GetOutput("step1", "version")
	if !ok || val != "1.0.0" {
		t.Errorf("GetOutput failed: %v, %s", ok, val)
	}

	// Test non-existent step
	val, ok = so.GetOutput("nonexistent", "key")
	if ok {
		t.Errorf("Expected false for non-existent step, got true")
	}
}

func TestExpressionContext(t *testing.T) {
	env := map[string]string{
		"MY_VAR": "hello",
	}
	github := map[string]string{
		"WORKFLOW": "test-workflow",
	}
	runner := map[string]string{
		"OS": "Linux",
	}
	stepOutputs := NewStepOutputs()
	stepOutputs.SetOutputs("step1", map[string]string{"output1": "value1"})

	ctx := NewExpressionContext(env, github, runner, stepOutputs)

	// Test env interpolation
	result, err := ctx.Interpolate("${{ env.MY_VAR }}")
	if err != nil {
		t.Fatalf("Interpolate failed: %v", err)
	}
	if result != "hello" {
		t.Errorf("Expected 'hello', got '%s'", result)
	}

	// Test github interpolation
	result, err = ctx.Interpolate("${{ github.WORKFLOW }}")
	if err != nil {
		t.Fatalf("Interpolate failed: %v", err)
	}
	if result != "test-workflow" {
		t.Errorf("Expected 'test-workflow', got '%s'", result)
	}

	// Test runner interpolation
	result, err = ctx.Interpolate("${{ runner.OS }}")
	if err != nil {
		t.Fatalf("Interpolate failed: %v", err)
	}
	if result != "Linux" {
		t.Errorf("Expected 'Linux', got '%s'", result)
	}

	// Test steps interpolation
	result, err = ctx.Interpolate("${{ steps.step1.outputs.output1 }}")
	if err != nil {
		t.Fatalf("Interpolate failed: %v", err)
	}
	if result != "value1" {
		t.Errorf("Expected 'value1', got '%s'", result)
	}

	// Test multiple interpolations in one string
	result, err = ctx.Interpolate("${{ env.MY_VAR }}-${{ github.WORKFLOW }}")
	if err != nil {
		t.Fatalf("Interpolate failed: %v", err)
	}
	if result != "hello-test-workflow" {
		t.Errorf("Expected 'hello-test-workflow', got '%s'", result)
	}

	// Test missing variable returns empty string
	result, err = ctx.Interpolate("${{ env.MISSING }}")
	if err != nil {
		t.Fatalf("Interpolate failed: %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty string for missing var, got '%s'", result)
	}
}

func TestExpressionContextUnsupported(t *testing.T) {
	env := map[string]string{"VAR": "val"}
	github := map[string]string{}
	runner := map[string]string{}
	stepOutputs := NewStepOutputs()

	ctx := NewExpressionContext(env, github, runner, stepOutputs)

	// Test unsupported expressions
	unsupported := []string{
		"${{ env.VAR <= 'val' }}",
		"${{ github.something() }}",
		"${{ runner.os < 'Linux' }}",
	}

	for _, expr := range unsupported {
		_, err := ctx.Interpolate(expr)
		if err == nil {
			t.Errorf("Expected error for unsupported expression: %s", expr)
		}
	}
}

func TestExpressionContextInterpolateMap(t *testing.T) {
	env := map[string]string{"MY_VAR": "hello"}
	github := map[string]string{"WORKFLOW": "test"}
	runner := map[string]string{}
	stepOutputs := NewStepOutputs()

	ctx := NewExpressionContext(env, github, runner, stepOutputs)

	input := map[string]string{
		"KEY1": "${{ env.MY_VAR }}",
		"KEY2": "${{ github.WORKFLOW }}",
		"KEY3": "static",
	}

	result, err := ctx.InterpolateMap(input)
	if err != nil {
		t.Fatalf("InterpolateMap failed: %v", err)
	}

	if result["KEY1"] != "hello" {
		t.Errorf("Expected KEY1=hello, got %s", result["KEY1"])
	}
	if result["KEY2"] != "test" {
		t.Errorf("Expected KEY2=test, got %s", result["KEY2"])
	}
	if result["KEY3"] != "static" {
		t.Errorf("Expected KEY3=static, got %s", result["KEY3"])
	}
}