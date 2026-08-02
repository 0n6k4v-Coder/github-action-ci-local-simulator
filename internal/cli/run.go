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
		Use:   "run",
		Short: "Run GitHub Actions workflows locally",
		Long:  "Run GitHub Actions workflows locally using Docker.",
		RunE: func(cmd *cobra.Command, args []string) error {
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
			return executeWorkflows(cmd.Context(), workflows, paths)
		},
	}

	BindRunFlags(cmd, flags)

	return cmd
}

// executeWorkflows executes the loaded workflows using Docker.
func executeWorkflows(ctx context.Context, workflows []*workflow.Workflow, paths []string) error {
	// Create Docker client
	cli, err := dockerx.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("create docker client: %w", err)
	}
	defer cli.Close()

	// Create job runner
	jobRunner := runner.NewJobRunner(cli)

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
			return fmt.Errorf("expand matrix jobs: %w", err)
		}

		// Run each job in the workflow
		for jobID, job := range expandedWf.Jobs {
			fmt.Printf("  Job: %s\n", jobID)

			// Determine workspace path - use the directory containing the workflow file
			var jobWorkspacePath string
			if i < len(paths) {
				info, err := os.Stat(paths[i])
				if err != nil {
					return fmt.Errorf("stat workflow path: %w", err)
				}
				if info.IsDir() {
					// If path is a directory, use it directly
					jobWorkspacePath = paths[i]
				} else {
					// If path is a file, use its directory
					jobWorkspacePath = filepath.Dir(paths[i])
				}
			} else {
				jobWorkspacePath = "."
			}
			// Make it absolute
			jobWorkspacePath, err = filepath.Abs(jobWorkspacePath)
			if err != nil {
				return fmt.Errorf("resolve workspace path: %w", err)
			}

			result, err := jobRunner.RunJob(ctx, job, jobID, workflowEnv, wf.Defaults, wf, jobWorkspacePath)
			if err != nil {
				return fmt.Errorf("run job %s: %w", jobID, err)
			}

			// Print step outputs
			for j, stepResult := range result.Steps {
				if stepResult.Stdout != "" {
					fmt.Printf("    Step %d stdout:\n%s", j+1, stepResult.Stdout)
				}
				if stepResult.Stderr != "" {
					fmt.Printf("    Step %d stderr:\n%s", j+1, stepResult.Stderr)
				}
			}

			if result.ExitCode != 0 {
				if result.Error != nil {
					fmt.Printf("  Job %s failed: %v\n", jobID, result.Error)
				} else {
					fmt.Printf("  Job %s failed with exit code %d\n", jobID, result.ExitCode)
				}
				return fmt.Errorf("job %s failed with exit code %d", jobID, result.ExitCode)
			}

			fmt.Printf("  Job %s completed successfully\n", jobID)
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