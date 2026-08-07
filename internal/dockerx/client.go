package dockerx

import (
	"github.com/docker/docker/client"
)

// CreateDockerClient creates a Docker client that respects environment variables
// and automatically detects the correct socket path (Linux native, WSL2, etc.)
func CreateDockerClient() (*client.Client, error) {
	return client.NewClientWithOpts(
		client.FromEnv,                     // Read DOCKER_HOST env var
		client.WithAPIVersionNegotiation(), // Auto-negotiate API version
	)
}
