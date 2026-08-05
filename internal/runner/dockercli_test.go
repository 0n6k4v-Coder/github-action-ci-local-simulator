package runner

import (
	"context"
	"strings"
	"testing"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/dockerx"
	"github.com/docker/docker/client"
)

func TestLooksLikeDockerCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected bool
	}{
		{
			name:     "docker build",
			cmd:      "docker build -t test .",
			expected: true,
		},
		{
			name:     "docker push",
			cmd:      "docker push myimage",
			expected: true,
		},
		{
			name:     "npm install",
			cmd:      "npm install",
			expected: false,
		},
		{
			name:     "echo docker",
			cmd:      "echo docker",
			expected: false,
		},
		{
			name:     "docker run",
			cmd:      "docker run --rm alpine echo hi",
			expected: true,
		},
		{
			name:     "docker pull",
			cmd:      "docker pull ubuntu",
			expected: true,
		},
		{
			name:     "docker compose",
			cmd:      "docker compose up -d",
			expected: true,
		},
		{
			name:     "docker-compose",
			cmd:      "docker-compose up -d",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeDockerCommand(tt.cmd)
			if got != tt.expected {
				t.Errorf("looksLikeDockerCommand(%q) = %v; want %v", tt.cmd, got, tt.expected)
			}
		})
	}
}

func TestEnsureDockerCLIAvailable_SkippedWhenDockerImage(t *testing.T) {
	ctx := context.Background()
	// Should skip immediately when image is docker:* (cli can be nil)
	err := ensureDockerCLIAvailable(ctx, nil, "container-id", "docker:dind")
	if err != nil {
		t.Fatalf("expected nil error when image is docker:dind, got %v", err)
	}
}

func TestEnsureDockerCLIAvailable_Idempotent(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()

	installed := false
	installCount := 0

	execCommand = func(ctx context.Context, cli *client.Client, containerID, workingDir string, cmd []string, env map[string]string) (*dockerx.ExecResult, error) {
		if len(cmd) > 0 && cmd[0] == "which" {
			if installed {
				return &dockerx.ExecResult{ExitCode: 0, Stdout: "/usr/bin/docker\n"}, nil
			}
			return &dockerx.ExecResult{ExitCode: 1, Stderr: "no docker in path\n"}, nil
		}
		if len(cmd) > 2 && strings.Contains(cmd[2], "apt-get install") {
			installCount++
			installed = true
			return &dockerx.ExecResult{ExitCode: 0}, nil
		}
		return &dockerx.ExecResult{ExitCode: 0}, nil
	}

	ctx := context.Background()
	// First call: docker not found, triggers install
	err := ensureDockerCLIAvailable(ctx, nil, "container-1", "ubuntu:24.04")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if installCount != 1 {
		t.Errorf("expected 1 install call, got %d", installCount)
	}

	// Second call: docker now available, skips install
	err = ensureDockerCLIAvailable(ctx, nil, "container-1", "ubuntu:24.04")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if installCount != 1 {
		t.Errorf("expected still 1 install call after second call (idempotent), got %d", installCount)
	}
}
