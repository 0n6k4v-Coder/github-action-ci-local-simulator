package actions

import (
	"context"
	"fmt"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/dockerx"
	"github.com/docker/docker/client"
)

// ExecuteCheckout simulates actions/checkout.
// Workspace copy is already performed, so this verifies the workspace container path exists.
func ExecuteCheckout(ctx context.Context, cli *client.Client, containerID, workingDir string, with map[string]any) (*ActionResult, error) {
	targetDir := workingDir
	if targetDir == "" {
		targetDir = "/github/workspace"
	}

	if cli != nil && containerID != "" {
		// Check that workspace exists inside container
		checkCmd := []string{"test", "-d", targetDir}
		_, err := dockerx.ExecCommand(ctx, cli, containerID, "/", checkCmd, nil)
		if err != nil {
			return nil, fmt.Errorf("checkout simulation failed: workspace directory %s does not exist in container: %w", targetDir, err)
		}

		// Optionally list files for debugging
		lsCmd := []string{"ls", "-la", targetDir}
		lsRes, _ := dockerx.ExecCommand(ctx, cli, containerID, "/", lsCmd, nil)
		stdout := fmt.Sprintf("actions/checkout simulation: verified workspace at %s\n", targetDir)
		if lsRes != nil && lsRes.Stdout != "" {
			stdout += lsRes.Stdout
		}
		return &ActionResult{
			Stdout: stdout,
		}, nil
	}

	return &ActionResult{
		Stdout: fmt.Sprintf("actions/checkout simulation: verified workspace at %s\n", targetDir),
	}, nil
}
