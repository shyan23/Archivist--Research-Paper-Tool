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

// getLoadingSpinner returns different spinner frames
func getLoadingSpinner(frame int) string {
	spinners := []string{
		"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
	}
	return spinners[frame%len(spinners)]
}

// getLoadingMessage returns fun loading messages
func getLoadingMessage(frame int) string {
	messages := []string{
		"🔍 Searching the depths of arXiv...",
		"📚 Consulting the ancient scrolls of OpenReview...",
		"🎓 Browsing through ACL Anthology...",
		"🤖 Asking the AI overlords for papers...",
		"✨ Summoning research papers from the void...",
		"🧙 Casting search spells...",
		"🚀 Launching paper-seeking rockets...",
		"🔬 Analyzing quantum paper states...",
		"🌟 Gathering academic stardust...",
		"📖 Flipping through digital libraries...",
	}
	return messages[(frame/3)%len(messages)]
}

// renderLoadingAnimation renders a fun loading animation
func renderLoadingAnimation(frame int, query string) string {
	var sb strings.Builder

	spinner := getLoadingSpinner(frame)
	message := getLoadingMessage(frame)

	sb.WriteString("\n\n")
	sb.WriteString(titleStyle.Render("🔍 Searching for Papers") + "\n\n")

	// Search query
	sb.WriteString(successStyle.Render(fmt.Sprintf("Query: \"%s\"", query)) + "\n\n")

	// Animated spinner with message
	sb.WriteString(fmt.Sprintf("  %s  %s\n\n",
		highlightStyle.Render(spinner),
		infoStyle.Render(message)))

	// Progress bar (fake but fun!)
	progressWidth := 40
	progress := (frame % progressWidth)
	progressBar := strings.Repeat("━", progress) +
		highlightStyle.Render("●") +
		strings.Repeat("─", progressWidth-progress-1)

	sb.WriteString(fmt.Sprintf("  %s\n\n", progressBar))

	// Fun tips
	tips := []string{
		"💡 Tip: Use specific terms for better results",
		"💡 Tip: Try searching for author names too",
		"💡 Tip: Conference names work great (e.g., NeurIPS)",
		"💡 Tip: Add year to narrow down results",
		"💡 Tip: Technical terms give more precise results",
	}
	tip := tips[(frame/10)%len(tips)]
	sb.WriteString(helpStyle.Render(tip) + "\n")

	// Elapsed time indicator
	elapsed := (frame / 10) // Roughly seconds
	sb.WriteString(helpStyle.Render(fmt.Sprintf("\n⏱️  Searching... %d seconds", elapsed)))

	return sb.String()
}

// renderSimilarLoadingAnimation renders loading for similar search
func renderSimilarLoadingAnimation(frame int, factorCount int) string {
	var sb strings.Builder

	spinner := getLoadingSpinner(frame)
	message := getLoadingMessage(frame)

	sb.WriteString("\n\n")
	sb.WriteString(titleStyle.Render("🔍 Finding Similar Papers") + "\n\n")

	// Factor count
	sb.WriteString(successStyle.Render(fmt.Sprintf("Searching with %d factors", factorCount)) + "\n\n")

	// Animated spinner with message
	sb.WriteString(fmt.Sprintf("  %s  %s\n\n",
		highlightStyle.Render(spinner),
		infoStyle.Render(message)))

	// Progress bar
	progressWidth := 40
	progress := (frame % progressWidth)
	progressBar := strings.Repeat("━", progress) +
		highlightStyle.Render("●") +
		strings.Repeat("─", progressWidth-progress-1)

	sb.WriteString(fmt.Sprintf("  %s\n\n", progressBar))

	// Fun academic quotes
	quotes := []string{
		"📜 \"Standing on the shoulders of giants...\"",
		"📜 \"Knowledge is power...\"",
		"📜 \"Research is what I'm doing when I don't know...\"",
		"📜 \"The important thing is not to stop questioning...\"",
		"📜 \"In the middle of difficulty lies opportunity...\"",
	}
	quote := quotes[(frame/10)%len(quotes)]
	sb.WriteString(helpStyle.Render(quote) + "\n")

	return sb.String()
}
