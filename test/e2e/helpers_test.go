package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createRepo creates a temporary repo with .github/workflows structure
func createRepo(t *testing.T, workflows map[string]string) string {
	tempDir := t.TempDir()

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
