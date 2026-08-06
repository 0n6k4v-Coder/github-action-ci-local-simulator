package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/dockerx"
	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/runner"
	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/workflow"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	flags := &RunFlags{}

	cmd := &cobra.Command{
		Use:          "run",
		Short:        "Run GitHub Actions workflows locally",
		Long:         "Run GitHub Actions workflows locally using Docker.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Warn about flags that are accepted but not yet implemented,
			// so users get actionable feedback instead of silent no-ops.
			if flags.Parallel != 0 {
				fmt.Fprintf(os.Stderr, "Warning: --parallel is not yet implemented and will be ignored.\n")
			}
			if flags.CRLF != "convert" {
				fmt.Fprintf(os.Stderr, "Warning: --crlf=%s is not yet implemented; defaulting to 'convert' behavior.\n", flags.CRLF)
			}
			if flags.Platform != "" {
				fmt.Fprintf(os.Stderr, "Warning: --platform is not yet implemented and will be ignored.\n")
			}
			if flags.Offline {
				fmt.Fprintf(os.Stderr, "Warning: --offline is not yet implemented; images will still be pulled if not cached locally.\n")
			}

			// Use default workflow directory if not specified
			workflowPath := flags.Workflow
			if workflowPath == "" {
				workflowPath = workflow.DefaultWorkflowDir()
			}

			// Check if path exists
			if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
				return fmt.Errorf("workflow path does not exist: %s", workflowPath)
			}

			// Load workflows (file or directory)
			workflows, paths, err := loadWorkflows(workflowPath)
			if err != nil {
				return fmt.Errorf("load workflows: %w", err)
			}

			// Process each workflow
			for i, wf := range workflows {
				// Normalize
				if err := workflow.Normalize(wf); err != nil {
					return fmt.Errorf("normalize workflow %s: %w", paths[i], err)
				}

				// Validate
				if err := workflow.Validate(wf); err != nil {
					// Check if it's a validation error with exit code
					if verr, ok := err.(*workflow.ValidationErrorWithCode); ok {
						fmt.Fprintf(os.Stderr, "Error: %v\n", verr)
						os.Exit(verr.ExitCode)
					}
					return fmt.Errorf("validate workflow %s: %w", paths[i], err)
				}
			}

			// Generate dry-run plan
			planSet := workflow.GenerateDryRunPlanSet(workflows, paths)

			if flags.DryRun {
				printDryRunPlanSet(planSet)
				return nil
			}

			// Execute workflows
			return executeWorkflows(cmd.Context(), workflows, paths, flags.Job)
		},
	}

	BindRunFlags(cmd, flags)

	return cmd
}

// executeWorkflows executes the loaded workflows using Docker.
func executeWorkflows(ctx context.Context, workflows []*workflow.Workflow, paths []string, jobFilter string) error {
	// Create Docker client
	cli, err := dockerx.CreateDockerClient()
	if err != nil {
		return fmt.Errorf("create docker client: %w\n  Hint: Check Docker daemon is running and network connectivity", err)
	}
	defer cli.Close()

	// Create job runner and workflow runner
	jobRunner := runner.NewJobRunner(cli)
	workflowRunner := runner.NewWorkflowRunner(jobRunner)

	// Process each workflow
	for i, wf := range workflows {
		fmt.Printf("Running workflow: %s (%s)\n", wf.Name, paths[i])

		// Build workflow-level environment
		workflowEnv := make(map[string]string)
		for k, v := range wf.Env {
			workflowEnv[k] = fmt.Sprintf("%v", v)
		}

		// Expand matrix jobs
		expandedWf, err := workflow.ExpandWorkflowJobs(wf)
		if err != nil {
			if verr, ok := err.(*workflow.ValidationErrorWithCode); ok {
				fmt.Fprintf(os.Stderr, "Error: %v\n", verr)
				os.Exit(verr.ExitCode)
			}
			return fmt.Errorf("expand matrix jobs: %w", err)
		}

		// Determine workspace path
		var jobWorkspacePath string
		if i < len(paths) {
			var err error
			jobWorkspacePath, err = runner.FindRepoRoot(paths[i])
			if err != nil {
				return fmt.Errorf("find repo root: %w", err)
			}
		} else {
			jobWorkspacePath = "."
			jobWorkspacePath, err = filepath.Abs(jobWorkspacePath)
			if err != nil {
				return fmt.Errorf("resolve workspace path: %w", err)
			}
		}

		result, err := workflowRunner.RunWorkflow(ctx, wf, expandedWf, jobWorkspacePath, workflowEnv, jobFilter)
		if err != nil {
			if verr, ok := err.(*workflow.ValidationErrorWithCode); ok {
				fmt.Fprintf(os.Stderr, "Error: %v\n", verr)
				os.Exit(verr.ExitCode)
			}
			if uerr, ok := err.(*runner.UnsupportedError); ok {
				fmt.Fprintf(os.Stderr, "Error: %v\n", uerr)
				os.Exit(uerr.ExitCode)
			}
			if ecErr, ok := err.(interface{ Code() int }); ok {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(ecErr.Code())
			}
			return fmt.Errorf("run workflow %s: %w", wf.Name, err)
		}

		// Print job results in execution order
		jobIDs := result.JobOrder
		if len(jobIDs) == 0 {
			for id := range result.Jobs {
				jobIDs = append(jobIDs, id)
			}
		}

		for _, jobID := range jobIDs {
			jobRes, ok := result.Jobs[jobID]
			if !ok || jobRes == nil {
				continue
			}

			if jobRes.Status == runner.StatusSkipped {
				fmt.Printf("  Job %s skipped\n", jobID)
				continue
			}

			fmt.Printf("  Job: %s\n", jobID)
			for j, stepResult := range jobRes.Steps {
				if stepResult.Stdout != "" {
					fmt.Printf("    Step %d stdout:\n%s", j+1, stepResult.Stdout)
				}
				if stepResult.Stderr != "" {
					fmt.Printf("    Step %d stderr:\n%s", j+1, stepResult.Stderr)
				}
			}

			if jobRes.ExitCode != 0 {
				if jobRes.Error != nil {
					fmt.Printf("  Job %s failed: %v\n", jobID, jobRes.Error)
				} else {
					fmt.Printf("  Job %s failed with exit code %d\n", jobID, jobRes.ExitCode)
				}
			} else {
				fmt.Printf("  Job %s completed successfully\n", jobID)
			}
		}

		if result.ExitCode != 0 {
			for _, jobID := range jobIDs {
				if jobRes, ok := result.Jobs[jobID]; ok && jobRes != nil {
					if jobRes.Error != nil {
						if ecErr, ok := jobRes.Error.(interface{ Code() int }); ok {
							fmt.Fprintf(os.Stderr, "Error: %v\n", jobRes.Error)
							os.Exit(ecErr.Code())
						}
					}
					if jobRes.ExitCode != 0 {
						os.Exit(jobRes.ExitCode)
					}
				}
			}
			return fmt.Errorf("workflow %s failed with exit code %d", wf.Name, result.ExitCode)
		}
	}

	return nil
}

// loadWorkflows loads workflows from a file or directory.
func loadWorkflows(path string) ([]*workflow.Workflow, []string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil, err
	}

	if info.IsDir() {
		workflows, err := workflow.LoadWorkflows(path)
		if err != nil {
			return nil, nil, err
		}
		// Build paths for each workflow
		paths := make([]string, len(workflows))
		for i := range workflows {
			// Find the matching file
			entries, _ := os.ReadDir(path)
			idx := 0
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				if filepath.Ext(name) == ".yml" || filepath.Ext(name) == ".yaml" {
					if idx == i {
						paths[i] = filepath.Join(path, name)
						break
					}
					idx++
				}
			}
		}
		return workflows, paths, nil
	}

	// It's a file
	wf, err := workflow.LoadWorkflow(path)
	if err != nil {
		return nil, nil, err
	}
	return []*workflow.Workflow{wf}, []string{path}, nil
}

func printDryRunPlanSet(planSet *workflow.DryRunPlanSet) {
	for i, plan := range planSet.Plans {
		if i > 0 {
			fmt.Println()
			fmt.Println("---")
			fmt.Println()
		}
		printDryRunPlan(plan)
	}
}

func printDryRunPlan(plan *workflow.DryRunPlan) {
	fmt.Printf("Workflow: %s\n", plan.WorkflowName)
	fmt.Printf("File: %s\n", plan.WorkflowPath)
	fmt.Println()

	for _, job := range plan.Jobs {
		fmt.Printf("Job: %s\n", job.ID)
		if job.Name != "" {
			fmt.Printf("  Name: %s\n", job.Name)
		}
		fmt.Printf("  Runs-on: %s\n", job.RunsOn)
		if job.Needs != "" {
			fmt.Printf("  Needs: %s\n", job.Needs)
		}
		fmt.Println("  Steps:")
		for _, step := range job.Steps {
			fmt.Printf("    %d. ", step.Index)
			if step.ID != "" {
				fmt.Printf("id=%s ", step.ID)
			}
			if step.Name != "" {
				fmt.Printf("name=%s ", step.Name)
			}
			if step.If != "" {
				fmt.Printf("if=%s ", step.If)
			}
			if step.Run != "" {
				fmt.Printf("run=%q", step.Run)
			}
			if step.Uses != "" {
				fmt.Printf("uses=%q", step.Uses)
			}
			if step.ContinueOnError {
				fmt.Printf(" continue-on-error=true")
			}
			if step.TimeoutMinutes > 0 {
				fmt.Printf(" timeout-minutes=%g", step.TimeoutMinutes)
			}
			fmt.Println()
		}
		fmt.Println()
	}
}