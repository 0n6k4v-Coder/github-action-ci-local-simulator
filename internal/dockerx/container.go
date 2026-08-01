package dockerx

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

// CreateContainer creates a new Docker container for running a job.
func CreateContainer(ctx context.Context, cli *client.Client, imageName, workingDir string, githubEnvPath, githubPathPath string, githubContext, runnerContext map[string]string) (string, error) {
	// Build environment variables
	envVars := []string{
		"CI=true",
		"GITHUB_ACTIONS=true",
		"RUNNER_OS=Linux",
		"RUNNER_ARCH=X64",
		"RUNNER_NAME=gacils-local",
		"RUNNER_TEMP=/tmp/gacils",
		"RUNNER_TOOL_CACHE=/opt/hostedtoolcache",
		"RUNNER_DEBUG=0",
	}

	// Add GitHub context variables
	for k, v := range githubContext {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	// Add runner context variables
	for k, v := range runnerContext {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	// Add GITHUB_ENV and GITHUB_PATH
	if githubEnvPath != "" {
		envVars = append(envVars, fmt.Sprintf("GITHUB_ENV=%s", githubEnvPath))
	}
	if githubPathPath != "" {
		envVars = append(envVars, fmt.Sprintf("GITHUB_PATH=%s", githubPathPath))
	}

	// Create mount for /tmp/gacils as a named volume to share GITHUB_ENV, GITHUB_PATH, and GITHUB_OUTPUT files
	// This works in Docker Desktop where the host path may not be accessible
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: "gacils-tmp",
			Target: "/tmp/gacils",
		},
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:      imageName,
		WorkingDir: workingDir,
		Entrypoint: []string{"tail", "-f", "/dev/null"}, // Keep container running
		Tty:        false,
		Env:        envVars,
	}, &container.HostConfig{
		AutoRemove: false,
		Mounts:     mounts,
	}, &network.NetworkingConfig{}, nil, "")
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}

	return resp.ID, nil
}

// StartContainer starts a Docker container.
func StartContainer(ctx context.Context, cli *client.Client, containerID string) error {
	if err := cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("start container: %w", err)
	}
	return nil
}

// RemoveContainer removes a Docker container.
func RemoveContainer(ctx context.Context, cli *client.Client, containerID string) error {
	if err := cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: true,
	}); err != nil {
		return fmt.Errorf("remove container: %w", err)
	}
	return nil
}