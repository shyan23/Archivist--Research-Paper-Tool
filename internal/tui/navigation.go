package tui

// navigateTo pushes current screen to history and navigates to new screen
func (m *Model) navigateTo(newScreen screen) {
	// Only push to history if we're moving to a different screen
	if m.screen != newScreen {
		m.screenHistory = append(m.screenHistory, m.screen)
	}
	m.screen = newScreen
}

// navigateBack pops the last screen from history and goes back
func (m *Model) navigateBack() {
	if len(m.screenHistory) > 0 {
		// Pop the last screen from history
		lastScreen := m.screenHistory[len(m.screenHistory)-1]
		m.screenHistory = m.screenHistory[:len(m.screenHistory)-1]
		m.screen = lastScreen
	} else {
		// If no history, go to main screen
		m.screen = screenMain
	}
}

// getHelp returns context-appropriate help text
func (m Model) getHelp() string {
	switch m.screen {
	case screenMain:
		return "↑/↓: Navigate • Enter: Select • Q: Quit"
	case screenViewLibrary:
		return "↑/↓: Navigate • Enter: Open PDF • ESC: Back • Q: Quit"
	case screenViewProcessed:
		return "↑/↓: Navigate • Enter: Open Report • ESC: Back • Q: Quit"
	case screenSelectPaper:
		return "↑/↓: Navigate • Enter: Process Paper • ESC: Back • Q: Quit"
	default:
		return "↑/↓: Navigate • Enter: Select • ESC: Back • Q: Quit"
	}
}
