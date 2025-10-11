package ui

import (
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

// ProcessingMode represents the quality vs speed tradeoff
type ProcessingMode string

const (
	ModeFast    ProcessingMode = "fast"
	ModeQuality ProcessingMode = "quality"
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
		ModeQuality: {
			Name:               "Quality Mode",
			Description:        "Thorough analysis with self-reflection and validation",
			Icon:               "🎯",
			AgenticEnabled:     true,
			SelfReflection:     true,
			MaxIterations:      2,
			ValidationEnabled:  true,
			Model:              "gemini-1.5-pro",
			EstimatedTime:      "~90-120 seconds per paper",
			QualityRating:      "⭐⭐⭐⭐⭐ Excellent",
			MultiStageAnalysis: true,
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

// PromptMode asks the user to select a processing mode
// Returns empty string and error if user wants to go back
func PromptMode() (ProcessingMode, error) {
	modes := GetModeConfigs()

	items := []string{
		fmt.Sprintf("%s %s - %s (%s)", modes[ModeFast].Icon, modes[ModeFast].Name, modes[ModeFast].Description, modes[ModeFast].EstimatedTime),
		fmt.Sprintf("%s %s - %s (%s)", modes[ModeQuality].Icon, modes[ModeQuality].Name, modes[ModeQuality].Description, modes[ModeQuality].EstimatedTime),
		"⬅️  Go Back",
	}

	prompt := promptui.Select{
		Label: "Select Processing Mode (ESC to go back)",
		Items: items,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . | cyan | bold }}",
			Active:   "▸ {{ . | green | bold }}",
			Inactive: "  {{ . }}",
			Selected: "✔ {{ . | green }}",
		},
		Size: 3,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		// User pressed ESC or Ctrl+C
		return "", err
	}

	switch idx {
	case 0:
		return ModeFast, nil
	case 1:
		return ModeQuality, nil
	case 2:
		// User selected "Go Back"
		return "", fmt.Errorf("user cancelled")
	}

	return "", fmt.Errorf("invalid selection")
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
