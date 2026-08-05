package runner

import (
	"context"
	"strings"
	"testing"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/dockerx"
	"github.com/docker/docker/client"
)

func TestLooksLikePythonCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected bool
	}{
		{
			name:     "pip install flask",
			cmd:      "pip install flask",
			expected: true,
		},
		{
			name:     "python3 script.py",
			cmd:      "python3 script.py",
			expected: true,
		},
		{
			name:     "npm install",
			cmd:      "npm install",
			expected: false,
		},
		{
			name:     "echo pip",
			cmd:      "echo pip",
			expected: false,
		},
		{
			name:     "pytest",
			cmd:      "pytest --version",
			expected: true,
		},
		{
			name:     "ruff check",
			cmd:      "ruff check .",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikePythonCommand(tt.cmd)
			if got != tt.expected {
				t.Errorf("looksLikePythonCommand(%q) = %v; want %v", tt.cmd, got, tt.expected)
			}
		})
	}
}

func TestEnsurePythonInstalled_SkippedWhenSetupPythonPresent(t *testing.T) {
	ctx := context.Background()
	// Should skip immediately without calling execCommand (cli can be nil)
	err := ensurePythonInstalled(ctx, nil, "container-id", true, "ubuntu:24.04")
	if err != nil {
		t.Fatalf("expected nil error when setup-python is present, got %v", err)
	}
}

func TestEnsurePythonInstalled_SkippedWhenPythonImage(t *testing.T) {
	ctx := context.Background()
	// Should skip immediately when image is python:* (cli can be nil)
	err := ensurePythonInstalled(ctx, nil, "container-id", false, "python:3.12")
	if err != nil {
		t.Fatalf("expected nil error when image is python:3.12, got %v", err)
	}
}

func TestEnsurePythonInstalled_Idempotent(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()

	installed := false
	installCount := 0

	execCommand = func(ctx context.Context, cli *client.Client, containerID, workingDir string, cmd []string, env map[string]string) (*dockerx.ExecResult, error) {
		if len(cmd) > 0 && cmd[0] == "which" {
			if installed {
				return &dockerx.ExecResult{ExitCode: 0, Stdout: "/usr/bin/python3\n"}, nil
			}
			return &dockerx.ExecResult{ExitCode: 1, Stderr: "no python3 in path\n"}, nil
		}
		if len(cmd) > 2 && strings.Contains(cmd[2], "apt-get install") {
			installCount++
			installed = true
			return &dockerx.ExecResult{ExitCode: 0}, nil
		}
		return &dockerx.ExecResult{ExitCode: 0}, nil
	}

	ctx := context.Background()
	// First call: python3 not found, triggers install
	err := ensurePythonInstalled(ctx, nil, "container-1", false, "ubuntu:24.04")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if installCount != 1 {
		t.Errorf("expected 1 install call, got %d", installCount)
	}

	// Second call: python3 now available, skips install
	err = ensurePythonInstalled(ctx, nil, "container-1", false, "ubuntu:24.04")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if installCount != 1 {
		t.Errorf("expected still 1 install call after second call (idempotent), got %d", installCount)
	}
}

func TestEnsurePythonInstalled_RemovesExternallyManaged(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()

	var capturedCmd []string
	execCommand = func(ctx context.Context, cli *client.Client, containerID, workingDir string, cmd []string, env map[string]string) (*dockerx.ExecResult, error) {
		if len(cmd) > 0 && cmd[0] == "which" {
			return &dockerx.ExecResult{ExitCode: 1}, nil // python3 not found
		}
		if len(cmd) > 2 && strings.Contains(cmd[2], "apt-get install") {
			capturedCmd = cmd
			return &dockerx.ExecResult{ExitCode: 0}, nil
		}
		return &dockerx.ExecResult{ExitCode: 0}, nil
	}

	ctx := context.Background()
	err := ensurePythonInstalled(ctx, nil, "container-1", false, "ubuntu:24.04")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify install command contains EXTERNALLY-MANAGED removal
	if len(capturedCmd) < 3 {
		t.Fatalf("captured command too short: %v", capturedCmd)
	}

	installScript := capturedCmd[2]
	if !strings.Contains(installScript, "rm -f /usr/lib/python3.*/EXTERNALLY-MANAGED") {
		t.Errorf("install command should remove EXTERNALLY-MANAGED, got: %s", installScript)
	}
}
