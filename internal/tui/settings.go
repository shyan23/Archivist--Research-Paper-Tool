package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// loadSettingsMenu loads the settings menu
func (m *Model) loadSettingsMenu() {
	// Shorten paths for display
	inputDisplay := m.config.InputDir
	if len(inputDisplay) > 30 {
		inputDisplay = "..." + inputDisplay[len(inputDisplay)-27:]
	}

	outputDisplay := m.config.ReportOutputDir
	if len(outputDisplay) > 30 {
		outputDisplay = "..." + outputDisplay[len(outputDisplay)-27:]
	}

	items := []list.Item{
		item{
			title:       "📁 Directory Settings",
			description: fmt.Sprintf("📥 Input: %s | 📤 Output: %s", inputDisplay, outputDisplay),
			action:      "directory_settings",
		},
		item{
			title:       "🔙 Back to Main Menu",
			description: "Return to the main menu",
			action:      "main_menu",
		},
	}

	delegate := createStyledDelegate()
	m.settingsMenu = list.New(items, delegate, 0, 0)
	m.settingsMenu.Title = "⚙️  Settings"
	m.settingsMenu.SetShowStatusBar(false)
	m.settingsMenu.SetFilteringEnabled(false)
	m.settingsMenu.Styles.Title = titleStyle
}

// loadDirectorySettingsMenu loads the directory settings menu
func (m *Model) loadDirectorySettingsMenu() {
	// Check if directories exist
	inputExists := "✅"
	if _, err := os.Stat(m.config.InputDir); os.IsNotExist(err) {
		inputExists = "⚠️ "
	}

	outputExists := "✅"
	if _, err := os.Stat(m.config.ReportOutputDir); os.IsNotExist(err) {
		outputExists = "⚠️ "
	}

	items := []list.Item{
		item{
			title:       "📥 Browse Input Directory",
			description: fmt.Sprintf("%s %s (Visual folder browser)", inputExists, m.config.InputDir),
			action:      "browse_input_dir",
		},
		item{
			title:       "✏️  Type Input Directory Path",
			description: "Manually type or paste the folder path",
			action:      "type_input_dir",
		},
		item{
			title:       "📤 Browse Output Directory",
			description: fmt.Sprintf("%s %s (Visual folder browser)", outputExists, m.config.ReportOutputDir),
			action:      "browse_output_dir",
		},
		item{
			title:       "✏️  Type Output Directory Path",
			description: "Manually type or paste the folder path",
			action:      "type_output_dir",
		},
		item{
			title:       "✨ Create Missing Directories",
			description: "Create directories if they don't exist",
			action:      "create_directories",
		},
		item{
			title:       "🔙 Back",
			description: "Return to settings menu",
			action:      "back",
		},
	}

	delegate := createStyledDelegate()
	m.directorySettingsMenu = list.New(items, delegate, 0, 0)
	m.directorySettingsMenu.Title = "📁 Directory Configuration"
	m.directorySettingsMenu.SetShowStatusBar(false)
	m.directorySettingsMenu.SetFilteringEnabled(false)
	m.directorySettingsMenu.Styles.Title = titleStyle
}

// handleDirectoryInput handles file browser navigation or text input
func (m Model) handleDirectoryInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If file browser is active, use its handler
	if m.fileBrowserActive {
		return m.handleFileBrowserInput(msg)
	}

	// Handle text input for typing paths
	if m.directoryInputMode != "" && !m.fileBrowserActive {
		return m.handleDirectoryTextInput(msg)
	}

	return m, nil
}

// handleDirectoryTextInput handles typing directory paths
func (m Model) handleDirectoryTextInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Apply the typed path
		if m.directoryInput != "" {
			// Expand home directory
			path := m.directoryInput
			if len(path) >= 2 && path[:2] == "~/" {
				home, _ := os.UserHomeDir()
				path = filepath.Join(home, path[2:])
			}

			// Convert to absolute path
			absPath, err := filepath.Abs(path)
			if err == nil {
				// Create directory if it doesn't exist
				os.MkdirAll(absPath, 0755)

				// Apply the change
				if m.directoryInputMode == "input_dir" {
					m.config.InputDir = absPath
					m.directoryChanged = true
				} else if m.directoryInputMode == "output_dir" {
					m.config.ReportOutputDir = absPath
					m.directoryChanged = true
				}

				// Save preferences immediately
				saveDirectoryPreferences(m.config.InputDir, m.config.ReportOutputDir)
			}
		}

		// Reset and reload menu
		m.directoryInput = ""
		m.directoryInputMode = ""
		m.loadDirectorySettingsMenu()
		if m.width > 0 && m.height > 0 {
			m.directorySettingsMenu.SetSize(m.width-4, m.height-8)
		}
		return m, nil

	case "esc":
		// Cancel input
		m.directoryInput = ""
		m.directoryInputMode = ""
		m.loadDirectorySettingsMenu()
		if m.width > 0 && m.height > 0 {
			m.directorySettingsMenu.SetSize(m.width-4, m.height-8)
		}
		return m, nil

	case "backspace":
		if len(m.directoryInput) > 0 {
			m.directoryInput = m.directoryInput[:len(m.directoryInput)-1]
		}

	case "ctrl+c":
		return m, tea.Quit

	default:
		// Add character to input
		if len(msg.String()) == 1 {
			m.directoryInput += msg.String()
		} else if msg.Type == tea.KeySpace {
			m.directoryInput += " "
		}
	}

	return m, nil
}

// renderDirectorySettingsScreen renders the directory settings screen
func (m Model) renderDirectorySettingsScreen() string {
	// If file browser is active, show the file browser
	if m.fileBrowserActive {
		return m.renderFileBrowser()
	}

	// If typing mode is active, show text input screen
	if m.directoryInputMode != "" && !m.fileBrowserActive {
		return m.renderDirectoryTextInput()
	}

	// Otherwise show the directory settings menu
	return m.directorySettingsMenu.View()
}

// renderDirectoryTextInput renders the text input screen for directory path
func (m Model) renderDirectoryTextInput() string {
	var title string
	var emoji string

	if m.directoryInputMode == "input_dir" {
		title = "Type Input Directory Path"
		emoji = "📥"
	} else {
		title = "Type Output Directory Path"
		emoji = "📤"
	}

	titleBox := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(1, 2).
		Width(100).
		Align(lipgloss.Center).
		Render(emoji + " " + title)

	instructionBox := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true).
		Padding(0, 2).
		Render("Type or paste the full directory path below:")

	inputValue := m.directoryInput
	if inputValue == "" {
		inputValue = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Italic(true).
			Render("e.g., /home/user/papers or ~/Documents/research")
	}

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF06B7")).
		Padding(0, 2).
		Width(96).
		Render(inputValue + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF06B7")).
			Bold(true).
			Render("│"))

	tips := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true).
		Render("💡 Tips:")

	tipsList := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Padding(0, 2).
		Render(`• Absolute paths: /home/user/papers or /data/research
• Home directory: ~/Documents/papers or ~/research
• Relative paths: ./my-papers or ../papers
• Spaces are OK: /home/user/my research papers
• Press Enter to save, ESC to cancel`)

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		Padding(1, 0).
		Render("Enter: Save & Create Directory • ESC: Cancel • Ctrl+C: Quit")

	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n%s\n\n%s",
		titleBox,
		instructionBox,
		inputBox,
		tips,
		tipsList,
		help,
	)
}

// handleSettingsEnter handles enter key in settings screens
func (m Model) handleSettingsEnter() (tea.Model, tea.Cmd) {
	if m.screen == screenSettings {
		selected := m.settingsMenu.SelectedItem()
		if selected != nil {
			action := selected.(item).action
			switch action {
			case "directory_settings":
				m.navigateTo(screenDirectorySettings)
				m.loadDirectorySettingsMenu()
				// Ensure menu is sized if we have dimensions
				if m.width > 0 && m.height > 0 {
					m.directorySettingsMenu.SetSize(m.width-4, m.height-8)
				}
			case "main_menu":
				m.screen = screenMain
				m.screenHistory = []screen{}
			}
		}
	} else if m.screen == screenDirectorySettings && !m.fileBrowserActive {
		selected := m.directorySettingsMenu.SelectedItem()
		if selected != nil {
			action := selected.(item).action
			switch action {
			case "browse_input_dir":
				// Launch file browser starting from current input directory
				m.directoryInputMode = "input_dir"
				m.initFileBrowser(m.config.InputDir)
			case "type_input_dir":
				// Switch to text input mode for input directory
				m.directoryInputMode = "input_dir"
				m.directoryInput = m.config.InputDir
			case "browse_output_dir":
				// Launch file browser starting from current output directory
				m.directoryInputMode = "output_dir"
				m.initFileBrowser(m.config.ReportOutputDir)
			case "type_output_dir":
				// Switch to text input mode for output directory
				m.directoryInputMode = "output_dir"
				m.directoryInput = m.config.ReportOutputDir
			case "create_directories":
				// Create missing directories
				os.MkdirAll(m.config.InputDir, 0755)
				os.MkdirAll(m.config.ReportOutputDir, 0755)
				m.directoryChanged = true
				m.loadDirectorySettingsMenu() // Reload to show checkmarks
			case "back":
				if m.directoryChanged {
					// Save preferences before going back
					saveDirectoryPreferences(m.config.InputDir, m.config.ReportOutputDir)
					// Reload settings menu to show updated paths
					m.loadSettingsMenu()
				}
				m.navigateBack()
			}
		}
	}
	return m, nil
}
