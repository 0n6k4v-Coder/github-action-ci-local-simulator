package dockerx

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// ExecResult represents the result of executing a command in a container.
type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// ExecCommand executes a command inside a running container.
func ExecCommand(ctx context.Context, cli *client.Client, containerID, workingDir string, cmd []string) (*ExecResult, error) {
	// Create exec instance
	execConfig := container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
		WorkingDir:   workingDir,
	}

	createResp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("create exec: %w", err)
	}

	// Attach to exec
	attachResp, err := cli.ContainerExecAttach(ctx, createResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return nil, fmt.Errorf("attach exec: %w", err)
	}
	defer attachResp.Close()

	// Read output
	stdout, stderr, err := readExecOutput(attachResp.Reader)
	if err != nil {
		return nil, fmt.Errorf("read exec output: %w", err)
	}

	// Inspect exec to get exit code
	inspectResp, err := cli.ContainerExecInspect(ctx, createResp.ID)
	if err != nil {
		return nil, fmt.Errorf("inspect exec: %w", err)
	}

	return &ExecResult{
		ExitCode: inspectResp.ExitCode,
		Stdout:   stdout,
		Stderr:   stderr,
	}, nil
}

// readExecOutput reads stdout and stderr from the exec attach reader.
// The Docker exec output uses a multiplexed format with 8-byte headers.
func readExecOutput(reader io.Reader) (string, string, error) {
	var stdout, stderr string
	header := make([]byte, 8)

	for {
		// Read header
		n, err := io.ReadFull(reader, header)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return "", "", err
		}
		if n != 8 {
			break
		}

		// Parse header: stream type (1=stdout, 2=stderr), size (4 bytes, big-endian)
		streamType := header[0]
		size := int(header[4])<<24 | int(header[5])<<16 | int(header[6])<<8 | int(header[7])

		// Read payload
		payload := make([]byte, size)
		n, err = io.ReadFull(reader, payload)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return "", "", err
		}

		if streamType == 1 {
			stdout += string(payload)
		} else if streamType == 2 {
			stderr += string(payload)
		}
	}

	return stdout, stderr, nil
}