package actions

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/dockerx"
	"github.com/docker/docker/client"
)

// ExecuteCache simulates actions/cache.
func ExecuteCache(ctx context.Context, cli *client.Client, containerID, workingDir string, with map[string]any) (*ActionResult, error) {
	keyVal, ok := with["key"]
	if !ok || fmt.Sprintf("%v", keyVal) == "" {
		return nil, NewActionValidationError("action actions/cache requires input 'key'")
	}
	key := fmt.Sprintf("%v", keyVal)

	pathVal, ok := with["path"]
	if !ok || fmt.Sprintf("%v", pathVal) == "" {
		return nil, NewActionValidationError("action actions/cache requires input 'path'")
	}
	path := fmt.Sprintf("%v", pathVal)

	containerPath := resolveContainerPath(path, workingDir)
	hostCacheDir := GetCacheDir(ctx)

	if err := os.MkdirAll(hostCacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create host cache dir: %w", err)
	}

	keyHash := fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
	hostCachePath := filepath.Join(hostCacheDir, keyHash)

	// Check if host cache path exists
	cacheExists := false
	if info, err := os.Stat(hostCachePath); err == nil {
		if info.IsDir() {
			entries, _ := os.ReadDir(hostCachePath)
			if len(entries) > 0 {
				cacheExists = true
			}
		} else {
			cacheExists = true
		}
	}

	if cacheExists {
		// Restore cache to container
		if cli != nil && containerID != "" {
			if err := dockerx.CopyHostToContainer(ctx, cli, containerID, hostCachePath, containerPath); err != nil {
				return nil, fmt.Errorf("restore cache failed: %w", err)
			}
		}
		return &ActionResult{
			Stdout: fmt.Sprintf("actions/cache simulation: Cache restored from key: %s (cache-hit=true)\n", key),
			Env: map[string]string{
				"cache-hit": "true",
			},
		}, nil
	}

	// Cache miss: save container path to host cache directory if available
	_ = os.MkdirAll(hostCachePath, 0755)
	if cli != nil && containerID != "" {
		_ = dockerx.CopyContainerToHost(ctx, cli, containerID, containerPath, hostCachePath)
	}

	return &ActionResult{
		Stdout: fmt.Sprintf("actions/cache simulation: Cache not found for key: %s (cache-hit=false)\n", key),
		Env: map[string]string{
			"cache-hit": "false",
		},
	}, nil
}

// resolveContainerPath converts user specified path string to an absolute container path.
func resolveContainerPath(pathStr, workingDir string) string {
	pathStr = strings.TrimSpace(pathStr)
	if idx := strings.Index(pathStr, "\n"); idx != -1 {
		pathStr = strings.TrimSpace(pathStr[:idx])
	}
	if strings.HasPrefix(pathStr, "~/") {
		return "/root/" + pathStr[2:]
	}
	if pathStr == "~" {
		return "/root"
	}
	if !filepath.IsAbs(pathStr) {
		baseDir := workingDir
		if baseDir == "" {
			baseDir = "/github/workspace"
		}
		return filepath.Join(baseDir, pathStr)
	}
	return pathStr
}
