package actions

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// Input validation tests
func TestExecuteSetupPython_ValidInput(t *testing.T) {
	with := map[string]any{
		"python-version": "3.12",
	}

	res, err := ExecuteSetupPython(context.Background(), nil, "container-123", "/workspace", with)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Env["python-version"] != "3.12" {
		t.Errorf("expected python-version=3.12 in env, got %s", res.Env["python-version"])
	}
	if res.Env["python-location"] != "/usr/bin" {
		t.Errorf("expected python-location=/usr/bin in env, got %s", res.Env["python-location"])
	}
}

func TestExecuteSetupPython_MissingVersion(t *testing.T) {
	with := map[string]any{}

	_, err := ExecuteSetupPython(context.Background(), nil, "container-123", "/workspace", with)
	if err == nil {
		t.Fatal("expected error for missing python-version")
	}
	var valErr *ActionValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected ActionValidationError, got %T: %v", err, err)
	}
}

func TestExecuteSetupPython_InvalidVersion(t *testing.T) {
	// ExecuteSetupPython allows any string input (or nil/empty check)
	with := map[string]any{
		"python-version": "",
	}

	_, err := ExecuteSetupPython(context.Background(), nil, "container-123", "/workspace", with)
	if err == nil {
		t.Fatal("expected error for empty python-version")
	}
}

// Version formats tests
func TestExecuteSetupPython_VersionFormats(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"minor version", "3.12", "3.12"},
		{"patch version", "3.12.0", "3.12.0"},
		{"wildcard version", "3.x", "3.x"},
		{"integer version", 3, "3"},
		{"float version", 3.12, "3.12"},
		{"underscore key", "3.11", "3.11"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var with map[string]any
			if tt.name == "underscore key" {
				with = map[string]any{"python_version": tt.input}
			} else {
				with = map[string]any{"python-version": tt.input}
			}

			res, err := ExecuteSetupPython(context.Background(), nil, "c-1", "/ws", with)
			if err != nil {
				t.Fatalf("unexpected error for format %s: %v", tt.name, err)
			}
			if res.Env["python-version"] != tt.expected {
				t.Errorf("expected python-version %s, got %s", tt.expected, res.Env["python-version"])
			}
		})
	}
}

// Environment output tests
func TestExecuteSetupPython_EnvOutput(t *testing.T) {
	with := map[string]any{
		"python-version": "3.10",
	}

	res, err := ExecuteSetupPython(context.Background(), nil, "c-1", "/ws", with)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Env == nil {
		t.Fatal("expected non-nil Env map")
	}
	if res.Env["python-version"] != "3.10" {
		t.Errorf("expected python-version 3.10, got %s", res.Env["python-version"])
	}
	if res.Env["python-location"] != "/usr/bin" {
		t.Errorf("expected python-location /usr/bin, got %s", res.Env["python-location"])
	}
	if !strings.Contains(res.Stdout, "python-version=3.10") {
		t.Errorf("expected stdout to mention python-version=3.10, got %s", res.Stdout)
	}
}

// Edge cases tests
func TestExecuteSetupPython_DefaultVersion(t *testing.T) {
	// Nil 'with' map
	_, err := ExecuteSetupPython(context.Background(), nil, "c-1", "/ws", nil)
	if err == nil {
		t.Fatal("expected error for nil with map")
	}
}

func TestExecuteSetupPython_MatrixVariable(t *testing.T) {
	// Simulated interpolated matrix variable input string
	with := map[string]any{
		"python-version": "3.9",
	}

	res, err := ExecuteSetupPython(context.Background(), nil, "c-1", "/ws", with)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Env["python-version"] != "3.9" {
		t.Errorf("expected 3.9, got %s", res.Env["python-version"])
	}
}
