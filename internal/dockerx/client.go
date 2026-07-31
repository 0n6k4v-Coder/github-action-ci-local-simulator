package dockerx

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
)

// NewClient creates a new Docker client with environment-based configuration
// and API version negotiation.
func NewClient(ctx context.Context) (*client.Client, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}

	// Ping to verify Docker daemon is reachable
	if _, err := cli.Ping(ctx); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("docker daemon unavailable: %w", err)
	}

	return cli, nil
}