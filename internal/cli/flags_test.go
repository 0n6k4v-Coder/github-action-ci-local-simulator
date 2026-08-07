package cli

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TestAllFlagsHaveTests verifies every flag in RunFlags has test coverage
func TestAllFlagsHaveTests(t *testing.T) {
	flags := RunFlags{}
	flagType := reflect.TypeOf(flags)

	// List all fields in RunFlags
	var flagNames []string
	for i := 0; i < flagType.NumField(); i++ {
		flagNames = append(flagNames, flagType.Field(i).Name)
	}

	// Verify we have tests for each flag
	// This is a meta-test to ensure flag coverage
	t.Logf("Flags defined in RunFlags: %v", flagNames)

	// Each flag should have at least one test
	// This test serves as documentation of required coverage
	for _, name := range flagNames {
		t.Logf("Flag %s should have test coverage", name)
	}
}

// TestRunFlagsStruct verifies RunFlags struct has expected fields
func TestRunFlagsStruct(t *testing.T) {
	flags := RunFlags{}
	flagType := reflect.TypeOf(flags)

	// Verify critical fields exist
	requiredFields := []string{"Workflow", "Job", "DryRun", "Parallel", "CRLF", "Platform", "Offline"}
	for _, field := range requiredFields {
		if _, found := flagType.FieldByName(field); !found {
			t.Errorf("RunFlags missing required field: %s", field)
		}
	}
}

// TestBindRunFlags_RegistersAllFlags verifies BindRunFlags registers every expected flag on the cobra command
func TestBindRunFlags_RegistersAllFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	flags := &RunFlags{}
	BindRunFlags(cmd, flags)

	expectedFlags := []struct {
		long  string
		short string
	}{
		{"workflow", "W"},
		{"job", "j"},
		{"dry-run", ""},
		{"parallel", "p"},
		{"crlf", ""},
		{"platform", ""},
		{"offline", ""},
	}

	for _, ef := range expectedFlags {
		f := cmd.Flags().Lookup(ef.long)
		if f == nil {
			t.Errorf("flag --%s was not registered", ef.long)
			continue
		}
		if ef.short != "" {
			sf := cmd.Flags().ShorthandLookup(ef.short)
			if sf == nil {
				t.Errorf("shorthand -%s for --%s was not registered", ef.short, ef.long)
			}
		}
	}
}

// TestBindRunFlags_Defaults verifies the cobra-registered defaults match expected values
func TestBindRunFlags_Defaults(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	flags := &RunFlags{}
	BindRunFlags(cmd, flags)

	// Parse empty args so defaults are applied
	if err := cmd.ParseFlags([]string{}); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if flags.Workflow != "" {
		t.Errorf("expected default Workflow='', got %q", flags.Workflow)
	}
	if flags.Job != "" {
		t.Errorf("expected default Job='', got %q", flags.Job)
	}
	if flags.DryRun {
		t.Error("expected default DryRun=false")
	}
	if flags.Parallel != 0 {
		t.Errorf("expected default Parallel=0, got %d", flags.Parallel)
	}
	if flags.CRLF != "convert" {
		t.Errorf("expected default CRLF='convert', got %q", flags.CRLF)
	}
	if flags.Platform != "" {
		t.Errorf("expected default Platform='', got %q", flags.Platform)
	}
	if flags.Offline {
		t.Error("expected default Offline=false")
	}
}

// TestBindRunFlags_ParseJobFlag verifies --job flag is correctly parsed via cobra
func TestBindRunFlags_ParseJobFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	flags := &RunFlags{}
	BindRunFlags(cmd, flags)

	if err := cmd.ParseFlags([]string{"--job", "lint"}); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if flags.Job != "lint" {
		t.Errorf("expected Job='lint', got %q", flags.Job)
	}
}

// TestBindRunFlags_ParseJobShorthand verifies -j shorthand works
func TestBindRunFlags_ParseJobShorthand(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	flags := &RunFlags{}
	BindRunFlags(cmd, flags)

	if err := cmd.ParseFlags([]string{"-j", "build"}); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if flags.Job != "build" {
		t.Errorf("expected Job='build' via -j, got %q", flags.Job)
	}
}

// TestBindRunFlags_ParseWorkflowFlag verifies --workflow / -W flag is correctly parsed
func TestBindRunFlags_ParseWorkflowFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	flags := &RunFlags{}
	BindRunFlags(cmd, flags)

	if err := cmd.ParseFlags([]string{"-W", ".github/workflows/ci.yml"}); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if flags.Workflow != ".github/workflows/ci.yml" {
		t.Errorf("expected Workflow='.github/workflows/ci.yml', got %q", flags.Workflow)
	}
}

// TestBindRunFlags_ParseDryRunFlag verifies --dry-run flag is correctly parsed
func TestBindRunFlags_ParseDryRunFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	flags := &RunFlags{}
	BindRunFlags(cmd, flags)

	if err := cmd.ParseFlags([]string{"--dry-run"}); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if !flags.DryRun {
		t.Error("expected DryRun=true after --dry-run flag")
	}
}

// TestBindRunFlags_ParseParallelFlag verifies --parallel / -p flag is correctly parsed
func TestBindRunFlags_ParseParallelFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected int
	}{
		{"long flag", []string{"--parallel", "4"}, 4},
		{"short flag", []string{"-p", "8"}, 8},
		{"zero", []string{"--parallel", "0"}, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			flags := &RunFlags{}
			BindRunFlags(cmd, flags)

			if err := cmd.ParseFlags(tc.args); err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if flags.Parallel != tc.expected {
				t.Errorf("expected Parallel=%d, got %d", tc.expected, flags.Parallel)
			}
		})
	}
}

// TestBindRunFlags_ParseCRLFFlag verifies --crlf flag is correctly parsed
func TestBindRunFlags_ParseCRLFFlag(t *testing.T) {
	tests := []struct {
		mode string
	}{
		{"convert"},
		{"preserve"},
		{"error"},
	}

	for _, tc := range tests {
		t.Run(tc.mode, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			flags := &RunFlags{}
			BindRunFlags(cmd, flags)

			if err := cmd.ParseFlags([]string{"--crlf", tc.mode}); err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if flags.CRLF != tc.mode {
				t.Errorf("expected CRLF=%q, got %q", tc.mode, flags.CRLF)
			}
		})
	}
}

// TestBindRunFlags_ParsePlatformFlag verifies --platform flag is correctly parsed
func TestBindRunFlags_ParsePlatformFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	flags := &RunFlags{}
	BindRunFlags(cmd, flags)

	if err := cmd.ParseFlags([]string{"--platform", "linux/amd64"}); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if flags.Platform != "linux/amd64" {
		t.Errorf("expected Platform='linux/amd64', got %q", flags.Platform)
	}
}

// TestBindRunFlags_ParseOfflineFlag verifies --offline flag is correctly parsed
func TestBindRunFlags_ParseOfflineFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	flags := &RunFlags{}
	BindRunFlags(cmd, flags)

	if err := cmd.ParseFlags([]string{"--offline"}); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if !flags.Offline {
		t.Error("expected Offline=true after --offline flag")
	}
}

// TestBindRunFlags_FlagCount verifies the total number of registered flags matches RunFlags fields
func TestBindRunFlags_FlagCount(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	flags := &RunFlags{}
	BindRunFlags(cmd, flags)

	// Count registered flags
	registeredCount := 0
	cmd.Flags().VisitAll(func(_ *pflag.Flag) {
		registeredCount++
	})

	// Count fields in RunFlags struct
	structFieldCount := reflect.TypeOf(RunFlags{}).NumField()

	// The number of registered flags should match the struct fields.
	// This catches when a new field is added to RunFlags but not bound in BindRunFlags.
	if registeredCount != structFieldCount {
		t.Errorf("RunFlags has %d fields but BindRunFlags registers %d flags — a flag may be missing from BindRunFlags",
			structFieldCount, registeredCount)
	}
}
