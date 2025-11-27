package tui

import (
	"archivist/internal/app"
	"archivist/internal/compiler"
	"archivist/internal/ui"
	"archivist/internal/worker"
	"archivist/pkg/fileutil"
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// handleSpacebar toggles selection in multi-select mode
func (m Model) handleSpacebar() (tea.Model, tea.Cmd) {
	if m.screen != screenSelectMultiplePapers {
		return m, nil
	}

	// Get current index
	idx := m.multiPaperList.Index()

	// Toggle selection
	if m.multiSelectIndexes[idx] {
		delete(m.multiSelectIndexes, idx)
	} else {
		m.multiSelectIndexes[idx] = true
	}

	// Update title with selection count
	m.multiPaperList.Title = fmt.Sprintf("📋 Select Papers (Space to toggle, Enter to confirm) - %d selected", len(m.multiSelectIndexes))

	return m, nil
}

// handleEnter processes the enter key based on current screen
func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	if m.screen == screenMain {
		selectedItem := m.mainMenu.SelectedItem()
		if selectedItem == nil {
			return m, nil
		}

		action := selectedItem.(item).action

		switch action {
		case "search_papers":
			// Initialize search mode menu
			modeItems := []list.Item{
				item{
					title:       "📝 Manual Search",
					description: "Enter a search query manually",
					action:      "manual",
				},
				item{
					title:       "🔍 Find Similar Papers",
					description: "Select a paper from your library to find similar papers",
					action:      "similar",
				},
			}
			delegate := createStyledDelegate()
			m.searchModeMenu = list.New(modeItems, delegate, m.width, m.height)
			m.searchModeMenu.Title = "Choose Search Mode"
			m.searchModeMenu.SetShowStatusBar(false)
			m.searchModeMenu.SetFilteringEnabled(false)
			m.searchModeMenu.Styles.Title = titleStyle
			if m.width > 0 && m.height > 0 {
				m.searchModeMenu.SetSize(m.width-4, m.height-8)
			}

			m.navigateTo(screenSearchMode)
			m.searchInput = ""
			m.searchLoading = false
			m.searchError = ""
		case "view_library":
			m.navigateTo(screenViewLibrary)
			m.loadLibraryPapers()
		case "view_processed":
			m.navigateTo(screenViewProcessed)
			m.loadProcessedPapers()
		case "chat":
			m.navigateTo(screenChatMenu)
			m.loadChatMenu()
		case "process_single":
			m.navigateTo(screenSelectPaper)
			m.loadPapersForSelection()
		case "process_multiple":
			m.navigateTo(screenSelectMultiplePapers)
			m.loadPapersForMultiSelection()
		case "process_all":
			// Exit TUI and trigger batch processing
			m.processing = true
			m.processingMsg = "batch"
			return m, tea.Quit
		case "settings":
			m.navigateTo(screenSettings)
			m.loadSettingsMenu()
			// Ensure menu is sized if we have dimensions
			if m.width > 0 && m.height > 0 {
				m.settingsMenu.SetSize(m.width-4, m.height-8)
			}
		}
	} else if m.screen == screenSettings || m.screen == screenDirectorySettings {
		return m.handleSettingsEnter()
	} else if m.screen == screenSelectMultiplePapers {
		// Confirm selection of multiple papers
		if len(m.multiSelectIndexes) == 0 {
			// No papers selected, do nothing
			return m, nil
		}

		// Collect selected papers
		m.selectedPapers = []string{}
		for idx := range m.multiSelectIndexes {
			if idx < len(m.allPapersForSelect) {
				m.selectedPapers = append(m.selectedPapers, m.allPapersForSelect[idx])
			}
		}

		// Exit TUI to start processing
		return m, tea.Quit
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
			// Just open the PDF directly
			m.selectedPaper = selectedItem.(item).action
			m.processingMsg = "open_report"
			return m, tea.Quit
		}
	} else if m.screen == screenSelectPaper {
		// Handle paper selection
		selectedItem := m.singlePaperList.SelectedItem()
		if selectedItem != nil {
			m.selectedPaper = selectedItem.(item).action
			// Start processing
			return m, tea.Quit
		}
	} else if m.screen == screenChatMenu {
		// Handle chat menu selection
		selectedItem := m.chatMenu.SelectedItem()
		if selectedItem != nil {
			action := selectedItem.(item).action
			switch action {
			case "chat_processed":
				m.navigateTo(screenChatSelectPapers)
				m.loadPapersForChat()
			case "chat_any":
				m.navigateTo(screenChatSelectAnyPaper)
				m.loadAnyPaperForChat()
			}
		}
	} else if m.screen == screenChatSelectAnyPaper {
		// Handle selection from any paper list (similar to single select)
		selectedItem := m.chatPaperList.SelectedItem()
		if selectedItem != nil {
			m.selectedPaper = selectedItem.(item).action // This is the PDF path
			m.processingForChat = true
			m.processingMsg = "process_for_chat"
			return m, tea.Quit
		}
	} else if m.screen == screenSearchResults {
		// Handle search result selection (download paper)
		return m.handleSearchResultSelection()
	} else if m.screen == screenSearchMode {
		// Handle search mode selection
		return m.handleSearchModeSelection()
	} else if m.screen == screenSimilarPaperSelect {
		// Handle paper selection for similar search
		return m.handleSimilarPaperSelection()
	} else if m.screen == screenSimilarFactorsEdit {
		// Don't handle enter here - handled separately in Update
		return m, nil
	} else if m.screen == screenGraphMenu {
		// Handle graph menu selection
		selectedItem := m.graphMenu.SelectedItem()
		if selectedItem != nil {
			action := selectedItem.(item).action
			m.handleGraphMenuAction(action)
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
	ui.PrintInfo("Press Enter to return to main menu (or wait 3 seconds)...")

	// Wait for user input with timeout
	done := make(chan bool, 1)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
		done <- true
	}()

	// Wait for Enter or timeout
	select {
	case <-done:
		// User pressed Enter
	case <-time.After(3 * time.Second):
		// Timeout after 3 seconds
		fmt.Println("\nReturning to main menu...")
	}

	// Restart TUI
	return Run("config/config.yaml")
}

// handleSinglePaperProcessing processes a single selected paper
func handleSinglePaperProcessing(paperPath string, config *app.Config) error {
	// Clear screen and show banner
	fmt.Print("\033[H\033[2J")
	ui.ShowBanner()

	// Initialize logger
	logCleanup, err := app.InitLogger(config)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to initialize logger: %v", err))
		return err
	}
	defer logCleanup()

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

	// Ask if user wants to enable RAG indexing for chat
	enableRAG := ui.PromptEnableRAG()

	// Process the paper
	fmt.Println()
	ui.PrintStage("Processing Paper", filepath.Base(paperPath))
	ctx := context.Background()
	enableGraphBuilding := config.Graph.Enabled && ui.PromptEnableGraphBuilding()
	if err := worker.ProcessBatch(ctx, []string{paperPath}, config, false, enableRAG, enableGraphBuilding); err != nil {
		ui.PrintError(fmt.Sprintf("Processing failed: %v", err))
		fmt.Println()
		ui.PrintWarning("Processing encountered an error")
		fmt.Println()
		ui.PrintInfo("Error logged to .metadata/processing.log")
		fmt.Println()
		ui.PrintInfo("Press Enter to return to main menu (or wait 3 seconds)...")

		// Wait for user input with timeout
		done := make(chan bool, 1)
		go func() {
			reader := bufio.NewReader(os.Stdin)
			reader.ReadString('\n')
			done <- true
		}()

		// Wait for Enter or timeout
		select {
		case <-done:
			// User pressed Enter
		case <-time.After(3 * time.Second):
			// Timeout after 3 seconds
			fmt.Println("\nReturning to main menu...")
		}

		// Return to TUI
		return Run("config/config.yaml")
	}

	// Processing successful, return to main menu
	fmt.Println()
	ui.PrintSuccess("Processing complete!")
	fmt.Println()
	ui.PrintInfo("Press Enter to return to main menu (or wait 3 seconds)...")

	// Wait for user input with timeout
	done := make(chan bool, 1)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
		done <- true
	}()

	// Wait for Enter or timeout
	select {
	case <-done:
		// User pressed Enter
	case <-time.After(3 * time.Second):
		// Timeout after 3 seconds
		fmt.Println("\nReturning to main menu...")
	}

	// Return to TUI
	return Run("config/config.yaml")
}

// handleBatchProcessing processes all papers in the library
func handleBatchProcessing(config *app.Config) error {
	// Clear screen and show banner
	fmt.Print("\033[H\033[2J")
	ui.ShowBanner()

	// Initialize logger
	logCleanup, err := app.InitLogger(config)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to initialize logger: %v", err))
		return err
	}
	defer logCleanup()

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
	// Ask if user wants to enable RAG indexing for chat
	enableRAG := ui.PromptEnableRAG()
	enableGraphBuilding := config.Graph.Enabled && ui.PromptEnableGraphBuilding()

	if err := worker.ProcessBatch(ctx, files, config, false, enableRAG, enableGraphBuilding); err != nil {
		ui.PrintError(fmt.Sprintf("Processing failed: %v", err))
		fmt.Println()
		ui.PrintWarning("Batch processing encountered an error")
		fmt.Println()
		ui.PrintInfo("Error logged to .metadata/processing.log")
		fmt.Println()
		ui.PrintInfo("Press Enter to return to main menu (or wait 3 seconds)...")

		// Wait for user input with timeout
		done := make(chan bool, 1)
		go func() {
			reader := bufio.NewReader(os.Stdin)
			reader.ReadString('\n')
			done <- true
		}()

		// Wait for Enter or timeout
		select {
		case <-done:
			// User pressed Enter
		case <-time.After(3 * time.Second):
			// Timeout after 3 seconds
			fmt.Println("\nReturning to main menu...")
		}

		// Return to TUI
		return Run("config/config.yaml")
	}

	// Processing successful, return to main menu
	fmt.Println()
	ui.PrintSuccess("Batch processing complete!")
	fmt.Println()
	ui.PrintInfo("Press Enter to return to main menu (or wait 3 seconds)...")

	// Wait for user input with timeout
	done := make(chan bool, 1)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
		done <- true
	}()

	// Wait for Enter or timeout
	select {
	case <-done:
		// User pressed Enter
	case <-time.After(3 * time.Second):
		// Timeout after 3 seconds
		fmt.Println("\nReturning to main menu...")
	}

	// Return to TUI
	return Run("config/config.yaml")
}

// handleMultiplePapersProcessing processes multiple selected papers
func handleMultiplePapersProcessing(paperPaths []string, config *app.Config) error {
	// Clear screen and show banner
	fmt.Print("\033[H\033[2J")
	ui.ShowBanner()

	// Initialize logger
	logCleanup, err := app.InitLogger(config)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to initialize logger: %v", err))
		return err
	}
	defer logCleanup()

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

	ui.PrintInfo(fmt.Sprintf("Found %d PDF file(s) to process", len(paperPaths)))

	// Confirm processing
	if !ui.ConfirmProcessing(len(paperPaths)) {
		ui.PrintWarning("Processing cancelled by user")
		return nil
	}

	// Process the papers
	fmt.Println()
	ui.PrintStage("Processing Papers", "Starting batch processing")
	ctx := context.Background()
	// Ask if user wants to enable RAG indexing for chat
	enableRAG := ui.PromptEnableRAG()
	enableGraphBuilding := config.Graph.Enabled && ui.PromptEnableGraphBuilding()

	if err := worker.ProcessBatch(ctx, paperPaths, config, false, enableRAG, enableGraphBuilding); err != nil {
		ui.PrintError(fmt.Sprintf("Processing failed: %v", err))
		fmt.Println()
		ui.PrintWarning("Processing encountered an error")
		fmt.Println()
		ui.PrintInfo("Error logged to .metadata/processing.log")
		fmt.Println()
		ui.PrintInfo("Press Enter to return to main menu (or wait 3 seconds)...")

		// Wait for user input with timeout
		done := make(chan bool, 1)
		go func() {
			reader := bufio.NewReader(os.Stdin)
			reader.ReadString('\n')
			done <- true
		}()

		// Wait for Enter or timeout
		select {
		case <-done:
			// User pressed Enter
		case <-time.After(3 * time.Second):
			// Timeout after 3 seconds
			fmt.Println("\nReturning to main menu...")
		}

		// Return to TUI
		return Run("config/config.yaml")
	}

	// Processing successful, return to main menu
	fmt.Println()
	ui.PrintSuccess("Processing complete!")
	fmt.Println()
	ui.PrintInfo("Press Enter to return to main menu (or wait 3 seconds)...")

	// Wait for user input with timeout
	done := make(chan bool, 1)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
		done <- true
	}()

	// Wait for Enter or timeout
	select {
	case <-done:
		// User pressed Enter
	case <-time.After(3 * time.Second):
		// Timeout after 3 seconds
		fmt.Println("\nReturning to main menu...")
	}

	// Return to TUI
	return Run("config/config.yaml")
}

// handleProcessAndChat processes a paper and immediately starts chat
func handleProcessAndChat(paperPath string, config *app.Config) error {
	// Clear screen and show banner
	fmt.Print("\033[H\033[2J")
	ui.ShowBanner()

	// Initialize logger
	logCleanup, err := app.InitLogger(config)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to initialize logger: %v", err))
		return err
	}
	defer logCleanup()

	// Select processing mode
	selectedMode, err := ui.PromptMode()
	if err != nil {
		// User cancelled - return to TUI
		ui.PrintWarning("Mode selection cancelled, returning to main menu...")
		fmt.Println()
		ui.PrintInfo("Press Enter to continue...")
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
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
		return Run("config/config.yaml")
	}

	// Enable RAG indexing by default for chat
	ui.PrintInfo("📇 RAG indexing enabled - paper will be ready for chat after processing")

	// Process the paper
	fmt.Println()
	ui.PrintStage("Processing Paper for Chat", filepath.Base(paperPath))
	ctx := context.Background()
	enableGraphBuilding := config.Graph.Enabled && ui.PromptEnableGraphBuilding()
	if err := worker.ProcessBatch(ctx, []string{paperPath}, config, false, true, enableGraphBuilding); err != nil {
		ui.PrintError(fmt.Sprintf("Processing failed: %v", err))
		fmt.Println()
		ui.PrintWarning("Processing encountered an error")
		fmt.Println()
		ui.PrintInfo("Error logged to logs/processing.log")
		fmt.Println()
		ui.PrintInfo("Press Enter to return to main menu...")

		// Wait for user input
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')

		// Return to TUI
		return Run("config/config.yaml")
	}

	// Processing successful, now start chat directly
	fmt.Println()
	ui.PrintSuccess("Processing complete! Starting chat...")
	fmt.Println()

	// Extract paper title from path
	basename := filepath.Base(paperPath)
	paperTitle := strings.TrimSuffix(basename, ".pdf")
	paperTitle = strings.ReplaceAll(paperTitle, "_", " ")

	ui.PrintInfo(fmt.Sprintf("You can now ask questions about: %s", paperTitle))
	fmt.Println()
	ui.PrintInfo("Press Enter to start chat...")

	// Wait for user input
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	// TODO: Implement direct chat mode (for now, return to TUI)
	// In the future, this could launch a CLI chat interface
	return Run("config/config.yaml")
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

	// Use fast model for methodology analysis (only one mode now)
	config.Gemini.Agentic.Stages.MethodologyAnalysis.Model = "models/gemini-2.0-flash-exp"
}

// handleSearchInput handles text input for search query and count
func (m Model) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// If in count mode, go back to query mode
		if m.searchInputMode == "count" {
			m.searchInputMode = "query"
			m.searchMaxResults = ""
			return m, nil
		}
		// Otherwise go back to main menu
		m.navigateBack()
		return m, nil

	case "enter":
		// Call the search handler (handles mode switching)
		return m.handleSearchEnter()

	case "backspace":
		// Delete last character from appropriate field
		if m.searchInputMode == "count" {
			if len(m.searchMaxResults) > 0 {
				m.searchMaxResults = m.searchMaxResults[:len(m.searchMaxResults)-1]
			}
		} else {
			if len(m.searchInput) > 0 {
				m.searchInput = m.searchInput[:len(m.searchInput)-1]
			}
		}
		return m, nil

	default:
		// Add character to appropriate field
		if len(msg.Runes) == 1 {
			if m.searchInputMode == "count" {
				// Only allow digits for count
				if msg.Runes[0] >= '0' && msg.Runes[0] <= '9' {
					m.searchMaxResults += string(msg.Runes[0])
				}
			} else {
				// Normal text input for query
				m.searchInput += string(msg.Runes[0])
			}
		}
	}

	return m, nil
}
