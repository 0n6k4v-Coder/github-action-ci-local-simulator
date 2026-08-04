package runner

// Status represents the execution status of a step, job, or workflow.
type Status string

const (
	// StatusPending indicates the item has not started yet.
	StatusPending Status = "pending"
	// StatusRunning indicates the item is currently executing.
	StatusRunning Status = "running"
	// StatusSuccess indicates the item completed successfully.
	StatusSuccess Status = "success"
	// StatusFailure indicates the item failed.
	StatusFailure Status = "failure"
	// StatusCancelled indicates the item was cancelled.
	StatusCancelled Status = "cancelled"
	// StatusSkipped indicates the item was skipped.
	StatusSkipped Status = "skipped"
)

// StepResult represents the result of executing a step.
type StepResult struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Status     Status
	Outcome    Status // Raw outcome before continue-on-error
	Conclusion Status // Final conclusion after continue-on-error
	ContinueOnError bool
}

// JobResult represents the result of executing a job.
type JobResult struct {
	JobID    string
	Steps    []*StepResult
	ExitCode int
	Error    error
	Status   Status
	Outputs  map[string]string
}

// WorkflowResult represents the result of executing a workflow.
type WorkflowResult struct {
	Jobs     map[string]*JobResult
	JobOrder []string
	ExitCode int
	Error    error
	Status   Status
}

// UnsupportedError represents an error for unsupported features with exit code 3.
type UnsupportedError struct {
	Message  string
	ExitCode int
}

func (e *UnsupportedError) Error() string {
	return e.Message
}

func (e *UnsupportedError) Code() int {
	return e.ExitCode
}

// NewUnsupportedError creates a new UnsupportedError with exit code 3.
func NewUnsupportedError(msg string) *UnsupportedError {
	return &UnsupportedError{
		Message:  msg,
		ExitCode: 3,
	}
}