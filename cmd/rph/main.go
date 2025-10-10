package main

import (
	"archivist/internal/app"
	"archivist/internal/compiler"
	"archivist/internal/storage"
	"archivist/internal/tui"
	"archivist/internal/ui"
	"archivist/internal/worker"
	"archivist/pkg/fileutil"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	configPath  string
	force       bool
	parallel    int
	outputDir   string
	unprocessed bool
	mode        string // Processing mode: "fast" or "quality"
	interactive bool   // Enable interactive mode selection
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "rph",
		Short: "Research Paper Helper - Convert research papers to student-friendly LaTeX reports",
		Long: `Research Paper Helper analyzes AI/ML research papers using Gemini AI
and generates comprehensive, student-friendly LaTeX reports with detailed
explanations of methodologies, breakthroughs, and results.`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config/config.yaml", "config file path")

	// Process command
	processCmd := &cobra.Command{
		Use:   "process [file|directory]",
		Short: "Process research paper(s)",
		Long:  "Process a single PDF file or all PDF files in a directory",
		Args:  cobra.ExactArgs(1),
		Run:   runProcess,
	}
	processCmd.Flags().BoolVarP(&force, "force", "f", false, "reprocess even if already processed")
	processCmd.Flags().IntVarP(&parallel, "parallel", "p", 0, "number of parallel workers (default: config value)")
	processCmd.Flags().StringVarP(&mode, "mode", "m", "", "processing mode: 'fast' or 'quality' (default: interactive)")
	processCmd.Flags().BoolVarP(&interactive, "interactive", "i", true, "enable interactive mode selection")

	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List processed papers",
		Long:  "Display a list of all processed papers",
		Run:   runList,
	}
	listCmd.Flags().BoolVarP(&unprocessed, "unprocessed", "u", false, "show unprocessed papers")

	// Status command
	statusCmd := &cobra.Command{
		Use:   "status [file]",
		Short: "Show processing status",
		Long:  "Display processing status for a specific file",
		Args:  cobra.ExactArgs(1),
		Run:   runStatus,
	}

	// Clean command
	cleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean temporary files",
		Long:  "Remove auxiliary LaTeX files",
		Run:   runClean,
	}

	// Check command
	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Check dependencies",
		Long:  "Verify that all required tools (pdflatex, latexmk) are installed",
		Run:   runCheck,
	}

	// Run command - Interactive TUI
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Launch interactive TUI",
		Long:  "Start the interactive terminal UI for browsing and processing papers",
		Run:   runInteractive,
	}

	rootCmd.AddCommand(processCmd, listCmd, statusCmd, cleanCmd, checkCmd, runCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runProcess(cmd *cobra.Command, args []string) {
	// Show banner
	ui.ShowBanner()

	// Load config
	config, err := app.LoadConfig(configPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	// Initialize logger
	if err := app.InitLogger(config); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to initialize logger: %v", err))
		os.Exit(1)
	}

	// Select processing mode
	var selectedMode ui.ProcessingMode
	if mode != "" {
		// Mode specified via flag
		selectedMode = ui.ProcessingMode(mode)
	} else if interactive {
		// Interactive mode selection
		selectedMode, err = ui.PromptMode()
		if err != nil {
			ui.PrintError(fmt.Sprintf("Mode selection cancelled: %v", err))
			os.Exit(1)
		}
	} else {
		// Default to fast mode
		selectedMode = ui.ModeFast
	}

	// Apply mode configuration
	applyModeConfig(config, selectedMode)

	// Show mode details
	ui.ShowModeDetails(selectedMode)

	// Override parallel workers if specified
	if parallel > 0 {
		config.Processing.MaxWorkers = parallel
		ui.PrintInfo(fmt.Sprintf("Using %d parallel workers (overridden)", parallel))
	} else {
		ui.PrintInfo(fmt.Sprintf("Using %d parallel workers", config.Processing.MaxWorkers))
	}

	// Check dependencies
	ui.PrintStage("Checking Dependencies", "Verifying LaTeX installation")
	if err := compiler.CheckDependencies(config.Latex.Engine == "latexmk", config.Latex.Compiler); err != nil {
		ui.PrintError(fmt.Sprintf("Dependency check failed: %v", err))
		fmt.Println("\nPlease install the required LaTeX tools:")
		fmt.Println("  sudo apt install texlive-latex-extra latexmk")
		os.Exit(1)
	}
	ui.PrintSuccess("All dependencies installed")

	// Get input path
	inputPath := args[0]

	// Determine if it's a file or directory
	info, err := os.Stat(inputPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to access path: %v", err))
		os.Exit(1)
	}

	var files []string
	if info.IsDir() {
		files, err = fileutil.GetPDFFiles(inputPath)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to get PDF files: %v", err))
			os.Exit(1)
		}
	} else {
		if filepath.Ext(inputPath) != ".pdf" {
			ui.PrintError("File must be a PDF")
			os.Exit(1)
		}
		files = []string{inputPath}
	}

	if len(files) == 0 {
		ui.PrintWarning("No PDF files found")
		return
	}

	ui.PrintInfo(fmt.Sprintf("Found %d PDF file(s)", len(files)))

	// Confirm processing
	if interactive && !ui.ConfirmProcessing(len(files)) {
		ui.PrintWarning("Processing cancelled by user")
		return
	}

	// Process files
	fmt.Println()
	ui.PrintStage("Processing Papers", "Starting batch processing")
	ctx := context.Background()
	if err := worker.ProcessBatch(ctx, files, config, force); err != nil {
		ui.PrintError(fmt.Sprintf("Processing failed: %v", err))
		os.Exit(1)
	}
}

// applyModeConfig applies the selected mode's configuration
func applyModeConfig(config *app.Config, mode ui.ProcessingMode) {
	modes := ui.GetModeConfigs()
	modeConfig := modes[mode]

	config.Gemini.Agentic.Enabled = modeConfig.AgenticEnabled
	config.Gemini.Agentic.SelfReflection = modeConfig.SelfReflection
	config.Gemini.Agentic.MaxIterations = modeConfig.MaxIterations
	config.Gemini.Agentic.MultiStageAnalysis = modeConfig.MultiStageAnalysis
	config.Gemini.Agentic.Stages.LatexGeneration.Validation = modeConfig.ValidationEnabled
	config.Gemini.Model = modeConfig.Model

	// Use appropriate model for methodology analysis
	if mode == ui.ModeQuality {
		config.Gemini.Agentic.Stages.MethodologyAnalysis.Model = "gemini-1.5-pro"
	} else {
		config.Gemini.Agentic.Stages.MethodologyAnalysis.Model = "gemini-1.5-flash"
	}
}

func runList(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(configPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	metadataStore, err := storage.NewMetadataStore(config.MetadataDir)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load metadata: %v", err))
		os.Exit(1)
	}

	records := metadataStore.GetAllRecords()

	if unprocessed {
		// Show unprocessed files
		files, err := fileutil.GetPDFFiles(config.InputDir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to get PDF files: %v", err))
			os.Exit(1)
		}

		ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
		ui.ColorBold.Println("                   UNPROCESSED FILES                           ")
		ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
		fmt.Println()

		count := 0
		for _, file := range files {
			hash, _ := fileutil.ComputeFileHash(file)
			if !metadataStore.IsProcessed(hash) {
				count++
				ui.ColorWarning.Printf("  %d. %s\n", count, file)
			}
		}

		if count == 0 {
			ui.PrintSuccess("All files have been processed!")
		} else {
			fmt.Println()
			ui.PrintInfo(fmt.Sprintf("Total unprocessed: %d", count))
		}
		return
	}

	// Show processed files
	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	ui.ColorBold.Printf("              PROCESSED PAPERS (%d)                        \n", len(records))
	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	if len(records) == 0 {
		ui.PrintWarning("No papers have been processed yet")
		return
	}

	for i, record := range records {
		ui.ColorTitle.Printf("%d. %s\n", i+1, record.PaperTitle)
		ui.ColorSubtle.Printf("   File:      %s\n", record.FilePath)

		// Status with color
		switch record.Status {
		case storage.StatusCompleted:
			ui.ColorSuccess.Printf("   Status:    %s\n", record.Status)
		case storage.StatusFailed:
			ui.ColorError.Printf("   Status:    %s\n", record.Status)
		case storage.StatusProcessing:
			ui.ColorWarning.Printf("   Status:    %s\n", record.Status)
		default:
			fmt.Printf("   Status:    %s\n", record.Status)
		}

		ui.ColorSubtle.Printf("   Processed: %s\n", record.ProcessedAt.Format("2006-01-02 15:04:05"))

		if record.ReportPath != "" {
			ui.ColorInfo.Printf("   Report:    %s\n", record.ReportPath)
		}
		if record.Error != "" {
			ui.ColorError.Printf("   Error:     %s\n", record.Error)
		}
		fmt.Println()
	}
}

func runStatus(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(configPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	filePath := args[0]
	hash, err := fileutil.ComputeFileHash(filePath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to compute hash: %v", err))
		os.Exit(1)
	}

	metadataStore, err := storage.NewMetadataStore(config.MetadataDir)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load metadata: %v", err))
		os.Exit(1)
	}

	record, exists := metadataStore.GetRecord(hash)
	if !exists {
		ui.PrintWarning("Status: Not processed")
		return
	}

	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	ui.ColorBold.Println("                      FILE STATUS                              ")
	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	ui.ColorTitle.Printf("📄 Paper:     %s\n", record.PaperTitle)
	fmt.Println()

	switch record.Status {
	case storage.StatusCompleted:
		ui.ColorSuccess.Printf("✅ Status:    %s\n", record.Status)
	case storage.StatusFailed:
		ui.ColorError.Printf("❌ Status:    %s\n", record.Status)
	case storage.StatusProcessing:
		ui.ColorWarning.Printf("⏳ Status:    %s\n", record.Status)
	default:
		fmt.Printf("Status:       %s\n", record.Status)
	}

	ui.ColorSubtle.Printf("📅 Processed: %s\n", record.ProcessedAt.Format("2006-01-02 15:04:05"))

	if record.TexFilePath != "" {
		ui.ColorInfo.Printf("📝 LaTeX:     %s\n", record.TexFilePath)
	}
	if record.ReportPath != "" {
		ui.ColorInfo.Printf("📊 Report:    %s\n", record.ReportPath)
	}
	if record.Error != "" {
		fmt.Println()
		ui.ColorError.Printf("❌ Error:     %s\n", record.Error)
	}
	fmt.Println()
}

func runClean(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(configPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	ui.PrintStage("Cleaning", "Removing auxiliary LaTeX files")

	// Clean aux files from tex_files directory
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

func runCheck(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(configPath)
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

func runInteractive(cmd *cobra.Command, args []string) {
	// Launch interactive TUI
	if err := tui.Run(configPath); err != nil {
		ui.PrintError(fmt.Sprintf("TUI error: %v", err))
		os.Exit(1)
	}
}
