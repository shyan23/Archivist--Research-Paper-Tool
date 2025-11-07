package commands

import (
	"archivist/internal/analyzer"
	"archivist/internal/app"
	"archivist/internal/ui"
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewModelsCommand creates the models command
func NewModelsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "models",
		Short: "List available Gemini AI models",
		Long:  "Query the Gemini API to list all available models and find the best thinking model",
		Run:   runModels,
	}
}

func runModels(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(ConfigPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	ui.PrintStage("Querying Gemini API", "Finding available models")

	// Create analyzer to query models
	analyzer, err := analyzer.NewAnalyzer(config)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to create analyzer: %v", err))
		os.Exit(1)
	}
	defer analyzer.Close()

	ctx := context.Background()

	// List all available models
	ui.PrintInfo("Fetching list of available models...")
	models, err := analyzer.GetClient().ListAvailableModels(ctx)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to list models: %v", err))
		os.Exit(1)
	}

	ui.ColorBold.Println("\n═══════════════════════════════════════════════════════════════")
	ui.ColorBold.Printf("           AVAILABLE GEMINI MODELS (%d)                    \n", len(models))
	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	for i, model := range models {
		ui.ColorInfo.Printf("  %d. %s\n", i+1, model)
	}

	// Find best thinking model
	fmt.Println()
	ui.PrintInfo("Finding best thinking model...")
	thinkingModel, err := analyzer.GetClient().FindThinkingModel(ctx)
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Could not find thinking model: %v", err))
	} else {
		fmt.Println()
		ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
		ui.ColorSuccess.Printf("  RECOMMENDED THINKING MODEL: %s\n", thinkingModel)
		ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
		fmt.Println()
		ui.PrintInfo("To use this model, update your config/config.yaml:")
		ui.ColorSubtle.Printf("  gemini:\n")
		ui.ColorSubtle.Printf("    agentic:\n")
		ui.ColorSubtle.Printf("      stages:\n")
		ui.ColorSubtle.Printf("        methodology_analysis:\n")
		ui.ColorSubtle.Printf("          model: \"%s\"\n", thinkingModel)
		fmt.Println()
	}
}
