package commands

import (
	"github.com/spf13/cobra"
)

var (
	// Global flags
	ConfigPath    string
	EnableProfile bool
	ProfileDir    string
)

// NewRootCommand creates the root command
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "rph",
		Short: "Research Paper Helper - Convert research papers to student-friendly LaTeX reports",
		Long: `Research Paper Helper analyzes AI/ML research papers using Gemini AI
and generates comprehensive, student-friendly LaTeX reports with detailed
explanations of methodologies, breakthroughs, and results.`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&ConfigPath, "config", "c", "config/config.yaml", "config file path")
	rootCmd.PersistentFlags().BoolVar(&EnableProfile, "profile", false, "enable CPU and memory profiling")
	rootCmd.PersistentFlags().StringVar(&ProfileDir, "profile-dir", "./profiles", "directory for profile output")

	// Add subcommands
	rootCmd.AddCommand(
		NewProcessCommand(),
		NewListCommand(),
		NewStatusCommand(),
		NewCleanCommand(),
		NewCheckCommand(),
		NewRunCommand(),
		NewModelsCommand(),
		NewCacheCommand(),
		NewConfigureCommand(),
		NewChatCommand(),
		NewIndexCommand(),
		NewSearchCommand(),
	)

	return rootCmd
}
