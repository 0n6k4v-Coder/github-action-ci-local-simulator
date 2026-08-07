package workflow

import (
	"fmt"
	"sort"
	"strings"
)

// ValidationError represents a validation error with location info.
type ValidationError struct {
	JobID   string
	Step    int
	Field   string
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

	if err := validateNeeds(wf); err != nil {
		return err
	}

	return nil
}

func validateJob(jobID string, job *Job) error {
	// Validate runs-on
	if job.RunsOn == nil {
		return &ValidationError{JobID: jobID, Message: "runs-on is required"}
	}

	// Validate strategy/matrix if present
	if job.Strategy != nil && job.Strategy.Matrix != nil {
		if err := validateMatrix(jobID, job.Strategy.Matrix); err != nil {
			return err
		}
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

	// Validate job-level if condition
	if job.If != "" {
		if err := validateIfCondition(job.If); err != nil {
			return &ValidationError{JobID: jobID, Message: fmt.Sprintf("invalid if condition: %v", err)}
		}
	}

	// Validate timeout-minutes
	if job.TimeoutMinutes < 0 {
		return &ValidationError{JobID: jobID, Message: "timeout-minutes cannot be negative"}
	}

	return nil
}

// ValidationErrorWithCode extends ValidationError with an exit code.
type ValidationErrorWithCode struct {
	ValidationError
	ExitCode int
}

func (e *ValidationErrorWithCode) Error() string {
	return e.ValidationError.Error()
}

// NewValidationErrorWithCode creates a validation error with a specific exit code.
func NewValidationErrorWithCode(jobID, message string, exitCode int) *ValidationErrorWithCode {
	return &ValidationErrorWithCode{
		ValidationError: ValidationError{JobID: jobID, Message: message},
		ExitCode:        exitCode,
	}
}

func validateMatrix(jobID string, matrix map[string]any) error {
	hasDimension := false
	dimensions := make(map[string][]any)
	var includeList []map[string]any
	var excludeList []map[string]any

	for key, value := range matrix {
		if key == "include" || key == "exclude" {
			// Validate include/exclude format
			if err := validateIncludeExclude(key, value); err != nil {
				return &ValidationError{JobID: jobID, Message: fmt.Sprintf("matrix.%s: %v", key, err)}
			}
			if key == "include" {
				if inc, ok := value.([]interface{}); ok {
					for _, item := range inc {
						if m, ok := item.(map[string]any); ok {
							includeList = append(includeList, m)
						}
					}
				}
			} else if key == "exclude" {
				if exc, ok := value.([]interface{}); ok {
					for _, item := range exc {
						if m, ok := item.(map[string]any); ok {
							excludeList = append(excludeList, m)
						}
					}
				}
			}
			continue
		}
		hasDimension = true
		// Validate matrix dimension values
		switch v := value.(type) {
		case []interface{}:
			if len(v) == 0 {
				return NewValidationErrorWithCode(jobID, fmt.Sprintf("matrix dimension %q cannot be empty", key), 2)
			}
			dimensions[key] = v
		case []string:
			if len(v) == 0 {
				return NewValidationErrorWithCode(jobID, fmt.Sprintf("matrix dimension %q cannot be empty", key), 2)
			}
			vals := make([]any, len(v))
			for i, s := range v {
				vals[i] = s
			}
			dimensions[key] = vals
		default:
			return &ValidationError{JobID: jobID, Message: fmt.Sprintf("matrix dimension %q must be a list, got %T", key, value)}
		}
	}
	// Check if matrix has no dimensions (only include/exclude)
	if !hasDimension {
		return NewValidationErrorWithCode(jobID, "matrix must have at least one dimension", 2)
	}

	// Check for zero job instances after exclude (simulate expansion for validation)
	if len(dimensions) > 0 {
		combinations := computeCartesianProduct(dimensions)
		combinations = applyExcludes(combinations, excludeList)
		// Don't apply includes for this check - just check if base + exclude produces zero
		if len(combinations) == 0 {
			return NewValidationErrorWithCode(jobID, "matrix expansion produced zero job instances after exclude", 2)
		}
	}

	return nil
}

func validateIncludeExclude(key string, value any) error {
	list, ok := value.([]interface{})
	if !ok {
		return fmt.Errorf("%s must be a list of objects", key)
	}
	for i, item := range list {
		if _, ok := item.(map[string]any); !ok {
			return fmt.Errorf("%s[%d] must be an object", key, i)
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

	// Validate step-level if condition
	if step.If != "" {
		if err := validateIfCondition(step.If); err != nil {
			return &ValidationError{JobID: jobID, Step: stepIndex, Message: fmt.Sprintf("invalid if condition: %v", err)}
		}
	}

	// Validate timeout-minutes
	if step.TimeoutMinutes < 0 {
		return &ValidationError{JobID: jobID, Step: stepIndex, Message: "timeout-minutes cannot be negative"}
	}

	return nil
}

// validateIfCondition does basic validation of if condition syntax.
func validateIfCondition(condition string) error {
	trimmed := strings.TrimSpace(condition)
	if trimmed == "" {
		return fmt.Errorf("empty condition")
	}

	// Allow both ${{ ... }} and bare expressions
	if strings.HasPrefix(trimmed, "${{") && strings.HasSuffix(trimmed, "}}") {
		inner := strings.TrimSpace(trimmed[3 : len(trimmed)-2])
		if inner == "" {
			return fmt.Errorf("empty expression in ${{ }}")
		}
	}

	// Check for supported status functions
	supportedFuncs := []string{"success()", "failure()", "always()", "cancelled()"}
	hasSupportedFunc := false
	for _, fn := range supportedFuncs {
		if strings.Contains(trimmed, fn) {
			hasSupportedFunc = true
			break
		}
	}

	// Also allow bare function calls without ${{ }}
	if !hasSupportedFunc && !strings.Contains(trimmed, "(") {
		// Might be a context reference like "github.ref" - allow for now
	}

	return nil
}

// DryRunPlan represents a dry-run execution plan for a single workflow.
type DryRunPlan struct {
	WorkflowName string
	WorkflowPath string
	Jobs         []DryRunJob
}

// DryRunPlanSet represents a dry-run execution plan for multiple workflows.
type DryRunPlanSet struct {
	Plans []*DryRunPlan
}

// DryRunJob represents a job in the dry-run plan.
type DryRunJob struct {
	ID     string
	Name   string
	RunsOn string
	Needs  string
	If     string
	Matrix map[string]any
	Steps  []DryRunStep
}

// DryRunStep represents a step in the dry-run plan.
type DryRunStep struct {
	Index           int
	ID              string
	Name            string
	Run             string
	Uses            string
	If              string
	ContinueOnError bool
	TimeoutMinutes  float64
}

// GenerateDryRunPlan generates a dry-run plan from the workflow.
func GenerateDryRunPlan(wf *Workflow, path string) *DryRunPlan {
	plan := &DryRunPlan{
		WorkflowName: wf.Name,
		WorkflowPath: path,
		Jobs:         make([]DryRunJob, 0, len(wf.Jobs)),
	}

	for id, job := range wf.Jobs {
		// Check if job has matrix
		if job.HasMatrix() {
			// Expand matrix jobs for dry-run
			expandedJobs, err := ExpandMatrix(id, job)
			if err != nil {
				// If expansion fails, just show the original job
				dryRunJob := DryRunJob{
					ID:     id,
					Name:   job.Name,
					RunsOn: job.GetRunsOnAsString(),
					Needs:  job.GetNeedsAsString(),
					If:     job.If,
					Matrix: job.GetMatrixContext(),
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
			} else {
				// Add each expanded job
				for _, expJob := range expandedJobs {
					dryRunJob := DryRunJob{
						ID:     expJob.InstanceID(),
						Name:   expJob.Name,
						RunsOn: expJob.GetRunsOnAsString(),
						Needs:  expJob.GetNeedsAsString(),
						If:     expJob.If,
						Matrix: expJob.GetMatrixContext(),
						Steps:  make([]DryRunStep, len(expJob.Steps)),
					}
					for i, step := range expJob.Steps {
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
			}
		} else {
			dryRunJob := DryRunJob{
				ID:     id,
				Name:   job.Name,
				RunsOn: job.GetRunsOnAsString(),
				Needs:  job.GetNeedsAsString(),
				If:     job.If,
				Matrix: nil,
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
	}

	return plan
}

// GenerateDryRunPlanSet generates a dry-run plan set from multiple workflows.
func GenerateDryRunPlanSet(workflows []*Workflow, paths []string) *DryRunPlanSet {
	planSet := &DryRunPlanSet{
		Plans: make([]*DryRunPlan, len(workflows)),
	}

	for i, wf := range workflows {
		planSet.Plans[i] = GenerateDryRunPlan(wf, paths[i])
	}

	return planSet
}

func validateNeeds(wf *Workflow) error {
	if wf == nil || len(wf.Jobs) == 0 {
		return nil
	}

	jobIDs := make([]string, 0, len(wf.Jobs))
	for id := range wf.Jobs {
		jobIDs = append(jobIDs, id)
	}
	sort.Strings(jobIDs)

	for _, jobID := range jobIDs {
		job := wf.Jobs[jobID]
		for _, needID := range job.GetNeedsList() {
			if _, exists := wf.Jobs[needID]; !exists {
				return NewValidationErrorWithCode(
					jobID,
					fmt.Sprintf("job '%s' depends on unknown job '%s'", jobID, needID),
					2,
				)
			}
		}
	}

	if cycle := findNeedsCycle(wf); cycle != "" {
		return NewValidationErrorWithCode(
			"",
			fmt.Sprintf("dependency cycle detected: %s", cycle),
			2,
		)
	}

	return nil
}

func findNeedsCycle(wf *Workflow) string {
	jobIDs := make([]string, 0, len(wf.Jobs))
	for id := range wf.Jobs {
		jobIDs = append(jobIDs, id)
	}
	sort.Strings(jobIDs)

	visited := make(map[string]bool)
	onPath := make(map[string]bool)
	var path []string
	var cycleStr string

	var dfs func(u string) bool
	dfs = func(u string) bool {
		visited[u] = true
		onPath[u] = true
		path = append(path, u)

		deps := wf.Jobs[u].GetNeedsList()
		sort.Strings(deps)

		for _, v := range deps {
			if onPath[v] {
				cycleStart := -1
				for i, node := range path {
					if node == v {
						cycleStart = i
						break
					}
				}
				if cycleStart != -1 {
					cycleNodes := append([]string{}, path[cycleStart:]...)
					cycleNodes = append(cycleNodes, v)
					cycleStr = strings.Join(cycleNodes, " -> ")
					return true
				}
			}
			if !visited[v] {
				if dfs(v) {
					return true
				}
			}
		}

		path = path[:len(path)-1]
		onPath[u] = false
		return false
	}

	for _, id := range jobIDs {
		if !visited[id] {
			if dfs(id) {
				return cycleStr
			}
		}
	}

	return ""
}
