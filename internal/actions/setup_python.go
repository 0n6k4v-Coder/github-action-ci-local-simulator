package actions

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/client"
)

// ExecuteSetupPython simulates actions/setup-python.
// Requires 'python-version' input in 'with'.
func ExecuteSetupPython(ctx context.Context, cli *client.Client, containerID, workingDir string, with map[string]any) (*ActionResult, error) {
	var pythonVersion string
	if with != nil {
		if val, ok := with["python-version"]; ok && val != nil {
			pythonVersion = strings.TrimSpace(fmt.Sprintf("%v", val))
		} else if val, ok := with["python_version"]; ok && val != nil {
			pythonVersion = strings.TrimSpace(fmt.Sprintf("%v", val))
		}
	}

	if pythonVersion == "" {
		return nil, NewActionValidationError("actions/setup-python: missing required input 'python-version'")
	}

	env := map[string]string{
		"python-location": "/usr/bin",
		"python-version":  pythonVersion,
	}

	stdout := fmt.Sprintf("actions/setup-python simulation: python-version=%s, python-location=/usr/bin\n", pythonVersion)

	return &ActionResult{
		Stdout: stdout,
		Env:    env,
	}, nil
}
