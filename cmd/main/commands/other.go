package commands

import (
	"archivist/internal/app"
	"archivist/internal/compiler"
	"archivist/internal/tui"
	"archivist/internal/ui"
	"archivist/internal/wizard"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// NewStatusCommand creates the status command
func NewStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status [file]",
		Short: "Show processing status",
		Long:  "Check if a paper has been processed by looking for its report",
		Args:  cobra.ExactArgs(1),
		Run:   runStatus,
	}
}

func runStatus(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(ConfigPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	filePath := args[0]
	basename := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	ui.ColorBold.Println("                      FILE STATUS                              ")
	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	ui.ColorTitle.Printf("📄 Input:     %s\n", filePath)
	fmt.Println()

	// Check if report exists in reports folder
	// Try to find matching report (approximate match since title may be modified)
	reports, _ := filepath.Glob(filepath.Join(config.ReportOutputDir, "*.pdf"))
	foundReport := false
	var reportPath string

	for _, report := range reports {
		reportBase := filepath.Base(report)
		if strings.Contains(strings.ToLower(reportBase), strings.ToLower(basename)) {
			foundReport = true
			reportPath = report
			break
		}
	}

	if foundReport {
		ui.ColorSuccess.Println("✅ Status:    Processed")
		ui.ColorInfo.Printf("📊 Report:    %s\n", reportPath)

		// Check for tex file
		texPath := filepath.Join(config.TexOutputDir, strings.TrimSuffix(filepath.Base(reportPath), ".pdf")+".tex")
		if _, err := os.Stat(texPath); err == nil {
			ui.ColorInfo.Printf("📝 LaTeX:     %s\n", texPath)
		}
	} else {
		ui.ColorWarning.Println("⏳ Status:    Not processed")
	}

	fmt.Println()
}

// NewCleanCommand creates the clean command
func NewCleanCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "Clean temporary files",
		Long:  "Remove auxiliary LaTeX files",
		Run:   runClean,
	}
}

func runClean(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(ConfigPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	ui.PrintStage("Cleaning", "Removing auxiliary LaTeX files")

	extensions := []string{"*.aux", "*.log", "*.out", "*.toc", "*.fdb_latexmk", "*.fls", "*.synctex.gz"}
	totalCleaned := 0

	for _, ext := range extensions {
		matches, _ := filepath.Glob(filepath.Join(config.TexOutputDir, ext))
		for _, file := range matches {
			if err := os.Remove(file); err == nil {
				totalCleaned++
			}
		}
	}

	ui.PrintSuccess(fmt.Sprintf("Cleaned %d auxiliary files", totalCleaned))
}

// NewCheckCommand creates the check command
func NewCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check dependencies",
		Long:  "Verify that all required tools (pdflatex, latexmk) are installed",
		Run:   runCheck,
	}
}

func runCheck(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(ConfigPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	ui.PrintStage("Dependency Check", "Verifying system requirements")

	useLatexmk := config.Latex.Engine == "latexmk"
	if err := compiler.CheckDependencies(useLatexmk, config.Latex.Compiler); err != nil {
		ui.PrintError(fmt.Sprintf("%v", err))
		fmt.Println()
		ui.ColorWarning.Println("Please install the following:")
		if useLatexmk {
			fmt.Println("  • latexmk:        sudo apt install latexmk")
		}
		fmt.Printf("  • %s:  sudo apt install texlive-latex-extra\n", config.Latex.Compiler)
		fmt.Println()
		os.Exit(1)
	}

	ui.PrintSuccess("All dependencies installed")
	fmt.Println()
	ui.ColorInfo.Printf("  📦 LaTeX Compiler:  %s\n", config.Latex.Compiler)
	if useLatexmk {
		ui.ColorInfo.Println("  ⚙️  Build Tool:      latexmk")
	}
	ui.ColorInfo.Printf("  🔧 Workers:         %d\n", config.Processing.MaxWorkers)
	ui.ColorInfo.Printf("  🤖 AI Model:        %s\n", config.Gemini.Model)
	fmt.Println()
}

// NewRunCommand creates the run command (interactive TUI)
func NewRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Launch interactive TUI",
		Long:  "Start the interactive terminal UI for browsing and processing papers",
		Run:   runInteractive,
	}
}

func runInteractive(cmd *cobra.Command, args []string) {
	if err := tui.Run(ConfigPath); err != nil {
		ui.PrintError(fmt.Sprintf("TUI error: %v", err))
		os.Exit(1)
	}
}

// NewConfigureCommand creates the configure command
func NewConfigureCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "configure",
		Short: "Interactive configuration wizard",
		Long:  "Launch an interactive wizard to set up your config.yaml file",
		Run:   runConfigure,
	}
}

func runConfigure(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	wiz := wizard.NewConfigWizard()
	if err := wiz.Run(ConfigPath); err != nil {
		ui.PrintError(fmt.Sprintf("Configuration wizard failed: %v", err))
		os.Exit(1)
	}
}
