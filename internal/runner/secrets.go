package runner

import (
	"fmt"
	"sort"
	"strings"
)

// MaskSecrets replaces all occurrences of secret values in output with "***".
func MaskSecrets(output string, secrets []string) string {
	if output == "" || len(secrets) == 0 {
		return output
	}
	result := output
	for _, secret := range secrets {
		if secret != "" {
			result = strings.ReplaceAll(result, secret, "***")
		}
	}
	return result
}

// CollectSecrets collects secret strings from a secrets map and environment variable maps.
func CollectSecrets(secretsMap map[string]any, envMaps ...map[string]string) []string {
	secretSet := make(map[string]bool)

	// Collect from explicit secrets map
	if secretsMap != nil {
		for _, v := range secretsMap {
			strVal := strings.TrimSpace(fmt.Sprintf("%v", v))
			if strVal != "" {
				secretSet[strVal] = true
			}
		}
	}

	// Collect secret-like values from environment maps
	for _, envMap := range envMaps {
		for k, v := range envMap {
			if IsSecretKey(k) {
				strVal := strings.TrimSpace(v)
				if strVal != "" {
					secretSet[strVal] = true
				}
			}
		}
	}

	var result []string
	for secret := range secretSet {
		result = append(result, secret)
	}

	return DeduplicateAndSortSecrets(result)
}

// DeduplicateAndSortSecrets deduplicates and sorts secret strings by length descending.
func DeduplicateAndSortSecrets(secrets []string) []string {
	secretSet := make(map[string]bool)
	for _, s := range secrets {
		strVal := strings.TrimSpace(s)
		if strVal != "" {
			secretSet[strVal] = true
		}
	}

	var result []string
	for secret := range secretSet {
		result = append(result, secret)
	}

	// Sort secrets by length descending so longer secret phrases get replaced before sub-phrases
	sort.Slice(result, func(i, j int) bool {
		if len(result[i]) == len(result[j]) {
			return result[i] < result[j]
		}
		return len(result[i]) > len(result[j])
	})

	return result
}

// IsSecretKey returns true if key contains common secret keywords (case-insensitive).
func IsSecretKey(key string) bool {
	upper := strings.ToUpper(key)
	return strings.Contains(upper, "SECRET") ||
		strings.Contains(upper, "KEY") ||
		strings.Contains(upper, "TOKEN") ||
		strings.Contains(upper, "PASSWORD")
}
