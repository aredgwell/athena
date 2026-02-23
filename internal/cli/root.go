package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is set at build time via ldflags.
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "athena",
	Short: "Athena CLI - AI-native repository lifecycle manager",
	Long: `Athena is a portable, schema-driven scaffolder and lifecycle manager
for AI-native repository workflows. It replaces shell-script-based
tooling with a single Go binary supporting safe upgrades, schema-driven
feature selection, and optional integrations.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		name := cmd.Name()
		if name == "version" || name == "completion" || name == "help" {
			return nil
		}
		return initRunContext(cmd)
	},
}

func init() {
	rootCmd.PersistentFlags().Bool("verbose", false, "User-facing progress detail")
	rootCmd.PersistentFlags().Bool("debug", false, "Structured internal traces")
	rootCmd.PersistentFlags().Bool("quiet", false, "Errors and final summary only")
	rootCmd.PersistentFlags().String("format", "text", "Output format: text or json")
	rootCmd.PersistentFlags().String("policy", "standard", "Policy level: strict, standard, or lenient")
	rootCmd.PersistentFlags().Duration("lock-timeout", 0, "Maximum wait for mutation lock acquisition")

	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print binary and schema version metadata",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "athena version %s\n", version)
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
