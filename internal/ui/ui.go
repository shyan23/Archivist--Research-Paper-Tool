package ui

import (
	"archivist/internal/app"
	"fmt"
	"time"

	"github.com/common-nighthawk/go-figure"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/schollz/progressbar/v3"
)

var (
	// Colors
	ColorSuccess = color.New(color.FgGreen, color.Bold)
	ColorError   = color.New(color.FgRed, color.Bold)
	ColorWarning = color.New(color.FgYellow, color.Bold)
	ColorInfo    = color.New(color.FgCyan, color.Bold)
	ColorTitle   = color.New(color.FgMagenta, color.Bold)
	ColorSubtle  = color.New(color.FgHiBlack)
	ColorBold    = color.New(color.Bold)
)

// ProcessingMode represents the processing mode
type ProcessingMode string

const (
	ModeFast ProcessingMode = "fast"
)

// ModeConfig holds the configuration for each mode
type ModeConfig struct {
	Name                string
	Description         string
	Icon                string
	AgenticEnabled      bool
	SelfReflection      bool
	MaxIterations       int
	ValidationEnabled   bool
	Model               string
	EstimatedTime       string
	QualityRating       string
	MultiStageAnalysis  bool
}

// GetModeConfigs returns available processing modes
func GetModeConfigs() map[ProcessingMode]ModeConfig {
	return map[ProcessingMode]ModeConfig{
		ModeFast: {
			Name:               "Fast Mode",
			Description:        "Quick processing with single-pass analysis",
			Icon:               "⚡",
			AgenticEnabled:     false,
			SelfReflection:     false,
			MaxIterations:      1,
			ValidationEnabled:  false,
			Model:              "gemini-2.0-flash",
			EstimatedTime:      "~45-60 seconds per paper",
			QualityRating:      "⭐⭐⭐ Good",
			MultiStageAnalysis: false,
		},
	}
}

// ShowBanner displays the application banner
func ShowBanner() {
	banner := figure.NewFigure("Archivist", "slant", true)
	ColorTitle.Println(banner.String())
	fmt.Println()
	ColorInfo.Println("  Research Paper Helper")
	ColorSubtle.Println("  Convert research papers to student-friendly LaTeX reports")
	fmt.Println()
}

// PromptMode returns the fast mode (since it's the only mode now)
// Returns empty string and error if user wants to go back
func PromptMode() (ProcessingMode, error) {
	// Since we only have one mode now, just return it directly
	// No need to prompt the user
	return ModeFast, nil
}

// ShowModeDetails displays detailed information about the selected mode
func ShowModeDetails(mode ProcessingMode) {
	configs := GetModeConfigs()
	config := configs[mode]

	fmt.Println()
	ColorBold.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("%s  %s Selected\n", config.Icon, config.Name)
	ColorBold.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Println()

	fmt.Printf("  📊 Quality:           %s\n", config.QualityRating)
	fmt.Printf("  ⏱️  Estimated Time:    %s\n", config.EstimatedTime)
	fmt.Printf("  🤖 AI Model:          %s\n", config.Model)
	fmt.Printf("  🔄 Self-Reflection:   %s\n", formatBool(config.SelfReflection))
	fmt.Printf("  ✅ Validation:        %s\n", formatBool(config.ValidationEnabled))
	fmt.Printf("  🔬 Multi-Stage:       %s\n", formatBool(config.MultiStageAnalysis))
	fmt.Println()
}

// ShowModeDetailsWithConfig displays detailed information using actual config values
func ShowModeDetailsWithConfig(mode ProcessingMode, actualConfig *app.Config) {
	configs := GetModeConfigs()
	config := configs[mode]

	// Get actual model name from config
	actualModel := actualConfig.Gemini.Model

	fmt.Println()
	ColorBold.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("%s  %s Selected\n", config.Icon, config.Name)
	ColorBold.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Println()

	fmt.Printf("  📊 Quality:           %s\n", config.QualityRating)
	fmt.Printf("  ⏱️  Estimated Time:    %s\n", config.EstimatedTime)
	fmt.Printf("  🤖 AI Model:          %s\n", actualModel)
	fmt.Printf("  🔄 Self-Reflection:   %s\n", formatBool(actualConfig.Gemini.Agentic.SelfReflection))
	fmt.Printf("  ✅ Validation:        %s\n", formatBool(actualConfig.Gemini.Agentic.Stages.LatexGeneration.Validation))
	fmt.Printf("  🔬 Multi-Stage:       %s\n", formatBool(actualConfig.Gemini.Agentic.MultiStageAnalysis))
	fmt.Println()
}

// PromptSelectPapers allows the user to select multiple papers from a list
func PromptSelectPapers(papers []string) ([]string, error) {
	if len(papers) == 0 {
		return nil, fmt.Errorf("no papers available")
	}

	// Create display items with indices
	items := make([]string, len(papers))
	for i, paper := range papers {
		items[i] = fmt.Sprintf("%d. %s", i+1, paper)
	}

	ColorBold.Println("═══════════════════════════════════════════════════════════════")
	ColorBold.Println("              SELECT PAPERS TO PROCESS                         ")
	ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	ColorInfo.Println("Available papers:")
	fmt.Println()

	for _, item := range items {
		ColorTitle.Println(item)
	}

	fmt.Println()
	ColorSubtle.Println("Enter paper numbers separated by commas (e.g., 1,3,5)")
	ColorSubtle.Println("Or enter 'all' to select all papers")
	ColorSubtle.Println("Press Ctrl+C to cancel")
	fmt.Println()

	prompt := promptui.Prompt{
		Label: "Select papers",
	}

	input, err := prompt.Run()
	if err != nil {
		return nil, err
	}

	// Handle "all" case
	if input == "all" || input == "ALL" {
		return papers, nil
	}

	// Parse comma-separated indices
	var selected []string
	var indices []int

	// Split by comma and parse each number
	parts := splitByComma(input)
	for _, part := range parts {
		var idx int
		_, err := fmt.Sscanf(part, "%d", &idx)
		if err != nil || idx < 1 || idx > len(papers) {
			PrintWarning(fmt.Sprintf("Invalid index: %s (must be between 1 and %d)", part, len(papers)))
			continue
		}
		indices = append(indices, idx-1)
	}

	if len(indices) == 0 {
		return nil, fmt.Errorf("no valid papers selected")
	}

	// Get selected papers
	for _, idx := range indices {
		selected = append(selected, papers[idx])
	}

	return selected, nil
}

// splitByComma splits a string by comma and trims whitespace
func splitByComma(s string) []string {
	var parts []string
	current := ""
	for _, char := range s {
		if char == ',' {
			if len(current) > 0 {
				parts = append(parts, current)
				current = ""
			}
		} else if char != ' ' && char != '\t' {
			current += string(char)
		}
	}
	if len(current) > 0 {
		parts = append(parts, current)
	}
	return parts
}

// ConfirmProcessing asks for final confirmation
func ConfirmProcessing(fileCount int) bool {
	prompt := promptui.Prompt{
		Label:     fmt.Sprintf("Process %d file(s)?", fileCount),
		IsConfirm: true,
		Default:   "y",
	}

	result, err := prompt.Run()
	if err != nil {
		return false
	}

	return result == "y" || result == "Y" || result == ""
}

// CreateProgressBar creates a new progress bar
func CreateProgressBar(total int, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions(total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWidth(50),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerHead:    "█",
			SaucerPadding: "░",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetPredictTime(true),
	)
}

// PrintSuccess prints a success message
func PrintSuccess(msg string) {
	ColorSuccess.Printf("✅ %s\n", msg)
}

// PrintError prints an error message
func PrintError(msg string) {
	ColorError.Printf("❌ %s\n", msg)
}

// PrintWarning prints a warning message
func PrintWarning(msg string) {
	ColorWarning.Printf("⚠️  %s\n", msg)
}

// PrintInfo prints an info message
func PrintInfo(msg string) {
	ColorInfo.Printf("ℹ️  %s\n", msg)
}

// PrintStage prints a processing stage header
func PrintStage(stage, description string) {
	ColorBold.Printf("\n┌─ %s\n", stage)
	ColorSubtle.Printf("└─ %s\n", description)
}

// PrintSummary prints a processing summary
func PrintSummary(successful, failed, skipped int, totalTime time.Duration) {
	fmt.Println()
	ColorBold.Println("═══════════════════════════════════════════════════════════════")
	ColorBold.Println("                    PROCESSING SUMMARY                         ")
	ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	if successful > 0 {
		ColorSuccess.Printf("  ✅ Successful:  %d\n", successful)
	}
	if failed > 0 {
		ColorError.Printf("  ❌ Failed:      %d\n", failed)
	}
	if skipped > 0 {
		ColorWarning.Printf("  ⏭️  Skipped:     %d\n", skipped)
	}

	fmt.Println()
	ColorInfo.Printf("  ⏱️  Total Time:  %s\n", formatDuration(totalTime))
	if successful > 0 {
		ColorInfo.Printf("  📊 Avg Time:    %s per paper\n", formatDuration(totalTime/time.Duration(successful)))
	}
	fmt.Println()
}

// Helper functions
func formatBool(b bool) string {
	if b {
		return ColorSuccess.Sprint("Enabled")
	}
	return ColorSubtle.Sprint("Disabled")
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}
