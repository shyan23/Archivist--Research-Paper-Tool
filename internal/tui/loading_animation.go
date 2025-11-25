package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// LoadingTickMsg is sent periodically to update the loading animation
type LoadingTickMsg time.Time

// tickEvery returns a command that sends LoadingTickMsg every interval
func tickEvery(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return LoadingTickMsg(t)
	})
}

// getPokeballAnimation returns frames for a Pokeball capture animation
func getPokeballAnimation(frame int) string {
	// More explicit keyframe-like sequence for: three shakes -> close -> glow
	// We'll map frame -> phase, then render with pastel styles defined in styles.go
	// Sequence length: 18 frames (0..17)
	// 0..5  -> shakes (3 left-right cycles)
	// 6..9  -> closing frames
	// 10..17 -> success glow frames

	seq := frame % 18

	// offsets to simulate tilt (number of spaces to indent)
	offsets := []int{0, -3, 3, -3, 3, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	offset := offsets[seq]

	// Build the pokeball using lipgloss styles (pastel)
	// center glyph: hollow when loading, solid when success
	center := "◎"
	useGlow := false
	if seq >= 10 {
		center = "●"
		useGlow = true
	}

	// Lines for the pokeball
	topLine := pokeballPastelTop.Render("  _____  ")
	midLine := pokeballPastelTop.Render(" /     \\")
	btn := pokeballButton.Render(center)
	if useGlow {
		btn = pokeballGlow.Render(center)
	}
	centerLine := pokeballPastelBottom.Render("| " + btn + " |")
	bottomLine := pokeballPastelBottom.Render(" \\_____/ ")

	// optional sparkle line for success
	sparkle := ""
	if useGlow {
		sparkle = "  ✨\n"
	}

	// indent pad (keep minimum padding)
	basePad := 10
	pad := basePad + offset
	if pad < 0 {
		pad = 0
	}
	prefix := strings.Repeat(" ", pad)

	art := fmt.Sprintf("%s%s\n%s%s\n%s%s\n%s%s\n%s", prefix, topLine, prefix, midLine, prefix, centerLine, prefix, bottomLine, prefix+sparkle)
	return art
}

// getPokemonMessage returns fun, Pokemon-themed loading messages
func getPokemonMessage(frame int) string {
	messages := []string{
		"A wild research paper appeared!",
		"Throwing a Poké Ball...",
		"Using 'Search'... It's super effective!",
		"Trying to catch a rare publication...",
		"Consulting the Pokédex for paper details...",
		"I choose you, ArXiv!",
		"The paper is evolving!",
	}
	return messages[(frame/10)%len(messages)]
}

// renderLoadingAnimation renders a Pokemon-themed loading animation
func renderLoadingAnimation(frame int, query string) string {
	var sb strings.Builder

	animation := getPokeballAnimation(frame)
	message := getPokemonMessage(frame)

	sb.WriteString("\n\n")
	sb.WriteString(titleStyle.Render("Gotta Catch 'Em All!") + "\n\n")
	sb.WriteString(successStyle.Render(fmt.Sprintf("Searching for: %s", query)) + "\n\n")

	// Center the animation (animation already uses pastel styles)
	sb.WriteString(animation + "\n")
	sb.WriteString(infoStyle.Render(message) + "\n\n")

	elapsed := (frame / 10)
	sb.WriteString(helpStyle.Render(fmt.Sprintf("Searching... %ds", elapsed)))

	return sb.String()
}

// renderSimilarLoadingAnimation renders a similar Pokemon-themed animation
func renderSimilarLoadingAnimation(frame int, factorCount int) string {
	var sb strings.Builder

	animation := getPokeballAnimation(frame)
	message := getPokemonMessage(frame)

	sb.WriteString("\n\n")
	sb.WriteString(titleStyle.Render("Exploring Similar Papers...") + "\n\n")
	sb.WriteString(successStyle.Render(fmt.Sprintf("Following %d similar trails", factorCount)) + "\n\n")

	sb.WriteString(animation + "\n")
	sb.WriteString(infoStyle.Render(message) + "\n\n")

	quotes := []string{
		"The more I learn, the more I realize how much I don't know. - Albert Einstein",
		"A true master is an eternal student.",
		"Every great discovery starts as a question.",
	}
	quote := quotes[(frame/15)%len(quotes)]
	sb.WriteString(helpStyle.Render(quote) + "\n")

	return sb.String()
}
