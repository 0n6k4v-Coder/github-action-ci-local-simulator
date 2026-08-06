package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker/client"
)

// isDockerAvailable checks if Docker daemon is accessible
func isDockerAvailable() bool {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return false
	}
	defer cli.Close()

	ctx := context.Background()
	_, err = cli.Ping(ctx)
	return err == nil
}

// skipIfNoDocker skips test if Docker is not available
func skipIfNoDocker(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	if !isDockerAvailable() {
		t.Skip("skipping E2E test: Docker not available")
	}
}

// runGacils runs gacils binary with given arguments
func runGacils(t *testing.T, args ...string) (string, error) {
	cmd := exec.Command("./gacils", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// runGacilsInDir runs gacils binary inside a specific working directory
func runGacilsInDir(t *testing.T, dir string, args ...string) (string, error) {
	absGacils, err := filepath.Abs("./gacils")
	if err != nil {
		t.Fatalf("failed to get absolute path for gacils: %v", err)
	}
	cmd := exec.Command(absGacils, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0755)
}

// buildGacils builds the gacils binary
func buildGacils(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "gacils", "../../cmd/gacils")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build gacils: %v", err)
	}
}

// cleanupGacils removes the gacils binary
func cleanupGacils(t *testing.T) {
	os.Remove("gacils")
}

// createRepo creates a temporary repo with .github/workflows structure and .git directory
func createRepo(t *testing.T, workflows map[string]string) string {
	tempDir := t.TempDir()

	// Create .git directory so FindRepoRoot resolves tempDir as repository root
	os.MkdirAll(filepath.Join(tempDir, ".git"), 0755)

	// Create .github/workflows directory
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	os.MkdirAll(workflowsDir, 0755)

	// Write workflow files
	for name, content := range workflows {
		path := filepath.Join(workflowsDir, name)
		os.WriteFile(path, []byte(content), 0644)
	}

	return tempDir
}

// assertOutputContains checks that output contains expected string
func assertOutputContains(t *testing.T, output, expected string) {
	if !strings.Contains(output, expected) {
		t.Errorf("output should contain %q\nGot: %s", expected, output)
	}
}

// assertOutputNotContains checks that output does NOT contain string
func assertOutputNotContains(t *testing.T, output, notExpected string) {
	if strings.Contains(output, notExpected) {
		t.Errorf("output should NOT contain %q\nGot: %s", notExpected, output)
	}
}

// assertExitCode checks the exit code
func assertExitCode(t *testing.T, err error, expectedCode int) {
	if expectedCode == 0 {
		if err != nil {
			t.Errorf("expected exit code 0, got error: %v", err)
		}
	} else {
		if err == nil {
			t.Errorf("expected exit code %d, got nil error", expectedCode)
		}
	}
}
