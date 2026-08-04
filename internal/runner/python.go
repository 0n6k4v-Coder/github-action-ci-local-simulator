package runner

import (
	"context"
	"fmt"
	"strings"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/dockerx"
	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
	"github.com/docker/docker/client"
)

// Commands that indicate Python usage
var pythonCommandPrefixes = []string{
	"pip ",
	"pip3 ",
	"python ",
	"python3 ",
	"pytest ",
	"ruff ",
	"black ",
	"mypy ",
	"uvicorn ",
	"gunicorn ",
}

var execCommand = dockerx.ExecCommand

func looksLikePythonCommand(cmd string) bool {
	lines := strings.Split(cmd, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, prefix := range pythonCommandPrefixes {
			if strings.HasPrefix(trimmed, prefix) || trimmed == strings.TrimSpace(prefix) {
				return true
			}
		}
	}
	return false
}

func jobHasPythonCommands(job workflow.Job) bool {
	for _, step := range job.Steps {
		if step.Run != "" && looksLikePythonCommand(step.Run) {
			return true
		}
	}
	return false
}

func jobHasSetupPython(job workflow.Job) bool {
	for _, step := range job.Steps {
		if step.Uses != "" && strings.Contains(step.Uses, "actions/setup-python") {
			return true
		}
	}
	return false
}

func ensurePythonInstalled(ctx context.Context, cli *client.Client, containerID string, hasSetupPython bool, imageName string) error {
	// Skip if setup-python action is present (it handles Python)
	if hasSetupPython {
		return nil
	}

	// Skip if image is already python-based
	if strings.HasPrefix(imageName, "python:") || imageName == "python" {
		return nil
	}

	// Check if python3 is already available
	checkCmd := []string{"which", "python3"}
	res, err := execCommand(ctx, cli, containerID, "/", checkCmd, nil)
	if err == nil && res != nil && res.ExitCode == 0 {
		return nil // Already installed, skip
	}

	// Install python3 + pip + venv
	installCmd := []string{
		"bash", "-c",
		"apt-get update -qq && apt-get install -y -qq python3 python3-pip python3-venv > /dev/null 2>&1",
	}
	result, err := execCommand(ctx, cli, containerID, "/", installCmd, nil)
	if err != nil {
		return fmt.Errorf("auto-install python3 failed: %w", err)
	}
	if result != nil && result.ExitCode != 0 {
		return fmt.Errorf("auto-install python3 failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	fmt.Printf("  ℹ️ Auto-installed python3 + pip for ubuntu image (simulating GitHub runner)\n")
	return nil
}
