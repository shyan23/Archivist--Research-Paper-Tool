package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	    titleStyle = lipgloss.NewStyle().
		    Bold(true).
		    Foreground(lipgloss.Color("#7B61FF")). // Purple theme (restored)
		    Background(lipgloss.Color("#1c1c1c")). // Dark background
			Padding(0, 1).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#588d9b")). // Muted Blue
			Bold(true).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#dcdcdc")). // Off-white
			Background(lipgloss.Color("#588d9b")). // Muted Blue
			Padding(0, 1).
			Width(80).
			Align(lipgloss.Center)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#588d9b")). // Muted Blue
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#d7c368")). // Muted Yellow
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c94a38")). // Muted Red
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ABB2BF")). // Soft Grey
			Padding(1, 0)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#588d9b")). // Muted Blue
			Padding(1, 2).
			Width(80)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7B61FF")). // Purple theme (restored)
				Bold(true)

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#588d9b")). // Muted Blue
			Padding(0, 1).
			Width(60)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7a9b58")). // Muted Green
			Bold(true)

	highlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1c1c1c")). // Dark background
			Background(lipgloss.Color("#d7c368")). // Muted Yellow
			Bold(true).
			Padding(0, 1)

	pokeballStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c94a38")). // Muted Red for Pokeball
			Bold(true)

	// Pastel Poké Ball styles (Matte & Peaceful palette)
	pastelSoftRed     = "#EE5E5E"
	pastelCreamyWhite = "#FDFBF7"
	pastelCharcoal    = "#333333"

	pokeballPastelTop = lipgloss.NewStyle().
				Foreground(lipgloss.Color(pastelCreamyWhite)).
				Background(lipgloss.Color(pastelSoftRed)).
				Padding(0, 2)

	pokeballPastelBottom = lipgloss.NewStyle().
				Foreground(lipgloss.Color(pastelCharcoal)).
				Background(lipgloss.Color(pastelCreamyWhite)).
				Padding(0, 2)

	pokeballButton = lipgloss.NewStyle().
			Foreground(lipgloss.Color(pastelCharcoal)).
			Background(lipgloss.Color(pastelCreamyWhite)).
			Bold(true).
			Padding(0, 1)

	pokeballGlow = lipgloss.NewStyle().
			Foreground(lipgloss.Color(pastelCreamyWhite)).
			Background(lipgloss.Color(pastelSoftRed)).
			Bold(true).
			Padding(0, 1)
)

// createStyledDelegate creates a consistently styled list delegate
func createStyledDelegate() list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#7B61FF")). // Purple theme (restored)
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("#7B61FF")). // Purple theme (restored)
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#588d9b")). // Muted Blue
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("#7B61FF")) // Purple theme (restored)
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(lipgloss.Color("#dcdcdc")) // Off-white
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.
		Foreground(lipgloss.Color("#ABB2BF")) // Soft Grey
	return delegate
}
