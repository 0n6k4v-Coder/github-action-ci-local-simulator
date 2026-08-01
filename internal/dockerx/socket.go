package dockerx

import (
	"os"
	"path/filepath"
	"runtime"
)

// DefaultDockerSocket is the default Docker socket path.
const DefaultDockerSocket = "/var/run/docker.sock"

// DetectDockerSocket attempts to find the Docker socket path.
// Checks: DOCKER_HOST env, default socket, Docker Desktop paths.
func DetectDockerSocket() string {
	// Check DOCKER_HOST environment variable
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		return host
	}

	// Check default socket
	if socketExists(DefaultDockerSocket) {
		return DefaultDockerSocket
	}

	// Check Docker Desktop for Mac socket
	macSocket := filepath.Join(os.Getenv("HOME"), ".docker", "run", "docker.sock")
	if socketExists(macSocket) {
		return "unix://" + macSocket
	}

	// Check Docker Desktop for Windows socket (in WSL)
	if runtime.GOOS == "linux" {
		wslSocket := "/mnt/wsl/docker.sock"
		if socketExists(wslSocket) {
			return "unix://" + wslSocket
		}
	}

	// Check podman socket
	podmanSocket := filepath.Join(os.Getenv("HOME"), ".local", "share", "containers", "podman", "machine", "qemu", "podman.sock")
	if socketExists(podmanSocket) {
		return "unix://" + podmanSocket
	}

	// Fall back to default
	return DefaultDockerSocket
}

// socketExists checks if a Unix socket file exists.
func socketExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSocket != 0
}

// ParseDockerHost parses a Docker host string into protocol and address.
func ParseDockerHost(host string) (protocol, address string) {
	if host == "" {
		return "unix", DefaultDockerSocket
	}

	// Handle unix:// prefix
	if len(host) >= 7 && host[:7] == "unix://" {
		return "unix", host[7:]
	}

	// Handle tcp:// prefix
	if len(host) >= 6 && host[:6] == "tcp://" {
		return "tcp", host[6:]
	}

	// Handle npipe:// prefix (Windows named pipe)
	if len(host) >= 7 && host[:7] == "npipe://" {
		return "npipe", host[7:]
	}

	// Default to unix socket
	return "unix", host
}

// IsLocalDockerHost checks if the Docker host is local (unix socket).
func IsLocalDockerHost(host string) bool {
	proto, _ := ParseDockerHost(host)
	return proto == "unix" || proto == "npipe"
}