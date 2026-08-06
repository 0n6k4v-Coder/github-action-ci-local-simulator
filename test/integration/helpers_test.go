package integration

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/docker/docker/client"
)

// isDockerAvailable checks if Docker daemon is accessible
func isDockerAvailable() bool {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return false
	}
	defer cli.Close()

	ctx := context.Background()
	_, err = cli.Ping(ctx)
	return err == nil
}

// skipIfNoDocker skips test if Docker is not available
func skipIfNoDocker(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if !isDockerAvailable() {
		t.Skip("skipping integration test: Docker not available")
	}
}

// createDockerClient creates a new Docker client
func createDockerClient(t *testing.T) *client.Client {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatalf("failed to create Docker client: %v", err)
	}
	return cli
}

// runGacils runs gacils binary with given arguments
func runGacils(t *testing.T, args ...string) (string, error) {
	cmd := exec.Command("./gacils", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// runGacilsInDir runs gacils binary inside a specific working directory
func runGacilsInDir(t *testing.T, dir string, args ...string) (string, error) {
	cmd := exec.Command("./gacils", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0755)
}

// buildGacils builds the gacils binary
func buildGacils(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "gacils", "../../cmd/gacils")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build gacils: %v", err)
	}
}

// cleanupGacils removes the gacils binary
func cleanupGacils(t *testing.T) {
	os.Remove("gacils")
}
