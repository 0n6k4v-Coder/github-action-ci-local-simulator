package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildTestBinary builds gacils for testing
func buildTestBinary(t *testing.T) string {
	t.Helper()
	binaryPath := filepath.Join(t.TempDir(), "gacils-test")
	cmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/gacils")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build gacils: %v", err)
	}
	return binaryPath
}

// createTestWorkflow creates a minimal workflow for testing
func createTestWorkflow(t *testing.T, dir string, content string) string {
	t.Helper()
	wfDir := filepath.Join(dir, ".github", "workflows")
	os.MkdirAll(wfDir, 0755)
	wfPath := filepath.Join(wfDir, "test.yml")
	os.WriteFile(wfPath, []byte(content), 0644)
	return wfPath
}

// TestCLI_Version verifies --version flag works
func TestCLI_Version(t *testing.T) {
	binary := buildTestBinary(t)

	cmd := exec.Command(binary, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--version failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "gacils version") {
		t.Errorf("expected version output, got: %s", output)
	}
}

// TestCLI_Help verifies --help flag works
func TestCLI_Help(t *testing.T) {
	binary := buildTestBinary(t)

	cmd := exec.Command(binary, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "run") {
		t.Errorf("expected 'run' command in help, got: %s", output)
	}
}

// TestCLI_RunHelp verifies run --help works
func TestCLI_RunHelp(t *testing.T) {
	binary := buildTestBinary(t)

	cmd := exec.Command(binary, "run", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run --help failed: %v\nOutput: %s", err, output)
	}

	// Verify all expected flags appear in help
	expectedFlags := []string{"--workflow", "--job", "--dry-run", "--parallel", "--crlf", "--platform", "--offline"}
	for _, flag := range expectedFlags {
		if !strings.Contains(string(output), flag) {
			t.Errorf("expected %s in help output, got: %s", flag, output)
		}
	}
}

// TestCLI_InvalidCommand verifies error on invalid command
func TestCLI_InvalidCommand(t *testing.T) {
	binary := buildTestBinary(t)

	cmd := exec.Command(binary, "invalid-command")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("expected error for invalid command, got nil")
	}

	if !strings.Contains(string(output), "unknown command") &&
		!strings.Contains(string(output), "Error") {
		t.Errorf("expected error message, got: %s", output)
	}
}

// TestCLI_MissingWorkflow verifies error when workflow file missing
func TestCLI_MissingWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	binary := buildTestBinary(t)

	cmd := exec.Command(binary, "run", "-W", "/nonexistent/workflow.yml")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("expected error for missing workflow, got nil")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "not found") &&
		!strings.Contains(outputStr, "no such file") &&
		!strings.Contains(outputStr, "Error") &&
		!strings.Contains(outputStr, "does not exist") {
		t.Errorf("expected file not found error, got: %s", outputStr)
	}
}

// TestCLI_JobFlagInHelp verifies --job flag is documented
func TestCLI_JobFlagInHelp(t *testing.T) {
	binary := buildTestBinary(t)

	cmd := exec.Command(binary, "run", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run --help failed: %v", err)
	}

	// Verify --job flag documentation
	if !strings.Contains(string(output), "Run specific job only") {
		t.Errorf("expected job flag description in help")
	}
}

// TestCLI_DryRunFlagInHelp verifies --dry-run flag is documented
func TestCLI_DryRunFlagInHelp(t *testing.T) {
	binary := buildTestBinary(t)

	cmd := exec.Command(binary, "run", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run --help failed: %v", err)
	}

	if !strings.Contains(string(output), "Print execution plan and exit") {
		t.Errorf("expected dry-run flag description in help")
	}
}

// TestCLI_ParallelFlagInHelp verifies --parallel flag is documented
func TestCLI_ParallelFlagInHelp(t *testing.T) {
	binary := buildTestBinary(t)

	cmd := exec.Command(binary, "run", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run --help failed: %v", err)
	}

	if !strings.Contains(string(output), "Number of independent jobs") {
		t.Errorf("expected parallel flag description in help")
	}
}

// TestCLI_ParallelFlag verifies --parallel flag is implemented (no warning)
func TestCLI_ParallelFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	binary := buildTestBinary(t)

	// Create a workflow with multiple independent jobs
	tmpDir := t.TempDir()
	wfPath := createTestWorkflow(t, tmpDir, `
name: test
on: push
jobs:
  job1:
    runs-on: ubuntu-latest
    steps:
      - run: echo "job1"
  job2:
    runs-on: ubuntu-latest
    steps:
      - run: echo "job2"
  job3:
    runs-on: ubuntu-latest
    steps:
      - run: echo "job3"
`)

	// Run with --parallel 1 (sequential execution)
	cmd := exec.Command(binary, "run", "-W", wfPath, "--parallel", "1")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("gacils run failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Should NOT see the warning stub
	if strings.Contains(outputStr, "not yet implemented") {
		t.Error("should not see 'not yet implemented' warning — flag should be implemented")
	}

	// Should complete successfully
	if !strings.Contains(outputStr, "job1") || !strings.Contains(outputStr, "job2") || !strings.Contains(outputStr, "job3") {
		t.Errorf("expected all jobs to run, output: %s", outputStr)
	}
}

// TestCLI_CRLFWarn verifies --crlf emits a "not yet implemented" warning
func TestCLI_CRLFWarn(t *testing.T) {
	binary := buildTestBinary(t)

	cmd := exec.Command(binary, "run", "--crlf", "preserve", "-W", "/nonexistent/workflow.yml")
	output, _ := cmd.CombinedOutput()

	if !strings.Contains(string(output), "Warning") || !strings.Contains(string(output), "--crlf") {
		t.Errorf("expected --crlf warning in output, got: %s", output)
	}
}

// TestCLI_PlatformFlag verifies --platform flag works without warning
func TestCLI_PlatformFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	binary := buildTestBinary(t)

	// Create a simple workflow
	tmpDir := t.TempDir()
	wfPath := createTestWorkflow(t, tmpDir, `
name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "platform test"
`)

	// Run with --platform flag (empty or linux/amd64)
	cmd := exec.Command(binary, "run", "-W", wfPath, "--platform", "linux/amd64")
	output, err := cmd.CombinedOutput()

	// Should NOT see the warning stub
	outputStr := string(output)
	if strings.Contains(outputStr, "not yet implemented") {
		t.Error("should not see 'not yet implemented' warning — flag should be implemented")
	}

	// If Docker is not available, that's fine for this test - we just verify the flag is accepted
	// The key is that there's no "not yet implemented" warning
	if err != nil {
		// Check if error is about Docker not running (acceptable in test env) vs platform-related
		if strings.Contains(outputStr, "Cannot connect to the Docker daemon") {
			t.Log("Docker not available in test environment, skipping error check")
		} else if !strings.Contains(outputStr, "platform") && !strings.Contains(outputStr, "architecture") {
			t.Errorf("unexpected error: %v\nOutput: %s", err, outputStr)
		}
	}
}

// TestCLI_OfflineFlag verifies --offline flag works without warning
func TestCLI_OfflineFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	binary := buildTestBinary(t)

	// Create a minimal workflow
	tmpDir := t.TempDir()
	wfPath := createTestWorkflow(t, tmpDir, `
name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "offline test"
`)

	// Run with --offline flag
	cmd := exec.Command(binary, "run", "-W", wfPath, "--offline")
	output, err := cmd.CombinedOutput()

	// Should either succeed (if image cached) or fail with offline error
	outputStr := string(output)
	if err != nil {
		// If it fails, error should mention offline or not found
		if !strings.Contains(outputStr, "offline") && !strings.Contains(outputStr, "not found") {
			t.Errorf("expected offline-related error, got: %s", outputStr)
		}
	}

	// Should NOT see the warning stub
	if strings.Contains(outputStr, "not yet implemented") {
		t.Error("should not see 'not yet implemented' warning — flag should be implemented")
	}
}
