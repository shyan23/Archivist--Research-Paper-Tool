package tui

import (
	"archivist/internal/app"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
)

// Screen types
type screen int

const (
	screenMain screen = iota
	screenViewLibrary
	screenViewProcessed
	screenSelectPaper
	screenSelectMultiplePapers
	screenProcessing
)

// Model represents the TUI application state
type Model struct {
	screen             screen
	screenHistory      []screen // Navigation stack for back button
	config             *app.Config
	mainMenu           list.Model
	libraryList        list.Model
	processedList      list.Model
	singlePaperList    list.Model
	multiPaperList     list.Model
	commandPalette     CommandPalette
	selectedPaper      string
	selectedPapers     []string          // For multi-select
	multiSelectIndexes map[int]bool      // Track selected indices
	allPapersForSelect []string          // Store all available papers
	width              int
	height             int
	err                error
	processing         bool
	processingMsg      string
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
