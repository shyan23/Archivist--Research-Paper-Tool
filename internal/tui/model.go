package tui

import (
	"archivist/internal/app"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// InitialModel creates a new TUI model
func InitialModel(configPath string) (*Model, error) {
	// Load config
	config, err := app.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
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
			description: "See generated reports from reports folder",
			action:      "view_processed",
		},
		item{
			title:       "📄 Process Single Paper",
			description: "Select and process one paper",
			action:      "process_single",
		},
		item{
			title:       "📋 Process Multiple Papers",
			description: "Select multiple papers to process (use spacebar to select)",
			action:      "process_multiple",
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
		screen:             screenMain,
		config:             config,
		mainMenu:           mainMenu,
		commandPalette:     NewCommandPalette(),
		multiSelectIndexes: make(map[int]bool),
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
		case screenSelectMultiplePapers:
			m.multiPaperList.SetSize(w, h)
		}

		return m, nil

	case tea.KeyMsg:
		// Handle command palette toggle (Ctrl+P)
		if msg.String() == "ctrl+p" {
			m.commandPalette.Toggle()
			if m.commandPalette.active {
				m.commandPalette.SetSize(m.width, m.height)
			}
			return m, nil
		}

		// If command palette is active, let it handle the input
		if m.commandPalette.active {
			if msg.String() == "enter" {
				// Execute selected command
				action := m.commandPalette.GetSelectedAction()
				m.commandPalette.Toggle() // Close palette
				return m.executeCommand(action)
			}
			cmd := m.commandPalette.Update(msg)
			return m, cmd
		}

		// Normal key handling
		switch msg.String() {
		case "ctrl+c", "q":
			if m.screen == screenMain {
				return m, tea.Quit
			}
			// On other screens, go back to main
			m.navigateBack()
			return m, nil

		case "esc", "backspace":
			if m.screen != screenMain {
				m.navigateBack()
				return m, nil
			}

		case " ": // Spacebar for multi-select
			if m.screen == screenSelectMultiplePapers {
				return m.handleSpacebar()
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
	case screenSelectMultiplePapers:
		m.multiPaperList, cmd = m.multiPaperList.Update(msg)
	}

	return m, cmd
}

// executeCommand executes a command from the command palette
func (m Model) executeCommand(action string) (tea.Model, tea.Cmd) {
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
	case "process_multiple":
		m.navigateTo(screenSelectMultiplePapers)
		m.loadPapersForMultiSelection()
	case "process_all":
		m.processing = true
		m.processingMsg = "batch"
		return m, tea.Quit
	case "main_menu":
		m.screen = screenMain
		m.screenHistory = []screen{} // Clear history
	case "quit":
		return m, tea.Quit
	case "settings", "clear_cache", "cache_stats", "check_deps":
		// These will be implemented as external commands
		m.processingMsg = action
		return m, tea.Quit
	}
	return m, nil
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

	// Handle multiple papers selected
	if len(finalM.selectedPapers) > 0 {
		return handleMultiplePapersProcessing(finalM.selectedPapers, finalM.config)
	}

	if finalM.selectedPaper != "" {
		switch finalM.processingMsg {
		case "open_pdf", "open_report":
			return handleOpenPDF(finalM.selectedPaper)
		default:
			return handleSinglePaperProcessing(finalM.selectedPaper, finalM.config)
		}
	}

	return nil
}
