package dockerx

import (
	"context"
	"io"

	"github.com/docker/docker/client"
)

// CopyConfig holds configuration for copying workspace to container.
type CopyConfig struct {
	SourcePath      string
	TargetPath      string
	ContainerID     string
	ExcludePatterns []string
	FollowSymlinks  bool
}

// CopyWorkspace copies the workspace to a running container.
// This is a minimal skeleton for Phase 4B workspace copy support.
func CopyWorkspace(ctx context.Context, cli *client.Client, config CopyConfig) error {
	// TODO: Implement workspace copy using docker cp or tar streaming
	// For now, this is a placeholder that will be implemented in Phase 4B
	return nil
}

// CopyToContainer copies files from host to container using tar stream.
func CopyToContainer(ctx context.Context, cli *client.Client, containerID, srcPath, dstPath string) error {
	// TODO: Implement using cli.CopyToContainer
	return nil
}

// CopyFromContainer copies files from container to host using tar stream.
func CopyFromContainer(ctx context.Context, cli *client.Client, containerID, srcPath, dstPath string) error {
	// TODO: Implement using cli.CopyFromContainer
	return nil
}

// TarDirectory creates a tar stream of a directory for docker copy operations.
func TarDirectory(srcPath string, excludePatterns []string) (io.Reader, error) {
	// TODO: Implement tar creation with exclusions
	return nil, nil
}

// EnsureWorkspaceDir ensures the workspace directory exists in the container.
func EnsureWorkspaceDir(ctx context.Context, cli *client.Client, containerID, workspacePath string) error {
	cmd := []string{"mkdir", "-p", workspacePath}
	_, err := ExecCommand(ctx, cli, containerID, "/", cmd, nil)
	return err
}

// CleanupWorkspace removes temporary workspace files from container.
func CleanupWorkspace(ctx context.Context, cli *client.Client, containerID, workspacePath string) error {
	cmd := []string{"rm", "-rf", workspacePath}
	_, err := ExecCommand(ctx, cli, containerID, "/", cmd, nil)
	return err
}