package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/dockerx"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

func newCleanCmd() *cobra.Command {
	flags := &CleanFlags{}

	cmd := &cobra.Command{
		Use:          "clean",
		Short:        "Clean up Docker resources created by gacils",
		Long:         "Clean up Docker resources (containers, images, volumes, build cache) created by gacils.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeClean(cmd.Context(), flags)
		},
	}

	BindCleanFlags(cmd, flags)
	return cmd
}

// executeClean runs the cleanup logic based on the configured flags.
func executeClean(ctx context.Context, flags *CleanFlags) error {
	// Create Docker client
	cli, err := dockerx.CreateDockerClient()
	if err != nil {
		return fmt.Errorf("create docker client: %w\n  Hint: Check Docker daemon is running", err)
	}
	defer cli.Close()

	// Show mode
	if flags.DryRun {
		fmt.Println("=== Dry Run Mode ===")
		fmt.Println("Showing what would be removed (no action taken)")
		fmt.Println()
	} else {
		fmt.Println("=== Cleanup Mode ===")
		fmt.Println()
	}

	// Count resources
	containerCount, err := countContainers(ctx, cli)
	if err != nil {
		return fmt.Errorf("count containers: %w", err)
	}

	imageCount, err := countImages(ctx, cli)
	if err != nil {
		return fmt.Errorf("count images: %w", err)
	}

	volumeCount, err := countVolumes(ctx, cli)
	if err != nil {
		return fmt.Errorf("count volumes: %w", err)
	}

	// Show summary
	fmt.Printf("Found: %d containers, %d images, %d volumes\n", containerCount, imageCount, volumeCount)
	fmt.Println()

	if flags.DryRun {
		fmt.Println("[DRY RUN] Would remove:")
		fmt.Printf("  - %d containers\n", containerCount)
		if flags.Images || flags.PruneImages {
			fmt.Printf("  - %d images\n", imageCount)
		}
		if flags.All {
			fmt.Printf("  - %d volumes\n", volumeCount)
			fmt.Println("  - Build cache")
		}
		fmt.Println()
		fmt.Println("Run without --dry-run to actually remove these resources.")
		return nil
	}

	// Confirm unless --force
	if !flags.Force {
		if err := confirmCleanup(); err != nil {
			return err
		}
	}

	// Execute cleanup
	fmt.Println("Cleaning up...")

	// Step 1: Remove containers
	if flags.All {
		fmt.Println("Step 1/4: Removing all containers...")
		if err := removeAllContainers(ctx, cli); err != nil {
			return fmt.Errorf("remove containers: %w", err)
		}
	} else {
		fmt.Println("Step 1/4: Removing stopped containers...")
		if err := removeStoppedContainers(ctx, cli); err != nil {
			return fmt.Errorf("remove containers: %w", err)
		}
	}

	// Step 2: Remove images
	if flags.PruneImages {
		fmt.Println("Step 2/4: Removing ALL unused images (no filter)...")
		if err := pruneAllImages(ctx, cli); err != nil {
			return fmt.Errorf("prune images: %w", err)
		}
	} else if flags.Images {
		fmt.Println("Step 2/4: Removing unused images (24h filter)...")
		if err := pruneUnusedImages(ctx, cli); err != nil {
			return fmt.Errorf("prune images: %w", err)
		}
	} else {
		fmt.Println("Step 2/4: Removing dangling images...")
		if err := pruneDanglingImages(ctx, cli); err != nil {
			return fmt.Errorf("prune images: %w", err)
		}
	}

	// Step 3: Remove volumes (only with --all)
	if flags.All {
		fmt.Println("Step 3/4: Removing unused volumes...")
		if err := pruneVolumes(ctx, cli); err != nil {
			return fmt.Errorf("prune volumes: %w", err)
		}

		// Step 4: Clean build cache
		fmt.Println("Step 4/4: Cleaning build cache...")
		if err := cleanBuildCache(ctx, cli); err != nil {
			return fmt.Errorf("clean build cache: %w", err)
		}
	} else {
		fmt.Println("Step 3/4: Skipping volumes (use --all)")
		fmt.Println("Step 4/4: Skipping build cache (use --all)")
	}

	fmt.Println()
	fmt.Println("=== Cleanup Complete ===")
	return nil
}

// confirmCleanup prompts the user for confirmation before proceeding.
func confirmCleanup() error {
	fmt.Print("Proceed with cleanup? [y/N] ")
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		// Non-interactive environment — abort safely
		fmt.Println("Aborted (no input received).")
		return nil
	}
	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Aborted.")
		return nil
	}
	fmt.Println()
	return nil
}

// --- Container helpers ---

func countContainers(ctx context.Context, cli *client.Client) (int, error) {
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return 0, err
	}
	return len(containers), nil
}

func removeStoppedContainers(ctx context.Context, cli *client.Client) error {
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("status", "exited")),
	})
	if err != nil {
		return err
	}

	for _, c := range containers {
		if err := cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true}); err != nil {
			// Log but don't fail
			fmt.Printf("  Warning: failed to remove container %s: %v\n", c.ID[:12], err)
		}
	}
	return nil
}

func removeAllContainers(ctx context.Context, cli *client.Client) error {
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return err
	}

	for _, c := range containers {
		if err := cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true}); err != nil {
			fmt.Printf("  Warning: failed to remove container %s: %v\n", c.ID[:12], err)
		}
	}
	return nil
}

// --- Image helpers ---

func countImages(ctx context.Context, cli *client.Client) (int, error) {
	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return 0, err
	}
	return len(images), nil
}

func pruneDanglingImages(ctx context.Context, cli *client.Client) error {
	_, err := cli.ImagesPrune(ctx, filters.NewArgs(filters.Arg("dangling", "true")))
	return err
}

func pruneUnusedImages(ctx context.Context, cli *client.Client) error {
	// Remove images older than 24h
	_, err := cli.ImagesPrune(ctx, filters.NewArgs(
		filters.Arg("dangling", "false"),
		filters.Arg("until", "24h"),
	))
	return err
}

func pruneAllImages(ctx context.Context, cli *client.Client) error {
	// Remove all unused images (no filter)
	_, err := cli.ImagesPrune(ctx, filters.NewArgs(filters.Arg("dangling", "false")))
	return err
}

// --- Volume helpers ---

func countVolumes(ctx context.Context, cli *client.Client) (int, error) {
	volumes, err := cli.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return 0, err
	}
	return len(volumes.Volumes), nil
}

func pruneVolumes(ctx context.Context, cli *client.Client) error {
	_, err := cli.VolumesPrune(ctx, filters.NewArgs())
	return err
}

// --- Build cache helper ---

func cleanBuildCache(ctx context.Context, cli *client.Client) error {
	_, err := cli.BuildCachePrune(ctx, build.CachePruneOptions{All: true})
	return err
}
