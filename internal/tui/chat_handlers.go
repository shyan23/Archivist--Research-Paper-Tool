package tui

import (
	"context"
	"fmt"
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// handleChatPaperSelection handles paper selection for chat
func (m Model) handleChatPaperSelection() (tea.Model, tea.Cmd) {
	if m.screen != screenChatSelectPapers {
		return m, nil
	}

	// Similar to multi-select, but for chat
	if len(m.multiSelectIndexes) == 0 {
		return m, nil
	}

	// Collect selected papers
	m.chatSelectedPapers = []string{}
	for idx := range m.multiSelectIndexes {
		if idx < len(m.allPapersForSelect) {
			m.chatSelectedPapers = append(m.chatSelectedPapers, m.allPapersForSelect[idx])
		}
	}

	// Index papers if needed (before starting chat)
	log.Printf("🔍 Checking if papers need indexing...")
	ctx := context.Background()
	for _, paperTitle := range m.chatSelectedPapers {
		if err := indexPaperIfNeeded(ctx, m.config, paperTitle); err != nil {
			log.Printf("❌ Failed to index %s: %v", paperTitle, err)
			// Add error message to chat
			m.chatMessages = append(m.chatMessages, ChatMessage{
				Role:    "assistant",
				Content: fmt.Sprintf("⚠️  Failed to index paper '%s': %v", paperTitle, err),
			})
		}
	}

	// Navigate to chat screen
	m.navigateTo(screenChat)
	m.chatInput = ""
	if len(m.chatMessages) == 0 {
		m.chatMessages = []ChatMessage{}
	}
	m.chatLoading = false

	// Generate session ID
	m.chatSessionID = fmt.Sprintf("tui_session_%d", time.Now().UnixNano())

	return m, nil
}

// handleChatSpacebar toggles selection in chat paper selection mode
func (m Model) handleChatSpacebar() (tea.Model, tea.Cmd) {
	if m.screen != screenChatSelectPapers {
		return m, nil
	}

	// Get current index
	idx := m.chatPaperList.Index()

	// Toggle selection
	if m.multiSelectIndexes[idx] {
		delete(m.multiSelectIndexes, idx)
	} else {
		m.multiSelectIndexes[idx] = true
	}

	// Update title with selection count
	m.chatPaperList.Title = fmt.Sprintf("💬 Select Papers to Chat About (Space to toggle, Enter to confirm) - %d selected", len(m.multiSelectIndexes))

	return m, nil
}

// handleChatInput handles text input in chat screen
func (m Model) handleChatInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.screen != screenChat {
		return m, nil
	}

	switch msg.String() {
	case "enter":
		// Send message
		if m.chatInput == "" || m.chatLoading {
			return m, nil
		}

		userMessage := m.chatInput

		// Add user message to history
		m.chatMessages = append(m.chatMessages, ChatMessage{
			Role:    "user",
			Content: userMessage,
		})

		// Clear input and set loading
		m.chatInput = ""
		m.chatLoading = true

		// Send message to chat engine
		return m, SendChatMessage(m.config, m.chatSessionID, userMessage, m.chatSelectedPapers)

	case "backspace":
		if len(m.chatInput) > 0 {
			m.chatInput = m.chatInput[:len(m.chatInput)-1]
		}

	case " ":
		m.chatInput += " "

	default:
		// Regular character input
		if len(msg.String()) == 1 {
			m.chatInput += msg.String()
		}
	}

	return m, nil
}

// handleChatResponse handles response from chat engine
func (m Model) handleChatResponse(msg ChatResponseMsg) (tea.Model, tea.Cmd) {
	m.chatLoading = false

	if msg.Err != nil {
		// Add error message
		m.chatMessages = append(m.chatMessages, ChatMessage{
			Role:    "assistant",
			Content: fmt.Sprintf("❌ Error: %v", msg.Err),
		})
		return m, nil
	}

	// Add assistant message
	m.chatMessages = append(m.chatMessages, ChatMessage{
		Role:      "assistant",
		Content:   msg.Message.Content,
		Citations: msg.Message.Citations,
	})

	return m, nil
}
