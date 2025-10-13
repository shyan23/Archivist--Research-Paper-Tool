package tui

import (
	"fmt"
)

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

	// Footer with help (add Ctrl+P hint)
	helpText := m.getHelp()
	if !m.commandPalette.active {
		helpText += " • Ctrl+P: Command Palette"
	}
	help := helpStyle.Render(helpText)

	baseView := fmt.Sprintf("%s\n\n%s\n\n%s", header, content, help)

	// Overlay command palette if active
	if m.commandPalette.active {
		palette := m.commandPalette.View()
		// Simple overlay - palette appears on top
		return baseView + "\n\n" + palette
	}

	return baseView
}
