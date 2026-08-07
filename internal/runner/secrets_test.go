package runner

import (
	"reflect"
	"testing"
)

func TestMaskSecrets(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		secrets  []string
		expected string
	}{
		{
			name:     "empty output",
			output:   "",
			secrets:  []string{"secret"},
			expected: "",
		},
		{
			name:     "empty secrets list",
			output:   "My secret token is 12345",
			secrets:  []string{},
			expected: "My secret token is 12345",
		},
		{
			name:     "single secret masking",
			output:   "My token is super-secret-value-12345",
			secrets:  []string{"super-secret-value-12345"},
			expected: "My token is ***",
		},
		{
			name:     "multiple occurrences of secret",
			output:   "Token 12345 and token 12345 again",
			secrets:  []string{"12345"},
			expected: "Token *** and token *** again",
		},
		{
			name:     "multiple different secrets",
			output:   "API_KEY=key123 and TOKEN=tok456",
			secrets:  []string{"key123", "tok456"},
			expected: "API_KEY=*** and TOKEN=***",
		},
		{
			name:     "nested/overlapping secrets ordered by length",
			output:   "Value: super-secret-value",
			secrets:  DeduplicateAndSortSecrets([]string{"secret", "super-secret-value"}),
			expected: "Value: ***",
		},
		{
			name:     "empty secret string in list ignored",
			output:   "Normal string",
			secrets:  []string{"", "  "},
			expected: "Normal string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSecrets(tt.output, tt.secrets)
			if result != tt.expected {
				t.Errorf("MaskSecrets() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestIsSecretKey(t *testing.T) {
	secretKeys := []string{
		"MY_SECRET",
		"API_KEY",
		"ACCESS_TOKEN",
		"DB_PASSWORD",
		"secret_val",
		"key",
		"token",
		"password",
	}

	for _, k := range secretKeys {
		if !IsSecretKey(k) {
			t.Errorf("IsSecretKey(%q) = false, expected true", k)
		}
	}

	nonSecretKeys := []string{
		"PATH",
		"HOME",
		"SHELL",
		"WORK_DIR",
		"USER",
	}

	for _, k := range nonSecretKeys {
		if IsSecretKey(k) {
			t.Errorf("IsSecretKey(%q) = true, expected false", k)
		}
	}
}

func TestCollectSecrets(t *testing.T) {
	secretsMap := map[string]any{
		"MY_SECRET": "sec123",
	}

	envMap1 := map[string]string{
		"API_KEY":    "key456",
		"NORMAL_ENV": "normal",
	}

	envMap2 := map[string]string{
		"AUTH_TOKEN": "tok789",
	}

	secrets := CollectSecrets(secretsMap, envMap1, envMap2)

	// Result should contain sec123, key456, tok789 (ordered by length descending)
	if len(secrets) != 3 {
		t.Fatalf("expected 3 secrets, got %d: %v", len(secrets), secrets)
	}

	expectedSet := map[string]bool{
		"sec123": true,
		"key456": true,
		"tok789": true,
	}

	for _, s := range secrets {
		if !expectedSet[s] {
			t.Errorf("unexpected secret collected: %q", s)
		}
	}
}

func TestDeduplicateAndSortSecrets(t *testing.T) {
	input := []string{"short", "very-long-secret-key", "short", "", "medium-secret"}
	result := DeduplicateAndSortSecrets(input)

	expected := []string{"very-long-secret-key", "medium-secret", "short"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("DeduplicateAndSortSecrets() = %v, expected %v", result, expected)
	}
}
