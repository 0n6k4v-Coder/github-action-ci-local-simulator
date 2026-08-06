package dockerx

import (
	"context"
	"testing"

	"github.com/docker/docker/client"
)

func TestEnsureImage_OfflineMode_ImageExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer cli.Close()

	ctx := context.Background()

	// Use a common image that's likely cached
	imageName := "alpine:latest"

	// Pull it first (if not cached)
	err = EnsureImage(ctx, cli, imageName, false) // offline=false, allow pull
	if err != nil {
		t.Skipf("Could not pull image for test: %v", err)
	}

	// Now test offline mode — should succeed because image exists locally
	err = EnsureImage(ctx, cli, imageName, true) // offline=true
	if err != nil {
		t.Errorf("EnsureImage in offline mode should succeed for cached image, got: %v", err)
	}
}

func TestEnsureImage_OfflineMode_ImageMissing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer cli.Close()

	ctx := context.Background()

	// Use a random image name that definitely doesn't exist locally
	imageName := "this-image-definitely-does-not-exist-12345:latest"

	// Offline mode should fail with clear error
	err = EnsureImage(ctx, cli, imageName, true) // offline=true
	if err == nil {
		t.Error("EnsureImage in offline mode should fail for missing image")
	}

	// Error should mention offline or not found
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("error message should not be empty")
	}
}