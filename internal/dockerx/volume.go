package dockerx

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

// VolumeManager manages Docker volumes for gacils.
type VolumeManager struct {
	cli *client.Client
}

// NewVolumeManager creates a new volume manager.
func NewVolumeManager(cli *client.Client) *VolumeManager {
	return &VolumeManager{cli: cli}
}

// EnsureVolume creates a volume if it doesn't exist.
func (vm *VolumeManager) EnsureVolume(ctx context.Context, name string, labels map[string]string) error {
	_, err := vm.cli.VolumeInspect(ctx, name)
	if err == nil {
		return nil // Volume already exists
	}

	// Create volume
	_, err = vm.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:   name,
		Labels: labels,
	})
	if err != nil {
		return fmt.Errorf("create volume %s: %w", name, err)
	}
	return nil
}

// RemoveVolume removes a volume by name.
func (vm *VolumeManager) RemoveVolume(ctx context.Context, name string, force bool) error {
	err := vm.cli.VolumeRemove(ctx, name, force)
	if err != nil {
		return fmt.Errorf("remove volume %s: %w", name, err)
	}
	return nil
}

// VolumeExists checks if a volume exists.
func (vm *VolumeManager) VolumeExists(ctx context.Context, name string) (bool, error) {
	_, err := vm.cli.VolumeInspect(ctx, name)
	if err == nil {
		return true, nil
	}
	// Check if it's a "not found" error
	return false, nil
}

// CleanupVolumes removes volumes with a specific label.
func (vm *VolumeManager) CleanupVolumes(ctx context.Context, labelKey, labelValue string) error {
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", labelKey+"="+labelValue)

	volumes, err := vm.cli.VolumeList(ctx, volume.ListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		return fmt.Errorf("list volumes: %w", err)
	}

	for _, vol := range volumes.Volumes {
		if err := vm.RemoveVolume(ctx, vol.Name, true); err != nil {
			// Log but continue
			fmt.Printf("Warning: failed to remove volume %s: %v\n", vol.Name, err)
		}
	}
	return nil
}

// GacilsVolumeName returns the standard gacils volume name.
const GacilsVolumeName = "gacils-tmp"

// EnsureGacilsVolume ensures the gacils temporary volume exists.
func (vm *VolumeManager) EnsureGacilsVolume(ctx context.Context) error {
	return vm.EnsureVolume(ctx, GacilsVolumeName, map[string]string{
		"gacils.managed": "true",
	})
}
