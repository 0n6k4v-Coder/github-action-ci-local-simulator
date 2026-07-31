package cli

import (
	"github.com/spf13/cobra"
)

// RunFlags holds all flags for the run command
type RunFlags struct {
	Workflow   string
	Job        string
	DryRun     bool
	Parallel   int
	CRLF       string
	Platform   string
	Offline    bool
	Verbose    bool
	NoColor    bool
	Strict     bool
}

// Global flags shared across commands
var (
	verbose   bool
	noColor   bool
	strict    bool
	offline   bool
)

// BindGlobalFlags binds global flags to the command
func BindGlobalFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	cmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	cmd.PersistentFlags().BoolVar(&strict, "strict", false, "Strict mode: fail on warnings")
	cmd.PersistentFlags().BoolVar(&offline, "offline", false, "Offline mode: do not pull images")
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

// GlobalFlags returns the global flags state
type GlobalFlags struct {
	Verbose bool
	NoColor bool
	Strict  bool
	Offline bool
}

func GetGlobalFlags() GlobalFlags {
	return GlobalFlags{
		Verbose: verbose,
		NoColor: noColor,
		Strict:  strict,
		Offline: offline,
	}
}