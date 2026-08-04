package dockerx

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// DefaultExcludePatterns are the default patterns to exclude when creating a tar archive.
var DefaultExcludePatterns = []string{
	".git",
	".gacils",
	".gacils-local",
	"node_modules",
	"__pycache__",
	".venv",
	"venv",
	"dist",
	"build",
	"target",
	"coverage.out",
	"*.log",
}

// CopyConfig holds configuration for copying workspace to container.
type CopyConfig struct {
	SourcePath      string
	TargetPath      string
	ContainerID     string
	ExcludePatterns []string
	FollowSymlinks  bool
}

// CopyWorkspace copies the workspace to a running container.
func CopyWorkspace(ctx context.Context, cli *client.Client, config CopyConfig) error {
	// Validate config
	if config.SourcePath == "" {
		return fmt.Errorf("source path is required")
	}
	if config.TargetPath == "" {
		config.TargetPath = "/github/workspace"
	}
	if config.ContainerID == "" {
		return fmt.Errorf("container ID is required")
	}

	// Ensure the target directory exists in the container
	if err := EnsureWorkspaceDir(ctx, cli, config.ContainerID, config.TargetPath); err != nil {
		return fmt.Errorf("ensure workspace dir: %w", err)
	}

	// Build exclude patterns
	excludePatterns := append(DefaultExcludePatterns, config.ExcludePatterns...)

	// Create tar stream from source directory
	tarReader, err := TarDirectory(config.SourcePath, excludePatterns)
	if err != nil {
		return fmt.Errorf("create tar stream: %w", err)
	}

	// Copy tar stream to container
	if err := CopyToContainer(ctx, cli, config.ContainerID, config.TargetPath, tarReader); err != nil {
		return fmt.Errorf("copy to container: %w", err)
	}

	return nil
}

// CopyToContainer copies files from host to container using tar stream.
func CopyToContainer(ctx context.Context, cli *client.Client, containerID, dstPath string, tarReader io.Reader) error {
	if containerID == "" {
		return fmt.Errorf("container ID is required")
	}
	if tarReader == nil {
		return fmt.Errorf("tar reader is required")
	}

	// Use Docker client's CopyToContainer API
	err := cli.CopyToContainer(ctx, containerID, dstPath, tarReader, container.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	})
	if err != nil {
		return fmt.Errorf("docker copy to container: %w", err)
	}

	return nil
}

// CopyFromContainer copies files from container to host using tar stream.
func CopyFromContainer(ctx context.Context, cli *client.Client, containerID, srcPath, dstHostPath string) error {
	if containerID == "" {
		return fmt.Errorf("container ID is required")
	}
	if srcPath == "" {
		return fmt.Errorf("source path is required")
	}
	if dstHostPath == "" {
		return fmt.Errorf("destination host path is required")
	}

	// Use Docker client's CopyFromContainer API
	reader, _, err := cli.CopyFromContainer(ctx, containerID, srcPath)
	if err != nil {
		return fmt.Errorf("docker copy from container: %w", err)
	}
	defer reader.Close()

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(dstHostPath, 0755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	// Extract tar content to host destination
	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}

		// Determine target path
		targetPath := filepath.Join(dstHostPath, header.Name)

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("create parent directory for %s: %w", targetPath, err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("create directory %s: %w", targetPath, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("create file %s: %w", targetPath, err)
			}
			if _, err := io.Copy(file, tr); err != nil {
				file.Close()
				return fmt.Errorf("write file %s: %w", targetPath, err)
			}
			file.Close()
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return fmt.Errorf("create symlink %s: %w", targetPath, err)
			}
		default:
			// Skip unsupported file types (e.g., device files, FIFOs)
			// but don't fail
		}
	}

	return nil
}

// TarDirectory creates a tar stream of a directory for docker copy operations.
func TarDirectory(srcPath string, excludePatterns []string) (io.Reader, error) {
	// Clean the source path
	srcPath = filepath.Clean(srcPath)

	// Check if source exists
	info, err := os.Stat(srcPath)
	if err != nil {
		return nil, fmt.Errorf("stat source path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("source path is not a directory: %s", srcPath)
	}

	// Create a pipe for streaming
	pr, pw := io.Pipe()

	// Create tar writer
	tw := tar.NewWriter(pw)

	// Start goroutine to walk directory and write to tar
	go func() {
		defer pw.Close()
		defer tw.Close()

		err := filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				// Skip unreadable files but log them
				fmt.Fprintf(os.Stderr, "Warning: skipping %s: %v\n", path, err)
				return nil
			}

			// Get relative path
			relPath, err := filepath.Rel(srcPath, path)
			if err != nil {
				return err
			}

			// Skip root directory entry
			if relPath == "." {
				return nil
			}

			// Check exclude patterns
			if shouldExclude(relPath, excludePatterns) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Create tar header - handle symlinks specially
			var header *tar.Header
			if info.Mode()&os.ModeSymlink != 0 {
				// It's a symlink - read the target
				target, err := os.Readlink(path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: cannot read symlink %s: %v\n", path, err)
					return nil
				}
				header, err = tar.FileInfoHeader(info, target)
				if err != nil {
					return fmt.Errorf("create tar header for %s: %w", path, err)
				}
				header.Typeflag = tar.TypeSymlink
				header.Linkname = target
			} else {
				header, err = tar.FileInfoHeader(info, "")
				if err != nil {
					return fmt.Errorf("create tar header for %s: %w", path, err)
				}
			}

			// Use relative path in archive
			header.Name = relPath

			// Write header
			if err := tw.WriteHeader(header); err != nil {
				return fmt.Errorf("write tar header for %s: %w", relPath, err)
			}

			// If it's a regular file, write content
			if info.Mode().IsRegular() {
				file, err := os.Open(path)
				if err != nil {
					// Skip unreadable files
					fmt.Fprintf(os.Stderr, "Warning: cannot read %s: %v\n", path, err)
					return nil
				}
				defer file.Close()

				if _, err := io.Copy(tw, file); err != nil {
					return fmt.Errorf("write file content for %s: %w", relPath, err)
				}
			}

			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error during tar creation: %v\n", err)
		}
	}()

	return pr, nil
}

// shouldExclude checks if a path matches any of the exclude patterns.
func shouldExclude(path string, patterns []string) bool {
	for _, pattern := range patterns {
		// Handle glob patterns
		if strings.Contains(pattern, "*") {
			matched, err := filepath.Match(pattern, filepath.Base(path))
			if err == nil && matched {
				return true
			}
			// Also check if any parent directory matches
			parts := strings.Split(path, string(filepath.Separator))
			for _, part := range parts {
				matched, err := filepath.Match(pattern, part)
				if err == nil && matched {
					return true
				}
			}
		} else {
			// Exact match for directory or file name
			if filepath.Base(path) == pattern {
				return true
			}
			// Check if any parent directory matches
			parts := strings.Split(path, string(filepath.Separator))
			for _, part := range parts {
				if part == pattern {
					return true
				}
			}
		}
	}
	return false
}

// EnsureWorkspaceDir ensures the workspace directory exists in the container.
func EnsureWorkspaceDir(ctx context.Context, cli *client.Client, containerID, workspacePath string) error {
	if containerID == "" {
		return fmt.Errorf("container ID is required")
	}
	if workspacePath == "" {
		workspacePath = "/github/workspace"
	}

	cmd := []string{"mkdir", "-p", workspacePath}
	_, err := ExecCommand(ctx, cli, containerID, "/", cmd, nil)
	if err != nil {
		return fmt.Errorf("create workspace directory in container: %w", err)
	}

	return nil
}

// CleanupWorkspace removes temporary workspace files from container.
func CleanupWorkspace(ctx context.Context, cli *client.Client, containerID, workspacePath string) error {
	if containerID == "" {
		return fmt.Errorf("container ID is required")
	}
	if workspacePath == "" {
		workspacePath = "/github/workspace"
	}

	// Safety check: only remove paths under /github/workspace or /tmp/gacils
	if !strings.HasPrefix(workspacePath, "/github/workspace") && !strings.HasPrefix(workspacePath, "/tmp/gacils") {
		return fmt.Errorf("refusing to cleanup path outside allowed directories: %s", workspacePath)
	}

	cmd := []string{"rm", "-rf", workspacePath}
	_, err := ExecCommand(ctx, cli, containerID, "/", cmd, nil)
	if err != nil {
		return fmt.Errorf("remove workspace directory in container: %w", err)
	}

	return nil
}

// CopyHostToContainer copies files from host directory or file to container path.
func CopyHostToContainer(ctx context.Context, cli *client.Client, containerID, srcHostPath, dstContainerPath string) error {
	if containerID == "" {
		return fmt.Errorf("container ID is required")
	}
	if srcHostPath == "" {
		return fmt.Errorf("source host path is required")
	}
	if dstContainerPath == "" {
		return fmt.Errorf("destination container path is required")
	}

	// Ensure destination directory exists in container
	if err := EnsureWorkspaceDir(ctx, cli, containerID, dstContainerPath); err != nil {
		return fmt.Errorf("ensure container dir: %w", err)
	}

	info, err := os.Stat(srcHostPath)
	if err != nil {
		return fmt.Errorf("stat source path: %w", err)
	}

	var tarReader io.Reader
	if info.IsDir() {
		tarReader, err = TarDirectory(srcHostPath, nil)
		if err != nil {
			return fmt.Errorf("tar directory: %w", err)
		}
	} else {
		pr, pw := io.Pipe()
		tw := tar.NewWriter(pw)
		go func() {
			defer pw.Close()
			defer tw.Close()
			file, err := os.Open(srcHostPath)
			if err != nil {
				return
			}
			defer file.Close()
			hdr, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return
			}
			hdr.Name = filepath.Base(srcHostPath)
			if err := tw.WriteHeader(hdr); err != nil {
				return
			}
			_, _ = io.Copy(tw, file)
		}()
		tarReader = pr
	}

	return CopyToContainer(ctx, cli, containerID, dstContainerPath, tarReader)
}

// CopyContainerToHost copies files or directories from container path to host destination path.
func CopyContainerToHost(ctx context.Context, cli *client.Client, containerID, srcContainerPath, dstHostPath string) error {
	return CopyFromContainer(ctx, cli, containerID, srcContainerPath, dstHostPath)
}