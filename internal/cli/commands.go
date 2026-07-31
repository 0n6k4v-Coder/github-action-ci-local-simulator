package cli

import (
	"fmt"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/app"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print version information including version, commit, and build date.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("gacils version %s\n", app.Version)
			fmt.Printf("commit: %s\n", app.Commit)
			fmt.Printf("date: %s\n", app.Date)
			return nil
		},
	}
	return cmd
}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workflows found in .github/workflows",
		Long:  "List all workflow files found in the .github/workflows directory.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("gacils list is not implemented yet")
			return nil
		},
	}

	cmd.Flags().StringSliceP("workflow", "W", nil, "Workflow file or directory (default: .github/workflows)")
	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new workflow file",
		Long:  "Initialize a new GitHub Actions workflow file in .github/workflows.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("gacils init is not implemented yet")
			return nil
		},
	}

	cmd.Flags().Bool("force", false, "Overwrite existing files")
	cmd.Flags().String("python", "3.12", "Python version for generated workflow")
	cmd.Flags().String("working-directory", ".", "Working directory for generated workflow")

	return cmd
}

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check local environment readiness",
		Long:  "Check local environment readiness for running workflows (Docker, Git, etc.).",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("gacils doctor is not implemented yet")
			return nil
		},
	}

	cmd.Flags().Bool("fix", false, "Attempt to fix issues automatically")
	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}

func newCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove gacils local cache, logs, containers, and volumes",
		Long:  "Remove gacils local cache, logs, containers, and volumes.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("gacils clean is not implemented yet")
			return nil
		},
	}

	cmd.Flags().Bool("logs", false, "Remove logs")
	cmd.Flags().Bool("cache", false, "Remove cache")
	cmd.Flags().Bool("containers", false, "Remove gacils containers")
	cmd.Flags().Bool("volumes", false, "Remove gacils Docker volumes")
	cmd.Flags().Bool("all", false, "Remove logs, cache, containers, and volumes")
	cmd.Flags().Duration("older-than", 0, "Remove logs older than duration")
	cmd.Flags().Bool("force", false, "Do not ask for confirmation")
	cmd.Flags().Bool("include-config", false, "Also remove config file")

	return cmd
}

func newSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Setup local caches, for example Python tool caches",
		Long:  "Setup local caches, for example Python tool caches.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("gacils setup is not implemented yet")
			return nil
		},
	}

	cmd.AddCommand(newSetupPythonCmd())

	return cmd
}

func newSetupPythonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "python <version>",
		Short: "Setup Python tool cache",
		Long:  "Setup Python tool cache for the specified version.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("gacils setup python is not implemented yet")
			return nil
		},
	}

	cmd.Flags().Bool("force", false, "Force re-setup even if already present")
	cmd.Flags().Bool("no-cache", false, "Do not use cached images")

	return cmd
}