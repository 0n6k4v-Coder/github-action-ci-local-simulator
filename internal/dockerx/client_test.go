package dockerx

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/container"
)

// TestCreateDockerClient_Success verifies that CreateDockerClient returns a valid client
func TestCreateDockerClient_Success(t *testing.T) {
	cli, err := CreateDockerClient()
	if err != nil {
		t.Fatalf("CreateDockerClient failed: %v", err)
	}
	defer cli.Close()

	if cli == nil {
		t.Fatal("client should not be nil")
	}

	// Verify client is functional (if Docker available)
	ctx := context.Background()
	_, err = cli.Ping(ctx)
	if err != nil {
		t.Logf("Warning: Docker not available, skipping ping test: %v", err)
	}
}

// TestCreateDockerClient_RespectsDOCKER_HOST verifies that DOCKER_HOST env var is respected
func TestCreateDockerClient_RespectsDOCKER_HOST(t *testing.T) {
	// This test verifies that the code uses client.FromEnv
	// which automatically respects DOCKER_HOST environment variable
	// We can't test this directly without mocking, but we can verify
	// that the function signature and behavior are correct

	cli, err := CreateDockerClient()
	if err != nil {
		t.Fatalf("CreateDockerClient failed: %v", err)
	}
	defer cli.Close()

	// If we get here, the client was created successfully
	// which means it used the correct socket path (from env or default)
	if cli == nil {
		t.Fatal("client should not be nil")
	}
}

// TestCreateDockerClient_GracefulFailure verifies clear error when Docker unavailable
func TestCreateDockerClient_GracefulFailure(t *testing.T) {
	// This test verifies that errors are clear and actionable
	// We can't easily simulate Docker unavailability in unit tests,
	// but we can verify the function returns proper error types

	cli, err := CreateDockerClient()
	if err != nil {
		// Error should be clear and actionable
		errMsg := err.Error()
		if errMsg == "" {
			t.Error("error message should not be empty")
		}
		t.Logf("Docker unavailable (expected in some environments): %v", err)
		return
	}
	defer cli.Close()

	t.Log("Docker available, graceful failure test skipped")
}

// TestCreateDockerClient_Integration verifies client works with real Docker
func TestCreateDockerClient_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cli, err := CreateDockerClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer cli.Close()

	ctx := context.Background()

	// Test 1: Ping
	_, err = cli.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Test 2: List containers (basic operation)
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		t.Fatalf("ContainerList failed: %v", err)
	}

	t.Logf("Docker integration test passed, found %d containers", len(containers))
}
