package cli

import (
	"fmt"

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
			if flags.Workflow == "" {
				return fmt.Errorf("workflow file is required (use -W flag)")
			}

			// Load workflow
			wf, err := workflow.LoadWorkflow(flags.Workflow)
			if err != nil {
				return fmt.Errorf("load workflow: %w", err)
			}

			// Normalize
			if err := workflow.Normalize(wf); err != nil {
				return fmt.Errorf("normalize workflow: %w", err)
			}

			// Validate
			if err := workflow.Validate(wf); err != nil {
				return fmt.Errorf("validate workflow: %w", err)
			}

			// Generate dry-run plan
			plan := workflow.GenerateDryRunPlan(wf, flags.Workflow)

			if flags.DryRun {
				printDryRunPlan(plan)
				return nil
			}

			// Not implemented yet
			fmt.Println("gacils run execution is not implemented yet")
			return nil
		},
	}

	BindRunFlags(cmd, flags)

	return cmd
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
				fmt.Printf(" timeout-minutes=%d", step.TimeoutMinutes)
			}
			fmt.Println()
		}
		fmt.Println()
	}
}