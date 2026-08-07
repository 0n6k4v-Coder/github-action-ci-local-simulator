package release

import (
	"os/exec"
	"strings"
	"testing"
)

// TestReleaseTag_HasCleanCommand verifies that the tag includes the clean command
// and that it is implemented (not a stub returning nil).
func TestReleaseTag_HasCleanCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	// Get current tag
	cmd := exec.Command("git", "describe", "--tags", "--exact-match")
	tagBytes, err := cmd.Output()
	if err != nil {
		t.Skip("Not on a tagged commit")
	}

	tag := strings.TrimSpace(string(tagBytes))
	t.Logf("Testing tag: %s", tag)

	// Check if clean.go exists in tag
	cmd = exec.Command("git", "show", tag+":internal/cli/clean.go")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clean.go not found in tag %s: %v", tag, err)
	}

	// Check if it's implemented (not a stub)
	outputStr := string(output)
	if strings.Contains(outputStr, "not implemented") {
		t.Errorf("clean command is still a stub in tag %s", tag)
	}

	// Check for required flags — they may be in clean.go or flags.go (bound via BindCleanFlags)
	requiredFlags := []string{"dry-run", "all", "images", "prune-images", "force"}
	for _, flag := range requiredFlags {
		if strings.Contains(outputStr, "\""+flag+"\"") {
			continue
		}
		// Check flags.go for the flag binding
		cmd = exec.Command("git", "show", tag+":internal/cli/flags.go")
		flagsOutput, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("Flag --%s missing in tag %s (not in clean.go or flags.go)", flag, tag)
		} else if !strings.Contains(string(flagsOutput), "\""+flag+"\"") {
			t.Errorf("Flag --%s missing in tag %s", flag, tag)
		}
	}
}

// TestReleaseTag_NoStubs verifies that no stub commands return nil (which
// would give users exit code 0 instead of a proper error).
func TestReleaseTag_NoStubs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	// Get current tag
	cmd := exec.Command("git", "describe", "--tags", "--exact-match")
	tagBytes, err := cmd.Output()
	if err != nil {
		t.Skip("Not on a tagged commit")
	}

	tag := strings.TrimSpace(string(tagBytes))

	// Check commands.go for stubs returning nil
	cmd = exec.Command("git", "show", tag+":internal/cli/commands.go")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("commands.go not found in tag %s: %v", tag, err)
	}

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	for i, line := range lines {
		if strings.Contains(line, "not implemented") {
			// Check next 5 lines for "return nil"
			for j := i + 1; j < len(lines) && j < i+6; j++ {
				if strings.Contains(lines[j], "return nil") {
					t.Errorf("Stub at line %d in tag %s returns nil instead of error", i+1, tag)
					break
				}
			}
		}
	}
}

// TestReleaseTag_READMEDocumentation verifies README mentions clean command
func TestReleaseTag_READMEDocumentation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	// Get current tag
	cmd := exec.Command("git", "describe", "--tags", "--exact-match")
	tagBytes, err := cmd.Output()
	if err != nil {
		t.Skip("Not on a tagged commit")
	}

	tag := strings.TrimSpace(string(tagBytes))

	// Check README mentions clean command
	cmd = exec.Command("git", "show", tag+":README.md")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("README.md not found in tag %s: %v", tag, err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "gacils clean") {
		t.Errorf("README.md in tag %s does not mention 'gacils clean'", tag)
	}
}

// TestReleaseTag_CHANGELOG verifies CHANGELOG has release notes for this version
func TestReleaseTag_CHANGELOG(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	// Get current tag
	cmd := exec.Command("git", "describe", "--tags", "--exact-match")
	tagBytes, err := cmd.Output()
	if err != nil {
		t.Skip("Not on a tagged commit")
	}

	tag := strings.TrimSpace(string(tagBytes))

	// Extract version (remove 'v' prefix)
	version := strings.TrimPrefix(tag, "v")

	// Check CHANGELOG has section for this version
	cmd = exec.Command("git", "show", tag+":CHANGELOG.md")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CHANGELOG.md not found in tag %s: %v", tag, err)
	}

	outputStr := string(output)
	expectedSection := "## [" + version + "]"
	if !strings.Contains(outputStr, expectedSection) {
		t.Errorf("CHANGELOG.md in tag %s missing section '%s'", tag, expectedSection)
	}
}
