package workflow

import (
	"fmt"
	"strings"
)

// ExpandMatrix expands a matrix configuration into a list of job instances.
// Each job instance has its own matrix context.
func ExpandMatrix(jobID string, job Job) ([]Job, error) {
	if job.Strategy == nil || job.Strategy.Matrix == nil {
		return []Job{job}, nil
	}

	matrix := job.Strategy.Matrix
	if len(matrix) == 0 {
		return []Job{job}, nil
	}

	// Extract include and exclude before computing Cartesian product
	var includeList []map[string]any
	var excludeList []map[string]any

	if inc, ok := matrix["include"].([]interface{}); ok {
		for _, item := range inc {
			if m, ok := item.(map[string]any); ok {
				includeList = append(includeList, m)
			}
		}
	}

	if exc, ok := matrix["exclude"].([]interface{}); ok {
		for _, item := range exc {
			if m, ok := item.(map[string]any); ok {
				excludeList = append(excludeList, m)
			}
		}
	}

	// Build the matrix dimensions (keys that are not include/exclude)
	dimensions := make(map[string][]any)
	for key, value := range matrix {
		if key == "include" || key == "exclude" {
			continue
		}
		switch v := value.(type) {
		case []interface{}:
			dimensions[key] = v
		case []string:
			vals := make([]any, len(v))
			for i, s := range v {
				vals[i] = s
			}
			dimensions[key] = vals
		default:
			return nil, fmt.Errorf("matrix key %q must be a list, got %T", key, value)
		}
	}

	// Compute Cartesian product
	combinations := computeCartesianProduct(dimensions)

	// Apply exclude rules
	combinations = applyExcludes(combinations, excludeList)

	// Apply include rules (add additional combinations)
	combinations = append(combinations, includeList...)

	// Create job instances
	var expandedJobs []Job
	for _, combo := range combinations {
		newJob := job
		// Deep copy steps
		newJob.Steps = make([]Step, len(job.Steps))
		copy(newJob.Steps, job.Steps)
		// Deep copy env
		if job.Env != nil {
			newJob.Env = make(map[string]any, len(job.Env))
			for k, v := range job.Env {
				newJob.Env[k] = v
			}
		}
		// Deep copy defaults
		if job.Defaults != nil {
			newJob.Defaults = &Defaults{
				Run: &RunDefaults{
					Shell:            job.Defaults.Run.Shell,
					WorkingDirectory: job.Defaults.Run.WorkingDirectory,
				},
			}
		}
		// Set matrix context
		newJob.Strategy = &Strategy{
			Matrix: map[string]any{
				"matrix": combo,
			},
		}
		// Create instance ID suffix
		suffix := buildMatrixSuffix(combo)
		newJob.instanceID = fmt.Sprintf("%s-%s", jobID, suffix)
		
		expandedJobs = append(expandedJobs, newJob)
	}

	return expandedJobs, nil
}

// computeCartesianProduct computes the Cartesian product of matrix dimensions.
func computeCartesianProduct(dimensions map[string][]any) []map[string]any {
	if len(dimensions) == 0 {
		return []map[string]any{{}}
	}

	keys := make([]string, 0, len(dimensions))
	for k := range dimensions {
		keys = append(keys, k)
	}

	var result []map[string]any
	var recurse func(int, map[string]any)
	recurse = func(idx int, current map[string]any) {
		if idx == len(keys) {
			// Copy the current combination
			combo := make(map[string]any, len(current))
			for k, v := range current {
				combo[k] = v
			}
			result = append(result, combo)
			return
		}

		key := keys[idx]
		for _, value := range dimensions[key] {
			current[key] = value
			recurse(idx+1, current)
		}
		delete(current, key)
	}

	recurse(0, make(map[string]any))
	return result
}

// applyExcludes removes combinations that match any exclude rule.
func applyExcludes(combinations []map[string]any, excludes []map[string]any) []map[string]any {
	if len(excludes) == 0 {
		return combinations
	}

	var result []map[string]any
	for _, combo := range combinations {
		excluded := false
		for _, exc := range excludes {
			if matchesExclude(combo, exc) {
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, combo)
		}
	}
	return result
}

// matchesExclude checks if a combination matches an exclude rule.
func matchesExclude(combo, exc map[string]any) bool {
	for key, value := range exc {
		if combo[key] != value {
			return false
		}
	}
	return true
}

// buildMatrixSuffix creates a readable suffix for the job instance ID.
func buildMatrixSuffix(combo map[string]any) string {
	var parts []string
	for k, v := range combo {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, ",")
}

// GetMatrixContext returns the matrix context for a job.
func (j *Job) GetMatrixContext() map[string]any {
	if j.Strategy == nil || j.Strategy.Matrix == nil {
		return nil
	}
	if matrix, ok := j.Strategy.Matrix["matrix"].(map[string]any); ok {
		return matrix
	}
	return nil
}

// HasMatrix returns true if the job has a matrix strategy.
func (j *Job) HasMatrix() bool {
	return j.Strategy != nil && j.Strategy.Matrix != nil
}