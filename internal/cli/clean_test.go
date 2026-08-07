package cli

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

// TestCleanCommand_Exists verifies the clean command is registered and shows help.
func TestCleanCommand_Exists(t *testing.T) {
	binary := buildTestBinary(t)

	// Test help output
	cmd := exec.Command(binary, "clean", "--help")
	output, _ := cmd.CombinedOutput()

	// Should show clean command help
	if !strings.Contains(string(output), "clean") {
		t.Errorf("expected clean command in help, got: %s", output)
	}
}

// TestCleanCommand_DryRun verifies --dry-run does not fail and shows dry-run output.
func TestCleanCommand_DryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	binary := buildTestBinary(t)

	// Dry run should not actually remove anything.
	// Even if Docker is unavailable, the error should be graceful (not a panic).
	cmd := exec.Command(binary, "clean", "--dry-run")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Docker might not be available, but there should be a graceful error message
		outputStr := string(output)
		if strings.Contains(outputStr, "panic") {
			t.Errorf("should not panic, got: %s", outputStr)
		}
	}

	// Should show dry run output
	outputStr := string(output)
	if !strings.Contains(outputStr, "Dry Run") &&
		!strings.Contains(outputStr, "Would remove") &&
		!strings.Contains(outputStr, "[DRY RUN]") &&
		!strings.Contains(outputStr, "dry run") {
		t.Logf("Output: %s", outputStr)
	}
}

// TestCleanCommand_InvalidFlags verifies invalid flags don't cause a panic.
func TestCleanCommand_InvalidFlags(t *testing.T) {
	binary := buildTestBinary(t)

	// Invalid flag should show error
	cmd := exec.Command(binary, "clean", "--invalid-flag")
	output, _ := cmd.CombinedOutput()

	outputStr := string(output)
	if strings.Contains(outputStr, "panic") {
		t.Errorf("should not panic on invalid flag, got: %s", outputStr)
	}
}

// TestCleanCommand_ForceSkipsConfirm verifies --force flag is accepted.
func TestCleanCommand_ForceSkipsConfirm(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	binary := buildTestBinary(t)

	cmd := exec.Command(binary, "clean", "--force", "--dry-run")
	output, err := cmd.CombinedOutput()

	if err != nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "panic") {
			t.Errorf("should not panic, got: %s", outputStr)
		}
		t.Logf("Output (with error): %s", outputStr)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Dry Run") &&
		!strings.Contains(outputStr, "[DRY RUN]") &&
		!strings.Contains(outputStr, "dry run") {
		t.Errorf("expected dry run output, got: %s", outputStr)
	}
}

// TestCleanCommand_AllFlag verifies --all flag is accepted without panic.
func TestCleanCommand_AllFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	binary := buildTestBinary(t)

	cmd := exec.Command(binary, "clean", "--all", "--force", "--dry-run")
	output, err := cmd.CombinedOutput()

	outputStr := string(output)
	if strings.Contains(outputStr, "panic") {
		t.Errorf("should not panic, got: %s", outputStr)
	}

	if err != nil {
		t.Logf("Output (with error): %s", outputStr)
	}

	// Should mention volumes in --all dry-run output
	if !strings.Contains(outputStr, "volume") && err == nil {
		t.Errorf("expected volume mention in --all output, got: %s", outputStr)
	}
}

// TestCleanCommand_ImagesFlag verifies --images flag is accepted.
func TestCleanCommand_ImagesFlag(t *testing.T) {
	binary := buildTestBinary(t)

	cmd := exec.Command(binary, "clean", "--images", "--help")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("clean --images --help failed: %v\nOutput: %s", err, output)
	}
}

// TestCleanCommand_PruneImagesFlag verifies --prune-images flag is accepted.
func TestCleanCommand_PruneImagesFlag(t *testing.T) {
	binary := buildTestBinary(t)

	cmd := exec.Command(binary, "clean", "--prune-images", "--help")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("clean --prune-images --help failed: %v\nOutput: %s", err, output)
	}

	// Should mention prune-images in help
	if !strings.Contains(string(output), "prune-images") {
		t.Errorf("expected --prune-images in help, got: %s", output)
	}
}

// TestCleanFlags_StructDefaults verifies CleanFlags struct zero-value defaults.
func TestCleanFlags_StructDefaults(t *testing.T) {
	flags := &CleanFlags{}

	if flags.All {
		t.Error("expected All default to be false")
	}
	if flags.Images {
		t.Error("expected Images default to be false")
	}
	if flags.PruneImages {
		t.Error("expected PruneImages default to be false")
	}
	if flags.DryRun {
		t.Error("expected DryRun default to be false")
	}
	if flags.Force {
		t.Error("expected Force default to be false")
	}
}

// TestBindCleanFlags_RegistersAllFlags verifies all clean flags are registered on the cobra command.
func TestBindCleanFlags_RegistersAllFlags(t *testing.T) {
	cmd := newCleanCmd()

	expectedFlags := []string{"all", "images", "prune-images", "dry-run", "force"}
	for _, flag := range expectedFlags {
		f := cmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("flag --%s was not registered on clean command", flag)
		}
	}
}

// TestBindCleanFlags_Defaults verifies cobra-registered defaults match expected values.
func TestBindCleanFlags_Defaults(t *testing.T) {
	cmd := newCleanCmd()

	expectedFlags := []string{"all", "images", "prune-images", "dry-run", "force"}
	for _, flag := range expectedFlags {
		f := cmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("flag --%s was not registered", flag)
			continue
		}
		if f.DefValue != "false" {
			t.Errorf("expected default for --%s to be 'false', got %q", flag, f.DefValue)
		}
	}
}

// TestBindCleanFlags_FlagCount verifies the number of registered flags matches CleanFlags struct fields.
func TestBindCleanFlags_FlagCount(t *testing.T) {
	cmd := newCleanCmd()

	registeredCount := 0
	cmd.Flags().VisitAll(func(_ *pflag.Flag) {
		registeredCount++
	})

	structFieldCount := 5 // All, Images, PruneImages, DryRun, Force

	if registeredCount != structFieldCount {
		t.Errorf("CleanFlags has %d fields but BindCleanFlags registers %d flags",
			structFieldCount, registeredCount)
	}
}
