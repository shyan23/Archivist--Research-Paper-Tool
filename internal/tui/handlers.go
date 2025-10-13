package tui

import (
	"archivist/internal/app"
	"archivist/internal/compiler"
	"archivist/internal/ui"
	"archivist/internal/worker"
	"archivist/pkg/fileutil"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// handleEnter processes the enter key based on current screen
func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	if m.screen == screenMain {
		selectedItem := m.mainMenu.SelectedItem()
		if selectedItem == nil {
			return m, nil
		}

		action := selectedItem.(item).action

		switch action {
		case "view_library":
			m.navigateTo(screenViewLibrary)
			m.loadLibraryPapers()
		case "view_processed":
			m.navigateTo(screenViewProcessed)
			m.loadProcessedPapers()
		case "process_single":
			m.navigateTo(screenSelectPaper)
			m.loadPapersForSelection()
		case "process_all":
			// Exit TUI and trigger batch processing
			m.processing = true
			m.processingMsg = "batch"
			return m, tea.Quit
		}
	} else if m.screen == screenViewLibrary {
		// Handle PDF opening in library view
		selectedItem := m.libraryList.SelectedItem()
		if selectedItem != nil {
			pdfPath := selectedItem.(item).action
			m.selectedPaper = pdfPath
			m.processingMsg = "open_pdf"
			return m, tea.Quit
		}
	} else if m.screen == screenViewProcessed {
		// Handle opening processed paper report
		selectedItem := m.processedList.SelectedItem()
		if selectedItem != nil {
			// Get the report path from metadata
			hash, _ := fileutil.ComputeFileHash(selectedItem.(item).action)
			if record, exists := m.metadataStore.GetRecord(hash); exists && record.ReportPath != "" {
				m.selectedPaper = record.ReportPath
				m.processingMsg = "open_report"
				return m, tea.Quit
			}
		}
	} else if m.screen == screenSelectPaper {
		// Handle paper selection
		selectedItem := m.singlePaperList.SelectedItem()
		if selectedItem != nil {
			m.selectedPaper = selectedItem.(item).description
			// Start processing
			return m, tea.Quit
		}
	}

	return m, nil
}

// handleOpenPDF opens a PDF file using the system's default PDF viewer
func handleOpenPDF(pdfPath string) error {
	fmt.Print("\033[H\033[2J") // Clear screen
	ui.ShowBanner()
	ui.PrintInfo(fmt.Sprintf("Opening: %s", pdfPath))

	// Try xdg-open (Linux), open (macOS), or start (Windows)
	var cmd string
	var args []string

	// Detect OS and use appropriate command
	cmd = "xdg-open" // Default for Linux
	args = []string{pdfPath}

	err := exec.Command(cmd, args...).Start()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to open PDF: %v", err))
		ui.PrintInfo("Please open the file manually:")
		ui.ColorBold.Printf("  %s\n\n", pdfPath)
		return err
	}

	ui.PrintSuccess("PDF opened in default viewer")
	fmt.Println()
	ui.PrintInfo("Press Enter to return to main menu...")
	fmt.Scanln()

	// Restart TUI
	return Run("config/config.yaml")
}

// handleSinglePaperProcessing processes a single selected paper
func handleSinglePaperProcessing(paperPath string, config *app.Config) error {
	// Clear screen and show banner
	fmt.Print("\033[H\033[2J")
	ui.ShowBanner()

	// Initialize logger
	if err := app.InitLogger(config); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to initialize logger: %v", err))
		return err
	}

	// Select processing mode
	selectedMode, err := ui.PromptMode()
	if err != nil {
		// User cancelled - return to TUI
		ui.PrintWarning("Mode selection cancelled, returning to main menu...")
		fmt.Println()
		ui.PrintInfo("Press Enter to continue...")
		fmt.Scanln()
		return Run("config/config.yaml")
	}

	// Apply mode configuration
	applyModeConfig(config, selectedMode)
	ui.ShowModeDetailsWithConfig(selectedMode, config)

	// Check dependencies
	ui.PrintStage("Checking Dependencies", "Verifying LaTeX installation")
	if err := compiler.CheckDependencies(config.Latex.Engine == "latexmk", config.Latex.Compiler); err != nil {
		ui.PrintError(fmt.Sprintf("Dependency check failed: %v", err))
		fmt.Println("\nPlease install the required LaTeX tools:")
		fmt.Println("  sudo apt install texlive-latex-extra latexmk")
		return err
	}
	ui.PrintSuccess("All dependencies installed")

	// Confirm processing
	if !ui.ConfirmProcessing(1) {
		ui.PrintWarning("Processing cancelled by user")
		return nil
	}

	// Process the paper
	fmt.Println()
	ui.PrintStage("Processing Paper", filepath.Base(paperPath))
	ctx := context.Background()
	if err := worker.ProcessBatch(ctx, []string{paperPath}, config, false); err != nil {
		ui.PrintError(fmt.Sprintf("Processing failed: %v", err))
		fmt.Println()
		ui.PrintWarning("Processing encountered an error")
		fmt.Println()
		ui.PrintInfo("Error logged to .metadata/processing.log")
		fmt.Println()
		ui.PrintInfo("Returning to main menu in 5 seconds...")
		fmt.Println()
		ui.PrintInfo("(Press Enter to return immediately or Ctrl+C to exit)")

		// Create channel to detect user input
		done := make(chan bool, 1)
		go func() {
			fmt.Scanln()
			done <- true
		}()

		// Wait for either timeout or user pressing Enter
		select {
		case <-done:
			// User pressed Enter, return immediately
			return Run("config/config.yaml")
		case <-time.After(5 * time.Second):
			// Timeout, auto-return
			return Run("config/config.yaml")
		}
	}

	// Processing successful, auto-return to main menu
	fmt.Println()
	ui.PrintSuccess("Processing complete!")
	fmt.Println()
	ui.PrintInfo("Returning to main menu in 3 seconds...")
	fmt.Println()
	ui.PrintInfo("(Press Enter to return immediately or Ctrl+C to exit)")

	// Create channel to detect user input
	done := make(chan bool, 1)
	go func() {
		fmt.Scanln()
		done <- true
	}()

	// Wait for either timeout or user pressing Enter
	select {
	case <-done:
		// User pressed Enter, return immediately
		return Run("config/config.yaml")
	case <-time.After(3 * time.Second):
		// Timeout, auto-return
		return Run("config/config.yaml")
	}
}

// handleBatchProcessing processes all papers in the library
func handleBatchProcessing(config *app.Config) error {
	// Clear screen and show banner
	fmt.Print("\033[H\033[2J")
	ui.ShowBanner()

	// Initialize logger
	if err := app.InitLogger(config); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to initialize logger: %v", err))
		return err
	}

	// Select processing mode
	selectedMode, err := ui.PromptMode()
	if err != nil {
		// User cancelled - return to TUI
		ui.PrintWarning("Mode selection cancelled, returning to main menu...")
		fmt.Println()
		ui.PrintInfo("Press Enter to continue...")
		fmt.Scanln()
		return Run("config/config.yaml")
	}

	// Apply mode configuration
	applyModeConfig(config, selectedMode)
	ui.ShowModeDetailsWithConfig(selectedMode, config)

	ui.PrintInfo(fmt.Sprintf("Using %d parallel workers", config.Processing.MaxWorkers))

	// Check dependencies
	ui.PrintStage("Checking Dependencies", "Verifying LaTeX installation")
	if err := compiler.CheckDependencies(config.Latex.Engine == "latexmk", config.Latex.Compiler); err != nil {
		ui.PrintError(fmt.Sprintf("Dependency check failed: %v", err))
		fmt.Println("\nPlease install the required LaTeX tools:")
		fmt.Println("  sudo apt install texlive-latex-extra latexmk")
		return err
	}
	ui.PrintSuccess("All dependencies installed")

	// Get all PDF files
	files, err := fileutil.GetPDFFiles(config.InputDir)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to get PDF files: %v", err))
		return err
	}

	if len(files) == 0 {
		ui.PrintWarning("No PDF files found in library")
		return nil
	}

	ui.PrintInfo(fmt.Sprintf("Found %d PDF file(s)", len(files)))

	// Confirm processing
	if !ui.ConfirmProcessing(len(files)) {
		ui.PrintWarning("Processing cancelled by user")
		return nil
	}

	// Process all files
	fmt.Println()
	ui.PrintStage("Processing Papers", "Starting batch processing")
	ctx := context.Background()
	if err := worker.ProcessBatch(ctx, files, config, false); err != nil {
		ui.PrintError(fmt.Sprintf("Processing failed: %v", err))
		fmt.Println()
		ui.PrintWarning("Batch processing encountered an error")
		fmt.Println()
		ui.PrintInfo("Error logged to .metadata/processing.log")
		fmt.Println()
		ui.PrintInfo("Returning to main menu in 5 seconds...")
		fmt.Println()
		ui.PrintInfo("(Press Enter to return immediately or Ctrl+C to exit)")

		// Create channel to detect user input
		done := make(chan bool, 1)
		go func() {
			fmt.Scanln()
			done <- true
		}()

		// Wait for either timeout or user pressing Enter
		select {
		case <-done:
			// User pressed Enter, return immediately
			return Run("config/config.yaml")
		case <-time.After(5 * time.Second):
			// Timeout, auto-return
			return Run("config/config.yaml")
		}
	}

	// Processing successful, auto-return to main menu
	fmt.Println()
	ui.PrintSuccess("Batch processing complete!")
	fmt.Println()
	ui.PrintInfo("Returning to main menu in 3 seconds...")
	fmt.Println()
	ui.PrintInfo("(Press Enter to return immediately or Ctrl+C to exit)")

	// Create channel to detect user input
	done := make(chan bool, 1)
	go func() {
		fmt.Scanln()
		done <- true
	}()

	// Wait for either timeout or user pressing Enter
	select {
	case <-done:
		// User pressed Enter, return immediately
		return Run("config/config.yaml")
	case <-time.After(3 * time.Second):
		// Timeout, auto-return
		return Run("config/config.yaml")
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
		config.Gemini.Agentic.Stages.MethodologyAnalysis.Model = "models/gemini-2.0-flash-thinking-exp"
	} else {
		config.Gemini.Agentic.Stages.MethodologyAnalysis.Model = "models/gemini-2.0-flash-exp"
	}
}
