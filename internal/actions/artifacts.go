package actions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/dockerx"
	"github.com/docker/docker/client"
)

// ExecuteUploadArtifact simulates actions/upload-artifact.
func ExecuteUploadArtifact(ctx context.Context, cli *client.Client, containerID, workingDir string, with map[string]any) (*ActionResult, error) {
	name := "artifact"
	if val, ok := with["name"]; ok && fmt.Sprintf("%v", val) != "" {
		name = fmt.Sprintf("%v", val)
	}

	pathVal, ok := with["path"]
	if !ok || fmt.Sprintf("%v", pathVal) == "" {
		return nil, NewActionValidationError("action actions/upload-artifact requires input 'path'")
	}
	path := fmt.Sprintf("%v", pathVal)

	containerPath := resolveContainerPath(path, workingDir)
	hostArtifactsDir := GetArtifactsDir(ctx)
	jobID := GetJobID(ctx)

	dstHostPath := filepath.Join(hostArtifactsDir, name)
	if err := os.MkdirAll(dstHostPath, 0755); err != nil {
		return nil, fmt.Errorf("create artifact host dir: %w", err)
	}

	if cli != nil && containerID != "" {
		if err := dockerx.CopyContainerToHost(ctx, cli, containerID, containerPath, dstHostPath); err != nil {
			return nil, fmt.Errorf("upload artifact failed: %w", err)
		}
	}

	if jobID != "" {
		jobDstPath := filepath.Join(hostArtifactsDir, jobID, name)
		if err := os.MkdirAll(jobDstPath, 0755); err != nil {
			return nil, fmt.Errorf("create job artifact host dir: %w", err)
		}
		if cli != nil && containerID != "" {
			_ = dockerx.CopyContainerToHost(ctx, cli, containerID, containerPath, jobDstPath)
		}
	}

	return &ActionResult{
		Stdout: fmt.Sprintf("actions/upload-artifact simulation: Uploaded artifact %q from %s\n", name, containerPath),
	}, nil
}

// ExecuteDownloadArtifact simulates actions/download-artifact.
func ExecuteDownloadArtifact(ctx context.Context, cli *client.Client, containerID, workingDir string, with map[string]any) (*ActionResult, error) {
	name := ""
	if val, ok := with["name"]; ok && fmt.Sprintf("%v", val) != "" {
		name = fmt.Sprintf("%v", val)
	}

	dstContainerPath := workingDir
	if dstContainerPath == "" {
		dstContainerPath = "/github/workspace"
	}
	if pathVal, ok := with["path"]; ok && fmt.Sprintf("%v", pathVal) != "" {
		dstContainerPath = resolveContainerPath(fmt.Sprintf("%v", pathVal), workingDir)
	}

	hostArtifactsDir := GetArtifactsDir(ctx)

	srcHostPath := findArtifactDir(hostArtifactsDir, name)
	if srcHostPath == "" {
		return nil, fmt.Errorf("download artifact failed: artifact %q not found in %s", name, hostArtifactsDir)
	}

	if cli != nil && containerID != "" {
		if err := dockerx.CopyHostToContainer(ctx, cli, containerID, srcHostPath, dstContainerPath); err != nil {
			return nil, fmt.Errorf("download artifact failed: %w", err)
		}
	}

	return &ActionResult{
		Stdout: fmt.Sprintf("actions/download-artifact simulation: Downloaded artifact %q to %s\n", name, dstContainerPath),
	}, nil
}

// findArtifactDir locates the host directory containing artifact files.
func findArtifactDir(hostArtifactsDir, name string) string {
	if name != "" {
		// 1. Check hostArtifactsDir/name
		target := filepath.Join(hostArtifactsDir, name)
		if info, err := os.Stat(target); err == nil && info.IsDir() {
			return target
		}

		// 2. Check hostArtifactsDir/*/name
		entries, err := os.ReadDir(hostArtifactsDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					subTarget := filepath.Join(hostArtifactsDir, entry.Name(), name)
					if info, err := os.Stat(subTarget); err == nil && info.IsDir() {
						return subTarget
					}
				}
			}
		}
	} else {
		// If name is empty, select first available artifact directory
		entries, err := os.ReadDir(hostArtifactsDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					return filepath.Join(hostArtifactsDir, entry.Name())
				}
			}
		}
	}
	return ""
}
