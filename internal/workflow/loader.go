package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// LoadWorkflow loads a workflow from a YAML file.
func LoadWorkflow(path string) (*Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read workflow file: %w", err)
	}

	var wf Workflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("parse workflow YAML: %w", err)
	}

	return &wf, nil
}

// LoadWorkflows loads all workflow files from a path (file or directory).
// If path is a file, loads that single file.
// If path is a directory, loads all .yml and .yaml files in it.
func LoadWorkflows(path string) ([]*Workflow, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
	}

	if info.IsDir() {
		return loadWorkflowsFromDir(path)
	}

	// It's a file
	wf, err := LoadWorkflow(path)
	if err != nil {
		return nil, err
	}
	return []*Workflow{wf}, nil
}

func loadWorkflowsFromDir(dirPath string) ([]*Workflow, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}

	var workflows []*Workflow
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
			filePath := filepath.Join(dirPath, name)
			wf, err := LoadWorkflow(filePath)
			if err != nil {
				return nil, fmt.Errorf("load %s: %w", filePath, err)
			}
			workflows = append(workflows, wf)
		}
	}

	if len(workflows) == 0 {
		return nil, fmt.Errorf("no workflow files (*.yml, *.yaml) found in %s", dirPath)
	}

	return workflows, nil
}

// DefaultWorkflowDir returns the default workflow directory.
func DefaultWorkflowDir() string {
	return ".github/workflows"
}