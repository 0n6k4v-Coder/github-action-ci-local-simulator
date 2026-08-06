package integration

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/docker/docker/client"
)

// IsDockerAvailable checks if Docker daemon is accessible
func IsDockerAvailable() bool {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return false
	}
	defer cli.Close()

	ctx := context.Background()
	_, err = cli.Ping(ctx)
	return err == nil
}

// SkipIfNoDocker skips test if Docker is not available
func SkipIfNoDocker(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if !IsDockerAvailable() {
		t.Skip("skipping integration test: Docker not available")
	}
}

// CreateDockerClient creates a new Docker client
func CreateDockerClient(t *testing.T) *client.Client {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatalf("failed to create Docker client: %v", err)
	}
	return cli
}

// RunGacils runs gacils binary with given arguments
func RunGacils(t *testing.T, args ...string) (string, error) {
	cmd := exec.Command("./gacils", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// RunGacilsInDir runs gacils binary inside a specific working directory
func RunGacilsInDir(t *testing.T, dir string, args ...string) (string, error) {
	cmd := exec.Command("./gacils", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0755)
}

// BuildGacils builds the gacils binary
func BuildGacils(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "gacils", "../../cmd/gacils")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build gacils: %v", err)
	}
}

// CleanupGacils removes the gacils binary
func CleanupGacils(t *testing.T) {
	os.Remove("gacils")
}
// unexported aliases for backward compatibility within integration package
func isDockerAvailable() bool { return IsDockerAvailable() }
func skipIfNoDocker(t *testing.T) { SkipIfNoDocker(t) }
func createDockerClient(t *testing.T) *client.Client { return CreateDockerClient(t) }
func runGacils(t *testing.T, args ...string) (string, error) { return RunGacils(t, args...) }
func runGacilsInDir(t *testing.T, dir string, args ...string) (string, error) { return RunGacilsInDir(t, dir, args...) }
func copyFile(src, dst string) error { return CopyFile(src, dst) }
func buildGacils(t *testing.T) { BuildGacils(t) }
func cleanupGacils(t *testing.T) { CleanupGacils(t) }
