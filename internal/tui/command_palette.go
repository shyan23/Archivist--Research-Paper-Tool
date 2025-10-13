package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Command represents a command in the palette
type Command struct {
	name        string
	description string
	action      string
	icon        string
}

func (c Command) Title() string       { return c.icon + " " + c.name }
func (c Command) Description() string { return c.description }
func (c Command) FilterValue() string { return c.name + " " + c.description }

// CommandPalette represents the command palette state
type CommandPalette struct {
	active    bool
	input     textinput.Model
	list      list.Model
	commands  []Command
	width     int
	height    int
}

// Styles for command palette
var (
	paletteContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FF06B7")).
				Padding(1, 2).
				MarginTop(2).
				MarginLeft(10).
				MarginRight(10)

	paletteInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Bold(true)

	paletteTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF06B7")).
				Bold(true).
				Padding(0, 1)
)

// NewCommandPalette creates a new command palette
func NewCommandPalette() CommandPalette {
	ti := textinput.New()
	ti.Placeholder = "Type to search commands..."
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 50

	commands := []Command{
		{name: "View Library", description: "Browse all papers in lib folder", action: "view_library", icon: "📚"},
		{name: "View Processed", description: "See successfully processed papers", action: "view_processed", icon: "✅"},
		{name: "Process Single Paper", description: "Select and process one paper", action: "process_single", icon: "📄"},
		{name: "Process All Papers", description: "Process entire library", action: "process_all", icon: "🚀"},
		{name: "Open Settings", description: "View current configuration", action: "settings", icon: "⚙️"},
		{name: "Clear Cache", description: "Clear Redis analysis cache", action: "clear_cache", icon: "🗑️"},
		{name: "View Cache Stats", description: "Show cache statistics", action: "cache_stats", icon: "📊"},
		{name: "Check Dependencies", description: "Verify LaTeX installation", action: "check_deps", icon: "🔍"},
		{name: "Main Menu", description: "Return to main menu", action: "main_menu", icon: "🏠"},
		{name: "Quit", description: "Exit Archivist", action: "quit", icon: "🚪"},
	}

	items := make([]list.Item, len(commands))
	for i, cmd := range commands {
		items[i] = cmd
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#FF06B7")).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#7D56F4"))

	l := list.New(items, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	return CommandPalette{
		active:   false,
		input:    ti,
		list:     l,
		commands: commands,
	}
}

// Toggle activates or deactivates the command palette
func (cp *CommandPalette) Toggle() {
	cp.active = !cp.active
	if cp.active {
		cp.input.Focus()
		cp.input.SetValue("")
		cp.updateFilter("")
	} else {
		cp.input.Blur()
	}
}

// updateFilter filters commands based on input
func (cp *CommandPalette) updateFilter(query string) {
	if query == "" {
		// Show all commands
		items := make([]list.Item, len(cp.commands))
		for i, cmd := range cp.commands {
			items[i] = cmd
		}
		cp.list.SetItems(items)
		return
	}

	// Filter commands
	query = strings.ToLower(query)
	filtered := []list.Item{}
	for _, cmd := range cp.commands {
		name := strings.ToLower(cmd.name)
		desc := strings.ToLower(cmd.description)
		if strings.Contains(name, query) || strings.Contains(desc, query) {
			filtered = append(filtered, cmd)
		}
	}
	cp.list.SetItems(filtered)
}

// Update handles command palette updates
func (cp *CommandPalette) Update(msg tea.Msg) tea.Cmd {
	if !cp.active {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			cp.Toggle()
			return nil
		case "enter":
			// Command will be handled by parent
			return nil
		case "up", "down":
			// Let list handle navigation
			var cmd tea.Cmd
			cp.list, cmd = cp.list.Update(msg)
			return cmd
		default:
			// Update input and filter
			var cmd tea.Cmd
			cp.input, cmd = cp.input.Update(msg)
			cp.updateFilter(cp.input.Value())
			return cmd
		}
	case tea.WindowSizeMsg:
		cp.width = msg.Width
		cp.height = msg.Height
		cp.list.SetSize(msg.Width-24, min(10, len(cp.list.Items())))
	}

	return nil
}

// View renders the command palette
func (cp CommandPalette) View() string {
	if !cp.active {
		return ""
	}

	title := paletteTitleStyle.Render("⌘ Command Palette")
	input := paletteInputStyle.Render(cp.input.View())
	listView := cp.list.View()

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		input,
		"",
		listView,
		"",
		helpStyle.Render("↑/↓: Navigate • Enter: Select • ESC: Close"),
	)

	return paletteContainerStyle.Render(content)
}

// GetSelectedAction returns the currently selected command action
func (cp CommandPalette) GetSelectedAction() string {
	if !cp.active {
		return ""
	}

	selected := cp.list.SelectedItem()
	if selected == nil {
		return ""
	}

	return selected.(Command).action
}

// SetSize updates the command palette size
func (cp *CommandPalette) SetSize(width, height int) {
	cp.width = width
	cp.height = height
	cp.list.SetSize(width-24, min(10, len(cp.list.Items())))
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
