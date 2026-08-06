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