package cli

import (
	"testing"
)

// TestRunFlags_Defaults verifies zero-value struct field defaults.
// Note: CRLF defaults to "convert" only after BindRunFlags is called (cobra sets it).
// Zero-value Go struct has CRLF="". See TestBindRunFlags_Defaults in flags_test.go for cobra defaults.
func TestRunFlags_Defaults(t *testing.T) {
	flags := &RunFlags{}

	// Verify zero-value struct defaults
	if flags.Workflow != "" {
		t.Errorf("expected empty workflow default, got %q", flags.Workflow)
	}
	if flags.Job != "" {
		t.Errorf("expected empty job default, got %q", flags.Job)
	}
	if flags.DryRun {
		t.Error("expected DryRun default to be false")
	}
	if flags.Parallel != 0 {
		t.Errorf("expected Parallel default to be 0, got %d", flags.Parallel)
	}
	// CRLF zero-value is "" — cobra sets the "convert" default via BindRunFlags
	if flags.CRLF != "" {
		t.Errorf("expected zero-value CRLF='', got %q", flags.CRLF)
	}
	if flags.Platform != "" {
		t.Errorf("expected empty Platform default, got %q", flags.Platform)
	}
	if flags.Offline {
		t.Error("expected Offline default to be false")
	}
}

// TestRunFlags_JobFlag verifies --job flag is properly defined
func TestRunFlags_JobFlag(t *testing.T) {
	flags := &RunFlags{}
	flags.Job = "lint"

	if flags.Job != "lint" {
		t.Errorf("expected job 'lint', got %q", flags.Job)
	}
}

// TestRunFlags_WorkflowFlag verifies --workflow flag
func TestRunFlags_WorkflowFlag(t *testing.T) {
	flags := &RunFlags{}
	flags.Workflow = ".github/workflows/ci.yml"

	if flags.Workflow != ".github/workflows/ci.yml" {
		t.Errorf("expected workflow path, got %q", flags.Workflow)
	}
}

// TestRunFlags_DryRunFlag verifies --dry-run flag
func TestRunFlags_DryRunFlag(t *testing.T) {
	flags := &RunFlags{}
	flags.DryRun = true

	if !flags.DryRun {
		t.Error("expected DryRun to be true")
	}
}

// TestRunFlags_ParallelFlag verifies --parallel flag
func TestRunFlags_ParallelFlag(t *testing.T) {
	flags := &RunFlags{}
	flags.Parallel = 4

	if flags.Parallel != 4 {
		t.Errorf("expected Parallel 4, got %d", flags.Parallel)
	}
}

// TestRunFlags_CRLFFlag verifies --crlf flag
func TestRunFlags_CRLFFlag(t *testing.T) {
	flags := &RunFlags{}
	flags.CRLF = "preserve"

	if flags.CRLF != "preserve" {
		t.Errorf("expected CRLF 'preserve', got %q", flags.CRLF)
	}
}

// TestRunFlags_PlatformFlag verifies --platform flag
func TestRunFlags_PlatformFlag(t *testing.T) {
	flags := &RunFlags{}
	flags.Platform = "linux/amd64"

	if flags.Platform != "linux/amd64" {
		t.Errorf("expected Platform 'linux/amd64', got %q", flags.Platform)
	}
}

// TestRunFlags_OfflineFlag verifies --offline flag
func TestRunFlags_OfflineFlag(t *testing.T) {
	flags := &RunFlags{}
	flags.Offline = true

	if !flags.Offline {
		t.Error("expected Offline to be true")
	}
}
