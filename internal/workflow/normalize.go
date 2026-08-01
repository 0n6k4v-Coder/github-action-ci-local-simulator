package workflow

import (
	"fmt"
	"strings"
)

// Normalize normalizes the workflow after loading to handle YAML quirks.
func Normalize(wf *Workflow) error {
	// Normalize jobs
	for id, job := range wf.Jobs {
		// Normalize runs-on
		if job.RunsOn != nil {
			normalized, err := normalizeStringOrList(job.RunsOn, "runs-on")
			if err != nil {
				return fmt.Errorf("job %q: %w", id, err)
			}
			job.RunsOn = normalized
		}

		// Normalize needs
		if job.Needs != nil {
			normalized, err := normalizeStringOrList(job.Needs, "needs")
			if err != nil {
				return fmt.Errorf("job %q: %w", id, err)
			}
			job.Needs = normalized
		}

		// Normalize steps
		for i := range job.Steps {
			if err := normalizeStep(&job.Steps[i]); err != nil {
				return fmt.Errorf("job %q step %d: %w", id, i+1, err)
			}
		}

		wf.Jobs[id] = job
	}

	return nil
}

// normalizeStringOrList normalizes a value that can be either a string or a list.
func normalizeStringOrList(value interface{}, fieldName string) ([]string, error) {
	switch v := value.(type) {
	case string:
		if v == "" {
			return nil, fmt.Errorf("%s cannot be empty", fieldName)
		}
		return []string{v}, nil
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			str, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%s[%d] must be a string", fieldName, i)
			}
			if str == "" {
				return nil, fmt.Errorf("%s[%d] cannot be empty", fieldName, i)
			}
			result[i] = str
		}
		return result, nil
	case []string:
		for i, str := range v {
			if str == "" {
				return nil, fmt.Errorf("%s[%d] cannot be empty", fieldName, i)
			}
		}
		return v, nil
	default:
		return nil, fmt.Errorf("%s must be a string or list of strings, got %T", fieldName, value)
	}
}

// normalizeStep normalizes a step to handle YAML quirks.
func normalizeStep(step *Step) error {
	// Step validation will be done in validator
	return nil
}

// GetRunsOnAsString returns the runs-on as a comma-separated string for display.
func (j *Job) GetRunsOnAsString() string {
	if j.RunsOn == nil {
		return ""
	}
	switch v := j.RunsOn.(type) {
	case []string:
		return strings.Join(v, ", ")
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetNeedsAsString returns the needs as a comma-separated string for display.
func (j *Job) GetNeedsAsString() string {
	if j.Needs == nil {
		return ""
	}
	switch v := j.Needs.(type) {
	case []string:
		return strings.Join(v, ", ")
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// SetInstanceID sets the job instance ID.
func (j *Job) SetInstanceID(id string) {
	j.instanceID = id
}

// InstanceID returns the job instance ID.
func (j *Job) InstanceID() string {
	return j.instanceID
}

// SetGitHubEnvPath sets the GITHUB_ENV file path.
func (j *Job) SetGitHubEnvPath(path string) {
	j.githubEnvPath = path
}

// GitHubEnvPath returns the GITHUB_ENV file path.
func (j *Job) GitHubEnvPath() string {
	return j.githubEnvPath
}

// SetGitHubPathPath sets the GITHUB_PATH file path.
func (j *Job) SetGitHubPathPath(path string) {
	j.githubPathPath = path
}

// GitHubPathPath returns the GITHUB_PATH file path.
func (j *Job) GitHubPathPath() string {
	return j.githubPathPath
}