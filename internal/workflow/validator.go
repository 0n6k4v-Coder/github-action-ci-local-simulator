package workflow

import (
	"fmt"
	"strings"
)

// ValidationError represents a validation error with location info.
type ValidationError struct {
	JobID  string
	Step   int
	Field  string
	Message string
}

func (e *ValidationError) Error() string {
	if e.JobID != "" && e.Step > 0 {
		return fmt.Sprintf("job %q step %d: %s", e.JobID, e.Step, e.Message)
	}
	if e.JobID != "" {
		return fmt.Sprintf("job %q: %s", e.JobID, e.Message)
	}
	return e.Message
}

// Validate validates the workflow structure.
func Validate(wf *Workflow) error {
	if len(wf.Jobs) == 0 {
		return &ValidationError{Message: "workflow must have at least one job"}
	}

	jobIDs := make(map[string]bool)
	for id, job := range wf.Jobs {
		if id == "" {
			return &ValidationError{Message: "job ID cannot be empty"}
		}
		if jobIDs[id] {
			return &ValidationError{JobID: id, Message: "duplicate job ID"}
		}
		jobIDs[id] = true

		if err := validateJob(id, &job); err != nil {
			return err
		}
	}

	return nil
}

func validateJob(jobID string, job *Job) error {
	// Validate runs-on
	if job.RunsOn == nil {
		return &ValidationError{JobID: jobID, Message: "runs-on is required"}
	}

	// Validate steps
	if len(job.Steps) == 0 {
		return &ValidationError{JobID: jobID, Message: "job must have at least one step"}
	}

	for i, step := range job.Steps {
		if err := validateStep(jobID, i+1, &step); err != nil {
			return err
		}
	}

	return nil
}

func validateStep(jobID string, stepIndex int, step *Step) error {
	hasRun := strings.TrimSpace(step.Run) != ""
	hasUses := strings.TrimSpace(step.Uses) != ""

	if hasRun && hasUses {
		return &ValidationError{JobID: jobID, Step: stepIndex, Message: "step cannot have both 'run' and 'uses'"}
	}

	if !hasRun && !hasUses {
		return &ValidationError{JobID: jobID, Step: stepIndex, Message: "step must have either 'run' or 'uses'"}
	}

	return nil
}

// DryRunPlan represents a dry-run execution plan.
type DryRunPlan struct {
	WorkflowName string
	WorkflowPath string
	Jobs         []DryRunJob
}

// DryRunJob represents a job in the dry-run plan.
type DryRunJob struct {
	ID        string
	Name      string
	RunsOn    string
	Needs     string
	Steps     []DryRunStep
}

// DryRunStep represents a step in the dry-run plan.
type DryRunStep struct {
	Index    int
	ID       string
	Name     string
	Run      string
	Uses     string
	If       string
	ContinueOnError bool
	TimeoutMinutes int
}

// GenerateDryRunPlan generates a dry-run plan from the workflow.
func GenerateDryRunPlan(wf *Workflow, path string) *DryRunPlan {
	plan := &DryRunPlan{
		WorkflowName: wf.Name,
		WorkflowPath: path,
		Jobs:         make([]DryRunJob, 0, len(wf.Jobs)),
	}

	for id, job := range wf.Jobs {
		dryRunJob := DryRunJob{
			ID:     id,
			Name:   job.Name,
			RunsOn: job.GetRunsOnAsString(),
			Needs:  job.GetNeedsAsString(),
			Steps:  make([]DryRunStep, len(job.Steps)),
		}

		for i, step := range job.Steps {
			dryRunJob.Steps[i] = DryRunStep{
				Index:           i + 1,
				ID:              step.ID,
				Name:            step.Name,
				Run:             step.Run,
				Uses:            step.Uses,
				If:              step.If,
				ContinueOnError: step.ContinueOnError,
				TimeoutMinutes:  step.TimeoutMinutes,
			}
		}

		plan.Jobs = append(plan.Jobs, dryRunJob)
	}

	return plan
}