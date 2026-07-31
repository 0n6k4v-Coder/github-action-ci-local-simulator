package cli

import (
	"fmt"
	"os"

	"github.com/0n6k4v-Coder/github-action-ci-local-simulator/internal/app"
	"github.com/spf13/cobra"
)

var globalFlags struct {
	configFile      string
	workingDir      string
	verbose         bool
	noColor         bool
	strict          bool
	jsonOutput      bool
	dockerHost      string
	dockerContext   string
	dockerTLSVerify string
	dockerCertPath  string
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "gacils",
		Short:   "Run GitHub Actions workflows locally using Docker",
		Long: `gacils is a personal open-source tool for running GitHub Actions workflows
locally using Docker. It helps you test workflows before pushing,
debug CI failures locally, and reduce GitHub Actions minute usage.

Supported features (MVP):
- Run .github/workflows/*.yml locally
- Matrix expansion, job dependencies (needs)
- actions/checkout, actions/setup-python
- Docker-based job execution with shared filesystem
- Environment variable precedence, GITHUB_ENV/GITHUB_OUTPUT/GITHUB_PATH
- Expression evaluation with expr-lang

Unsupported features fail clearly with exit code 3.`,
		Version: app.Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if globalFlags.noColor {
				os.Setenv("NO_COLOR", "1")
			}
			if globalFlags.workingDir != "." {
				if err := os.Chdir(globalFlags.workingDir); err != nil {
					return fmt.Errorf("change directory: %w", err)
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVar(&globalFlags.configFile, "config", "", "Config file path (default: ~/.gacils/config.yaml)")
	cmd.PersistentFlags().StringVarP(&globalFlags.workingDir, "working-directory", "w", ".", "Working directory (default: current directory)")
	cmd.PersistentFlags().BoolVarP(&globalFlags.verbose, "verbose", "v", false, "Verbose output")
	cmd.PersistentFlags().BoolVar(&globalFlags.noColor, "no-color", false, "Disable colored output")
	cmd.PersistentFlags().BoolVar(&globalFlags.strict, "strict", false, "Strict mode: treat warnings as errors")
	cmd.PersistentFlags().BoolVar(&globalFlags.jsonOutput, "json", false, "Output as JSON")
	cmd.PersistentFlags().StringVar(&globalFlags.dockerHost, "docker-host", "", "Docker host (overrides DOCKER_HOST)")
	cmd.PersistentFlags().StringVar(&globalFlags.dockerContext, "docker-context", "", "Docker context (overrides DOCKER_CONTEXT)")
	cmd.PersistentFlags().StringVar(&globalFlags.dockerTLSVerify, "docker-tls-verify", "", "Docker TLS verify (overrides DOCKER_TLS_VERIFY)")
	cmd.PersistentFlags().StringVar(&globalFlags.dockerCertPath, "docker-cert-path", "", "Docker cert path (overrides DOCKER_CERT_PATH)")

	// Add subcommands
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newDoctorCmd())
	cmd.AddCommand(newCleanCmd())
	cmd.AddCommand(newSetupCmd())

	return cmd
}

func Execute() error {
	rootCmd := newRootCmd()
	return rootCmd.Execute()
}