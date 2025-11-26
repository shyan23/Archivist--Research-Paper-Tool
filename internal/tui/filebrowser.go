package tui

import (
	"archivist/internal/app"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// saveDirectoryPreferences saves directory preferences to disk
func saveDirectoryPreferences(inputDir, outputDir string) {
	_ = app.UpdateDirectories(inputDir, outputDir)
}

// initFileBrowser initializes the file browser with a starting path
func (m *Model) initFileBrowser(startPath string) {
	m.fileBrowserActive = true

	// Expand and clean the path
	if startPath == "" {
		startPath = "."
	}

	// Expand home directory
	if strings.HasPrefix(startPath, "~/") {
		home, _ := os.UserHomeDir()
		startPath = filepath.Join(home, startPath[2:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		absPath = startPath
	}

	m.currentBrowserPath = absPath
	m.browserSelectedIndex = 0
	m.browserShowHidden = false
	m.loadBrowserItems()
}

// loadBrowserItems loads items in the current browser directory
func (m *Model) loadBrowserItems() {
	m.browserItems = []string{}

	// Always add parent directory option
	if m.currentBrowserPath != "/" {
		m.browserItems = append(m.browserItems, "..")
	}

	// Read directory contents
	entries, err := os.ReadDir(m.currentBrowserPath)
	if err != nil {
		// If we can't read, go to home directory
		home, _ := os.UserHomeDir()
		m.currentBrowserPath = home
		entries, _ = os.ReadDir(m.currentBrowserPath)
	}

	// Separate folders and files
	var folders []string
	var files []string

	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files unless showHidden is true
		if !m.browserShowHidden && strings.HasPrefix(name, ".") {
			continue
		}

		if entry.IsDir() {
			folders = append(folders, name)
		} else {
			files = append(files, name)
		}
	}

	// Sort folders and files
	sort.Strings(folders)
	sort.Strings(files)

	// Add folders first, then files
	m.browserItems = append(m.browserItems, folders...)
	m.browserItems = append(m.browserItems, files...)

	// Ensure selected index is valid
	if m.browserSelectedIndex >= len(m.browserItems) {
		m.browserSelectedIndex = len(m.browserItems) - 1
	}
	if m.browserSelectedIndex < 0 {
		m.browserSelectedIndex = 0
	}
}

// handleFileBrowserInput handles keyboard input in file browser
func (m Model) handleFileBrowserInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.browserSelectedIndex > 0 {
			m.browserSelectedIndex--
		}

	case "down", "j":
		if m.browserSelectedIndex < len(m.browserItems)-1 {
			m.browserSelectedIndex++
		}

	case "enter":
		// Select current item
		if len(m.browserItems) == 0 {
			return m, nil
		}

		selected := m.browserItems[m.browserSelectedIndex]

		if selected == ".." {
			// Go to parent directory
			m.currentBrowserPath = filepath.Dir(m.currentBrowserPath)
			m.browserSelectedIndex = 0
			m.loadBrowserItems()
		} else {
			// Check if it's a directory
			fullPath := filepath.Join(m.currentBrowserPath, selected)
			info, err := os.Stat(fullPath)

			if err == nil && info.IsDir() {
				// Navigate into directory
				m.currentBrowserPath = fullPath
				m.browserSelectedIndex = 0
				m.loadBrowserItems()
			} else {
				// It's a file, ignore (we only select directories)
			}
		}

	case "s":
		// Select current directory
		m.fileBrowserActive = false

		// Apply the selected directory
		if m.directoryInputMode == "input_dir" {
			m.config.InputDir = m.currentBrowserPath
			m.directoryChanged = true
		} else if m.directoryInputMode == "output_dir" {
			m.config.ReportOutputDir = m.currentBrowserPath
			m.directoryChanged = true
		}

		// Save preferences immediately
		saveDirectoryPreferences(m.config.InputDir, m.config.ReportOutputDir)

		// Reset and reload menu
		m.directoryInputMode = ""
		m.loadDirectorySettingsMenu()

	case "n":
		// Create new folder
		// We'll add this in a future update

	case "h":
		// Toggle hidden files
		m.browserShowHidden = !m.browserShowHidden
		m.loadBrowserItems()

	case "g":
		// Go to home directory
		home, _ := os.UserHomeDir()
		m.currentBrowserPath = home
		m.browserSelectedIndex = 0
		m.loadBrowserItems()

	case "r":
		// Refresh
		m.loadBrowserItems()

	case "esc":
		// Cancel and return to settings
		m.fileBrowserActive = false
		m.directoryInputMode = ""
		m.loadDirectorySettingsMenu()

	case "ctrl+c":
		return m, tea.Quit
	}

	return m, nil
}

// renderFileBrowser renders the file browser interface
func (m Model) renderFileBrowser() string {
	var title string
	var emoji string

	if m.directoryInputMode == "input_dir" {
		title = "Select Input Directory (Papers Source)"
		emoji = "📥"
	} else {
		title = "Select Output Directory (Reports Destination)"
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

	// Current path display
	pathLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true).
		Render("Current Path:")

	pathValue := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true).
		Render(m.currentBrowserPath)

	pathBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(0, 2).
		Width(96).
		Render(pathLabel + " " + pathValue)

	// Directory listing
	var itemsDisplay strings.Builder

	maxDisplay := 15 // Show up to 15 items
	startIdx := 0
	endIdx := len(m.browserItems)

	// Scroll to keep selected item visible
	if m.browserSelectedIndex >= maxDisplay {
		startIdx = m.browserSelectedIndex - maxDisplay + 1
	}
	if endIdx > startIdx+maxDisplay {
		endIdx = startIdx + maxDisplay
	}

	for i := startIdx; i < endIdx && i < len(m.browserItems); i++ {
		item := m.browserItems[i]

		// Determine if it's a directory
		isDir := false
		icon := "📄"

		if item == ".." {
			isDir = true
			icon = "⬆️ "
		} else {
			fullPath := filepath.Join(m.currentBrowserPath, item)
			info, err := os.Stat(fullPath)
			if err == nil && info.IsDir() {
				isDir = true
				icon = "📁"
			}
		}

		// Style the item
		var itemText string
		if i == m.browserSelectedIndex {
			// Selected item - highlighted
			if isDir {
				itemText = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#FFFFFF")).
					Background(lipgloss.Color("#FF06B7")).
					Bold(true).
					Padding(0, 1).
					Width(94).
					Render(fmt.Sprintf("▶ %s %s", icon, item))
			} else {
				itemText = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#626262")).
					Background(lipgloss.Color("#3a3a3a")).
					Padding(0, 1).
					Width(94).
					Render(fmt.Sprintf("  %s %s (file - not selectable)", icon, item))
			}
		} else {
			// Normal item
			if isDir {
				itemText = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#FAFAFA")).
					Padding(0, 1).
					Width(94).
					Render(fmt.Sprintf("  %s %s", icon, item))
			} else {
				itemText = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#626262")).
					Padding(0, 1).
					Width(94).
					Render(fmt.Sprintf("  %s %s", icon, item))
			}
		}

		itemsDisplay.WriteString(itemText + "\n")
	}

	listingBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF06B7")).
		Padding(1, 1).
		Width(96).
		Height(18).
		Render(itemsDisplay.String())

	// Keyboard shortcuts help
	shortcuts := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true).
		Render("⌨️  Keyboard Shortcuts:")

	shortcutsList := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Render(
			`  ↑/k: Up  ↓/j: Down  Enter: Open folder  S: Select this directory
  H: Toggle hidden  G: Go home  R: Refresh  ESC: Cancel`)

	helpBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#04B575")).
		Padding(0, 2).
		Width(96).
		Render(shortcuts + "\n" + shortcutsList)

	// Assemble everything
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
		titleBox,
		pathBox,
		listingBox,
		helpBox,
	)
}
