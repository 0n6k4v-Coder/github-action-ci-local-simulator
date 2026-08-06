package dockerx

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// RunnerLabelToImage maps GitHub runner labels to Docker images.
var RunnerLabelToImage = map[string]string{
	"ubuntu-latest": "ubuntu:24.04",
	"ubuntu-24.04":  "ubuntu:24.04",
	"ubuntu-22.04":  "ubuntu:22.04",
}

// ResolveImage resolves a runs-on label to a Docker image.
func ResolveImage(runsOn string) (string, error) {
	img, ok := RunnerLabelToImage[runsOn]
	if !ok {
		return "", fmt.Errorf("unsupported runs-on: %q", runsOn)
	}
	return img, nil
}

// EnsureImage checks if an image exists locally, and pulls it if not (unless offline mode).
func EnsureImage(ctx context.Context, cli *client.Client, imageName string, offline bool, platform string) error {
	// Check if image exists locally
	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return fmt.Errorf("list images: %w", err)
	}

	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == imageName {
				return nil // Image already exists
			}
		}
	}

	// Image not found locally
	if offline {
		return fmt.Errorf("image %q not found locally and --offline mode prevents pulling", imageName)
	}

	// Image not found, pull it
	pullOptions := image.PullOptions{}
	if platform != "" {
		pullOptions.Platform = platform
	}
	reader, err := cli.ImagePull(ctx, imageName, pullOptions)
	if err != nil {
		return fmt.Errorf("pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Read the pull output to ensure completion
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			// Optionally log pull progress
			_ = string(buf[:n])
		}
		if err != nil {
			break
		}
	}

	// Verify the image now exists
	images, err = cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return fmt.Errorf("list images after pull: %w", err)
	}

	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == imageName {
				return nil
			}
		}
	}

	// Handle case where image name might have a different tag format (e.g., ubuntu:24.04 vs docker.io/library/ubuntu:24.04)
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if strings.HasSuffix(tag, ":"+strings.Split(imageName, ":")[1]) {
				return nil
			}
		}
	}

	return fmt.Errorf("image %s not found after pull", imageName)
}
