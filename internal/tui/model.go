package tui

import (
	"archivist/internal/app"
	"archivist/internal/compiler"
	"archivist/internal/storage"
	"archivist/internal/ui"
	"archivist/internal/worker"
	"archivist/pkg/fileutil"
	"context"
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Screen types
type screen int

const (
	screenMain screen = iota
	screenViewLibrary
	screenViewProcessed
	screenSelectPaper
	screenProcessing
)

// Model represents the TUI application state
type Model struct {
	screen          screen
	config          *app.Config
	metadataStore   *storage.MetadataStore
	mainMenu        list.Model
	libraryList     list.Model
	processedList   list.Model
	singlePaperList list.Model
	selectedPaper   string
	width           int
	height          int
	err             error
	processing      bool
	processingMsg   string
}

// Item represents a menu item
type item struct {
	title       string
	description string
	action      string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.title }

// Custom key bindings
type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Back   key.Binding
	Quit   key.Binding
	Help   key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF06B7")).
			Background(lipgloss.Color("#1a1a1a")).
			Padding(0, 1).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Width(80).
			Align(lipgloss.Center)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Padding(1, 0)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2).
			Width(80)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF06B7")).
				Bold(true)
)

// createStyledDelegate creates a consistently styled list delegate
func createStyledDelegate() list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#FF06B7")).
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("#FF06B7")).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#7D56F4")).
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("#FF06B7"))
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(lipgloss.Color("#FAFAFA"))
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.
		Foreground(lipgloss.Color("#626262"))
	return delegate
}

// InitialModel creates a new TUI model
func InitialModel(configPath string) (*Model, error) {
	// Load config
	config, err := app.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize metadata store
	metadataStore, err := storage.NewMetadataStore(config.MetadataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load metadata: %w", err)
	}

	// Create main menu items
	items := []list.Item{
		item{
			title:       "📚 View All Papers in Library",
			description: "Browse all PDF files in the lib folder",
			action:      "view_library",
		},
		item{
			title:       "✅ View Processed Papers",
			description: "See papers that have been processed",
			action:      "view_processed",
		},
		item{
			title:       "📄 Process Single Paper",
			description: "Select and process one paper",
			action:      "process_single",
		},
		item{
			title:       "🚀 Process All Papers",
			description: "Process all papers in the lib folder",
			action:      "process_all",
		},
	}

	// Create main menu list with styled delegate
	delegate := createStyledDelegate()
	mainMenu := list.New(items, delegate, 0, 0)
	mainMenu.Title = "Archivist - Research Paper Helper"
	mainMenu.SetShowStatusBar(false)
	mainMenu.SetFilteringEnabled(false)
	mainMenu.Styles.Title = titleStyle
	mainMenu.Styles.TitleBar = titleStyle

	m := &Model{
		screen:        screenMain,
		config:        config,
		metadataStore: metadataStore,
		mainMenu:      mainMenu,
	}

	return m, nil
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		h := msg.Height - 8
		w := msg.Width - 4

		// Update size based on current screen
		switch m.screen {
		case screenMain:
			m.mainMenu.SetSize(w, h)
		case screenViewLibrary:
			m.libraryList.SetSize(w, h)
		case screenViewProcessed:
			m.processedList.SetSize(w, h)
		case screenSelectPaper:
			m.singlePaperList.SetSize(w, h)
		}

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.screen == screenMain {
				return m, tea.Quit
			}
			// On other screens, go back
			m.screen = screenMain
			return m, nil

		case "esc", "backspace":
			if m.screen != screenMain {
				m.screen = screenMain
				return m, nil
			}

		case "enter":
			return m.handleEnter()
		}
	}

	// Update the current list
	var cmd tea.Cmd
	switch m.screen {
	case screenMain:
		m.mainMenu, cmd = m.mainMenu.Update(msg)
	case screenViewLibrary:
		m.libraryList, cmd = m.libraryList.Update(msg)
	case screenViewProcessed:
		m.processedList, cmd = m.processedList.Update(msg)
	case screenSelectPaper:
		m.singlePaperList, cmd = m.singlePaperList.Update(msg)
	}

	return m, cmd
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
		case "view_library":
			m.screen = screenViewLibrary
			m.loadLibraryPapers()
		case "view_processed":
			m.screen = screenViewProcessed
			m.loadProcessedPapers()
		case "process_single":
			m.screen = screenSelectPaper
			m.loadPapersForSelection()
		case "process_all":
			// Exit TUI and trigger batch processing
			m.processing = true
			m.processingMsg = "batch"
			return m, tea.Quit
		}
	} else if m.screen == screenSelectPaper {
		// Handle paper selection
		selectedItem := m.singlePaperList.SelectedItem()
		if selectedItem != nil {
			m.selectedPaper = selectedItem.(item).description
			// TODO: Start processing
			return m, tea.Quit
		}
	}

	return m, nil
}

// loadLibraryPapers loads all papers from lib folder
func (m *Model) loadLibraryPapers() {
	files, err := fileutil.GetPDFFiles(m.config.InputDir)
	if err != nil {
		m.err = err
		return
	}

	items := make([]list.Item, len(files))
	for i, file := range files {
		basename := filepath.Base(file)

		// Check if processed
		hash, _ := fileutil.ComputeFileHash(file)
		status := "🔴 Unprocessed"
		if m.metadataStore.IsProcessed(hash) {
			status = "✅ Processed"
		}

		items[i] = item{
			title:       basename,
			description: fmt.Sprintf("%s • %s", status, file),
			action:      file,
		}
	}

	delegate := createStyledDelegate()
	m.libraryList = list.New(items, delegate, 0, 0)
	m.libraryList.Title = fmt.Sprintf("📚 Library Papers (%d total)", len(files))
	m.libraryList.SetShowStatusBar(false)
	m.libraryList.Styles.Title = titleStyle
	if m.width > 0 && m.height > 0 {
		m.libraryList.SetSize(m.width-4, m.height-8)
	}
}

// loadProcessedPapers loads processed papers
func (m *Model) loadProcessedPapers() {
	records := m.metadataStore.GetAllRecords()

	items := make([]list.Item, len(records))
	for i, record := range records {
		statusIcon := "✅"
		if record.Status == storage.StatusFailed {
			statusIcon = "❌"
		} else if record.Status == storage.StatusProcessing {
			statusIcon = "⏳"
		}

		items[i] = item{
			title:       record.PaperTitle,
			description: fmt.Sprintf("%s %s • Processed: %s", statusIcon, record.Status, record.ProcessedAt.Format("2006-01-02 15:04")),
			action:      record.FilePath,
		}
	}

	delegate := createStyledDelegate()
	m.processedList = list.New(items, delegate, 0, 0)
	m.processedList.Title = fmt.Sprintf("✅ Processed Papers (%d total)", len(records))
	m.processedList.SetShowStatusBar(false)
	m.processedList.Styles.Title = titleStyle
	if m.width > 0 && m.height > 0 {
		m.processedList.SetSize(m.width-4, m.height-8)
	}
}

// loadPapersForSelection loads papers for single selection
func (m *Model) loadPapersForSelection() {
	files, err := fileutil.GetPDFFiles(m.config.InputDir)
	if err != nil {
		m.err = err
		return
	}

	items := make([]list.Item, 0)
	for _, file := range files {
		basename := filepath.Base(file)

		// Check if processed
		hash, _ := fileutil.ComputeFileHash(file)
		if m.metadataStore.IsProcessed(hash) {
			continue // Skip already processed papers
		}

		items = append(items, item{
			title:       basename,
			description: file,
			action:      file,
		})
	}

	delegate := createStyledDelegate()
	m.singlePaperList = list.New(items, delegate, 0, 0)
	m.singlePaperList.Title = fmt.Sprintf("📄 Select Paper to Process (%d unprocessed)", len(items))
	m.singlePaperList.SetShowStatusBar(false)
	m.singlePaperList.Styles.Title = titleStyle
	if m.width > 0 && m.height > 0 {
		m.singlePaperList.SetSize(m.width-4, m.height-8)
	}
}

// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content string

	// Header
	header := headerStyle.Render("🎓 ARCHIVIST - Research Paper Helper")

	switch m.screen {
	case screenMain:
		content = m.mainMenu.View()
	case screenViewLibrary:
		content = m.libraryList.View()
	case screenViewProcessed:
		content = m.processedList.View()
	case screenSelectPaper:
		if len(m.singlePaperList.Items()) == 0 {
			content = warningStyle.Render("\n⚠️  No unprocessed papers found in library\n\n") +
				helpStyle.Render("Press ESC to go back")
		} else {
			content = m.singlePaperList.View()
		}
	}

	// Footer with help
	help := helpStyle.Render(m.getHelp())

	return fmt.Sprintf("%s\n\n%s\n\n%s", header, content, help)
}

// getHelp returns context-appropriate help text
func (m Model) getHelp() string {
	if m.screen == screenMain {
		return "↑/↓: Navigate • Enter: Select • Q: Quit"
	}
	return "↑/↓: Navigate • Enter: Select • ESC: Back • Q: Quit"
}

// Run starts the TUI application
func Run(configPath string) error {
	m, err := InitialModel(configPath)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()

	if err != nil {
		return err
	}

	// Handle post-TUI actions
	finalM := finalModel.(Model)

	if finalM.processing && finalM.processingMsg == "batch" {
		return handleBatchProcessing(finalM.config)
	}

	if finalM.selectedPaper != "" {
		return handleSinglePaperProcessing(finalM.selectedPaper, finalM.config)
	}

	return nil
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
		ui.PrintWarning("Mode selection cancelled")
		return err
	}

	// Apply mode configuration
	applyModeConfig(config, selectedMode)
	ui.ShowModeDetails(selectedMode)

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
		return err
	}

	return nil
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
		ui.PrintWarning("Mode selection cancelled")
		return err
	}

	// Apply mode configuration
	applyModeConfig(config, selectedMode)
	ui.ShowModeDetails(selectedMode)

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
		return err
	}

	return nil
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
		config.Gemini.Agentic.Stages.MethodologyAnalysis.Model = "gemini-2.5-pro"
	} else {
		config.Gemini.Agentic.Stages.MethodologyAnalysis.Model = "gemini-2.0-flash"
	}
}
