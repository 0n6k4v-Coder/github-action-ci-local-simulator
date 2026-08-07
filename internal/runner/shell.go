package runner

import (
	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
)

// ShellResolver provides shell resolution with precedence logic.
type ShellResolver struct {
	WorkflowDefaults *workflow.Defaults
	JobDefaults      *workflow.Defaults
}

// NewShellResolver creates a new shell resolver.
func NewShellResolver(workflowDefaults, jobDefaults *workflow.Defaults) *ShellResolver {
	return &ShellResolver{
		WorkflowDefaults: workflowDefaults,
		JobDefaults:      jobDefaults,
	}
}

// Resolve returns the shell to use for a step, with precedence:
// step.shell > job.defaults.run.shell > workflow.defaults.run.shell > "bash"
func (sr *ShellResolver) Resolve(step workflow.Step) string {
	if step.Shell != "" {
		return step.Shell
	}
	if sr.JobDefaults != nil && sr.JobDefaults.Run != nil && sr.JobDefaults.Run.Shell != "" {
		return sr.JobDefaults.Run.Shell
	}
	if sr.WorkflowDefaults != nil && sr.WorkflowDefaults.Run != nil && sr.WorkflowDefaults.Run.Shell != "" {
		return sr.WorkflowDefaults.Run.Shell
	}
	return "bash"
}

// BuildCommand builds the command slice for a given shell and run command.
func BuildCommand(runCommand, shell string) []string {
	switch shell {
	case "bash", "":
		return []string{"bash", "-e", "-c", runCommand}
	case "sh":
		return []string{"sh", "-e", "-c", runCommand}
	default:
		// Fallback to bash
		return []string{"bash", "-e", "-c", runCommand}
	}
}
