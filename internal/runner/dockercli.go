package runner

import (
	"context"
	"fmt"
	"strings"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
	"github.com/docker/docker/client"
)

var dockerCommandPrefixes = []string{
	"docker ",
	"docker build ",
	"docker push ",
	"docker run ",
	"docker pull ",
	"docker compose ",
	"docker-compose ",
}

func looksLikeDockerCommand(cmd string) bool {
	lines := strings.Split(cmd, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, prefix := range dockerCommandPrefixes {
			if strings.HasPrefix(trimmed, prefix) || trimmed == strings.TrimSpace(prefix) {
				return true
			}
		}
	}
	return false
}

func jobHasDockerCommands(job workflow.Job) bool {
	for _, step := range job.Steps {
		if step.Run != "" && looksLikeDockerCommand(step.Run) {
			return true
		}
	}
	return false
}

func ensureDockerCLIAvailable(ctx context.Context, cli *client.Client, containerID string, imageName string) error {
	// Skip if image is docker:dind or already has docker
	if strings.HasPrefix(imageName, "docker:") {
		return nil
	}

	// Check if docker CLI is already available
	checkCmd := []string{"which", "docker"}
	res, err := execCommand(ctx, cli, containerID, "/", checkCmd, nil)
	if err == nil && res != nil && res.ExitCode == 0 {
		return nil // Already installed
	}

	// Install docker-ce-cli (or fallback to docker.io / docker.io-cli package)
	installCmd := []string{
		"bash", "-c",
		"apt-get update -qq && (apt-get install -y -qq docker-ce-cli || apt-get install -y -qq docker.io || apt-get install -y -qq docker.io-cli) > /dev/null 2>&1",
	}
	result, err := execCommand(ctx, cli, containerID, "/", installCmd, nil)
	if err != nil {
		return fmt.Errorf("auto-install docker CLI failed: %w", err)
	}
	if result != nil && result.ExitCode != 0 {
		return fmt.Errorf("auto-install docker CLI failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	fmt.Printf("  ℹ️ Auto-installed docker-ce-cli for docker commands\n")
	return nil
}
