package cli

import (
	"github.com/spf13/cobra"
)

// RunFlags holds all flags for the run command
type RunFlags struct {
	Workflow string
	Job      string
	DryRun   bool
	Parallel int
	CRLF     string
	Platform string
	Offline  bool
}

// CleanFlags represents the clean command flags
type CleanFlags struct {
	All         bool // Remove all resources (containers, volumes, cache)
	Images      bool // Also remove unused images (with 24h safety filter)
	PruneImages bool // Remove ALL images (no filter, aggressive)
	DryRun      bool // Show what would be removed without action
	Force       bool // Skip confirmation prompt
}

// BindRunFlags binds run-specific flags to the command
func BindRunFlags(cmd *cobra.Command, flags *RunFlags) {
	cmd.Flags().StringVarP(&flags.Workflow, "workflow", "W", "", "Workflow file or directory (default: .github/workflows)")
	cmd.Flags().StringVarP(&flags.Job, "job", "j", "", "Run specific job only")
	cmd.Flags().BoolVar(&flags.DryRun, "dry-run", false, "Print execution plan and exit")
	cmd.Flags().IntVarP(&flags.Parallel, "parallel", "p", 0, "Number of independent jobs to run concurrently (default: 0 = all)")
	cmd.Flags().StringVar(&flags.CRLF, "crlf", "convert", "CRLF handling mode: convert | preserve | error (default: convert)")
	cmd.Flags().StringVar(&flags.Platform, "platform", "", "Force container platform (e.g., linux/amd64)")
	cmd.Flags().BoolVar(&flags.Offline, "offline", false, "Offline mode: do not pull images")
}

// BindCleanFlags binds clean command flags
func BindCleanFlags(cmd *cobra.Command, flags *CleanFlags) {
	cmd.Flags().BoolVar(&flags.All, "all", false, "Remove all resources (containers, volumes, build cache)")
	cmd.Flags().BoolVar(&flags.Images, "images", false, "Also remove unused images (with 24h safety filter)")
	cmd.Flags().BoolVar(&flags.PruneImages, "prune-images", false, "Remove ALL images (no filter, aggressive)")
	cmd.Flags().BoolVar(&flags.DryRun, "dry-run", false, "Show what would be removed without taking action")
	cmd.Flags().BoolVar(&flags.Force, "force", false, "Skip confirmation prompt")
}
