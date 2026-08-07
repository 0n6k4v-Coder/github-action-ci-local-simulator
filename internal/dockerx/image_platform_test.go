package dockerx

import (
	"context"
	"testing"

	"github.com/docker/docker/client"
)

func TestEnsureImage_WithPlatform(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer cli.Close()

	ctx := context.Background()

	// Test with explicit platform - use a common image
	imageName := "alpine:latest"
	platform := "linux/amd64"

	// This should succeed with the platform parameter
	err = EnsureImage(ctx, cli, imageName, false, platform)
	if err != nil {
		t.Errorf("EnsureImage with platform should succeed, got: %v", err)
	}
}

func TestEnsureImage_EmptyPlatform(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer cli.Close()

	ctx := context.Background()

	// Test with empty platform (default behavior)
	imageName := "alpine:latest"
	platform := ""

	// This should succeed with empty platform (uses host default)
	err = EnsureImage(ctx, cli, imageName, false, platform)
	if err != nil {
		t.Errorf("EnsureImage with empty platform should succeed, got: %v", err)
	}
}

func TestParsePlatform_ValidInputs(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expectOS   string
		expectArch string
		wantErr    bool
	}{
		{"empty", "", "", "", false},
		{"linux/amd64", "linux/amd64", "linux", "amd64", false},
		{"linux/arm64", "linux/arm64", "linux", "arm64", false},
		{"linux/amd64/v2", "linux/amd64/v2", "linux", "amd64/v2", false}, // 3-part: os/arch/variant
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os, arch, err := ParsePlatform(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %q: %v", tt.input, err)
				}
				if os != tt.expectOS {
					t.Errorf("expected os %q, got %q", tt.expectOS, os)
				}
				if arch != tt.expectArch {
					t.Errorf("expected arch %q, got %q", tt.expectArch, arch)
				}
			}
		})
	}
}

// TestParsePlatform_ThreePartPlatform verifies that ParsePlatform correctly
// handles Docker SDK 3-part platform strings like "linux/amd64/v2"
// (os/arch/variant). Before the fix it returned empty os and arch.
func TestParsePlatform_ThreePartPlatform(t *testing.T) {
	// Docker SDK supports "linux/amd64/v2" (os/arch/variant)
	os, arch, err := ParsePlatform("linux/amd64/v2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if os != "linux" {
		t.Errorf("expected os 'linux', got %q", os)
	}
	if arch == "" {
		t.Fatalf("expected non-empty arch for 'linux/amd64/v2', got empty string")
	}
	t.Logf("ParsePlatform(linux/amd64/v2) = (%q, %q)", os, arch)
}

// TestParsePlatform_VariousFormats verifies multiple platform string formats.
func TestParsePlatform_VariousFormats(t *testing.T) {
	tests := []struct {
		input    string
		expectOS string
		wantErr  bool
	}{
		{"", "", false},
		{"linux/amd64", "linux", false},
		{"linux/arm64", "linux", false},
		{"linux/amd64/v2", "linux", false}, // 3-part: must NOT return empty
		{"linux/arm64/v8", "linux", false}, // 3-part variant
		{"windows/amd64", "windows", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			os, arch, err := ParsePlatform(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for %q: %v", tt.input, err)
				}
				if tt.expectOS != "" && os != tt.expectOS {
					t.Errorf("expected os %q for %q, got %q", tt.expectOS, tt.input, os)
				}
				if tt.input != "" && arch == "" {
					t.Errorf("expected non-empty arch for %q, got empty", tt.input)
				}
			}
		})
	}
}
