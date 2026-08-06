package dockerx

import (
	"runtime"
	"strings"
)

// Platform represents a normalized Docker platform.
type Platform string

const (
	// PlatformLinuxAMD64 represents linux/amd64.
	PlatformLinuxAMD64 Platform = "linux/amd64"
	// PlatformLinuxARM64 represents linux/arm64.
	PlatformLinuxARM64 Platform = "linux/arm64"
	// PlatformLinuxARMv7 represents linux/arm/v7.
	PlatformLinuxARMv7 Platform = "linux/arm/v7"
	// PlatformLinux386 represents linux/386.
	PlatformLinux386 Platform = "linux/386"
	// PlatformLinuxPPC64le represents linux/ppc64le.
	PlatformLinuxPPC64le Platform = "linux/ppc64le"
	// PlatformLinuxS390x represents linux/s390x.
	PlatformLinuxS390x Platform = "linux/s390x"
)

// KnownPlatforms is the list of supported platforms.
var KnownPlatforms = []Platform{
	PlatformLinuxAMD64,
	PlatformLinuxARM64,
	PlatformLinuxARMv7,
	PlatformLinux386,
	PlatformLinuxPPC64le,
	PlatformLinuxS390x,
}

// NormalizePlatform normalizes a platform string to a standard format.
// Accepts formats like: "linux/amd64", "amd64", "x86_64", "arm64", "aarch64"
func NormalizePlatform(input string) Platform {
	input = strings.ToLower(strings.TrimSpace(input))

	// Already in standard format
	for _, p := range KnownPlatforms {
		if string(p) == input {
			return p
		}
	}

	// Handle common aliases
	switch input {
	case "amd64", "x86_64", "x64":
		return PlatformLinuxAMD64
	case "arm64", "aarch64", "armv8":
		return PlatformLinuxARM64
	case "arm", "armv7", "arm32":
		return PlatformLinuxARMv7
	case "386", "i386", "i686":
		return PlatformLinux386
	case "ppc64le", "powerpc64le":
		return PlatformLinuxPPC64le
	case "s390x":
		return PlatformLinuxS390x
	}

	// Default to host platform
	return HostPlatform()
}

// HostPlatform returns the current host platform.
func HostPlatform() Platform {
	switch runtime.GOARCH {
	case "amd64":
		return PlatformLinuxAMD64
	case "arm64":
		return PlatformLinuxARM64
	case "arm":
		return PlatformLinuxARMv7
	case "386":
		return PlatformLinux386
	case "ppc64le":
		return PlatformLinuxPPC64le
	case "s390x":
		return PlatformLinuxS390x
	default:
		return PlatformLinuxAMD64
	}
}

// ParsePlatform parses a platform string in "os/arch" format.
func ParsePlatform(s string) (os, arch string, err error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", "", nil // Return empty for invalid format
	}
	return parts[0], parts[1], nil
}

// IsValidPlatform checks if a platform string is valid and known.
func IsValidPlatform(s string) bool {
	p := NormalizePlatform(s)
	for _, known := range KnownPlatforms {
		if p == known {
			return true
		}
	}
	return false
}
