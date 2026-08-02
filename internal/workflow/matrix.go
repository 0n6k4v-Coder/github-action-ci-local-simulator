package workflow

import (
	"fmt"
	"sort"
	"strings"
)

// MatrixExpansion represents a single expanded matrix combination
type MatrixExpansion map[string]any

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

	// Apply exclude rules (before include)
	combinations = applyExcludes(combinations, excludeList)

	// Apply include rules
	combinations = applyIncludes(combinations, includeList, getMatrixKeys(dimensions))

	// Deduplicate combinations
	combinations = deduplicateCombinations(combinations)

	// Check for zero combinations
	if len(combinations) == 0 {
		return nil, fmt.Errorf("matrix expansion produced zero job instances for job %q", jobID)
	}

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

	// Get keys in stable order
	keys := make([]string, 0, len(dimensions))
	for k := range dimensions {
		keys = append(keys, k)
	}
	sort.Strings(keys)

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

// getMatrixKeys returns the base matrix keys (dimension keys) in stable order.
func getMatrixKeys(dimensions map[string][]any) []string {
	keys := make([]string, 0, len(dimensions))
	for k := range dimensions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// applyExcludes removes combinations that match any exclude rule.
// An exclude rule matches when all key/value pairs in the exclude object match the combination.
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
// All key/value pairs in the exclude rule must match the combination.
func matchesExclude(combo, exc map[string]any) bool {
	for key, value := range exc {
		comboVal, ok := combo[key]
		if !ok {
			return false
		}
		if !valuesEqual(comboVal, value) {
			return false
		}
	}
	return true
}

// applyIncludes applies include rules to the combinations.
// Include processing is sequential - later includes see results of earlier ones.
func applyIncludes(combinations []map[string]any, includes []map[string]any, baseKeys []string) []map[string]any {
	for _, inc := range includes {
		combinations = applySingleInclude(combinations, inc, baseKeys)
	}
	return combinations
}

// applySingleInclude applies a single include object.
func applySingleInclude(combinations []map[string]any, include map[string]any, baseKeys []string) []map[string]any {
	// Check if include has any base matrix keys
	hasBaseKey := false
	for key := range include {
		for _, baseKey := range baseKeys {
			if key == baseKey {
				hasBaseKey = true
				break
			}
		}
		if hasBaseKey {
			break
		}
	}

	if !hasBaseKey {
		// Include has only new keys - add to all combinations
		for i := range combinations {
			for k, v := range include {
				combinations[i][k] = v
			}
		}
		return combinations
	}

	// Include has base keys - find matching combinations
	matched := false
	for i := range combinations {
		if matchesInclude(combinations[i], include, baseKeys) {
			// Merge include into matching combination
			for k, v := range include {
				combinations[i][k] = v
			}
			matched = true
		}
	}

	if !matched {
		// No matching combination - create new one from include
		combinations = append(combinations, include)
	}

	return combinations
}

// matchesInclude checks if a combination matches an include rule on base keys.
// All base keys specified in the include must match the combination.
func matchesInclude(combo, include map[string]any, baseKeys []string) bool {
	for key, value := range include {
		// Check if this is a base key
		isBaseKey := false
		for _, baseKey := range baseKeys {
			if key == baseKey {
				isBaseKey = true
				break
			}
		}
		if !isBaseKey {
			continue
		}

		comboVal, ok := combo[key]
		if !ok {
			return false
		}
		if !valuesEqual(comboVal, value) {
			return false
		}
	}
	return true
}

// valuesEqual compares two values for equality, handling different types.
func valuesEqual(a, b any) bool {
	// Handle string comparison
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return aStr == bStr
}

// deduplicateCombinations removes duplicate combinations.
// Two combinations are duplicates if all key/value pairs are equal.
// Preserves first occurrence order.
func deduplicateCombinations(combinations []map[string]any) []map[string]any {
	seen := make(map[string]bool)
	var result []map[string]any

	for _, combo := range combinations {
		key := comboToString(combo)
		if !seen[key] {
			seen[key] = true
			result = append(result, combo)
		}
	}
	return result
}

// comboToString creates a string representation of a combination for deduplication.
// Keys are sorted for consistent representation.
func comboToString(combo map[string]any) string {
	if combo == nil {
		return ""
	}
	keys := make([]string, 0, len(combo))
	for k := range combo {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, combo[k]))
	}
	return strings.Join(parts, ",")
}

// buildMatrixSuffix creates a readable suffix for the job instance ID.
func buildMatrixSuffix(combo map[string]any) string {
	var parts []string
	keys := make([]string, 0, len(combo))
	for k := range combo {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, combo[k]))
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