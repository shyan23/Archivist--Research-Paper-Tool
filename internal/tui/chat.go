package tui

import (
	"archivist/internal/analyzer"
	"archivist/internal/app"
	"archivist/internal/chat"
	"archivist/internal/rag"
	"archivist/pkg/fileutil"
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/redis/go-redis/v9"
)

// ChatResponseMsg is sent when a chat response is received
type ChatResponseMsg struct {
	Message *chat.Message
	Err     error
}

// InitChatSession initializes the chat session
func (m *Model) initChatSession() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Initialize Redis client (for chat history only)
		redisClient := redis.NewClient(&redis.Options{
			Addr:     m.config.Cache.Redis.Addr,
			Password: m.config.Cache.Redis.Password,
			DB:       m.config.Cache.Redis.DB,
		})
		defer redisClient.Close()

		// Test Redis connection
		if err := redisClient.Ping(ctx).Err(); err != nil {
			return ChatResponseMsg{Err: fmt.Errorf("Redis not available: %w", err)}
		}

		// Initialize RAG components with FAISS
		embedClient, err := rag.NewEmbeddingClient(m.config.Gemini.APIKey)
		if err != nil {
			return ChatResponseMsg{Err: fmt.Errorf("failed to create embedding client: %w", err)}
		}
		defer embedClient.Close()

		// Use FAISS vector store
		indexDir := filepath.Join(".metadata", "vector_index")
		vectorStore, err := rag.NewFAISSVectorStore(indexDir)
		if err != nil {
			return ChatResponseMsg{Err: fmt.Errorf("failed to create FAISS vector store: %w", err)}
		}

		retrievalConfig := rag.DefaultRetrievalConfig()
		retriever := rag.NewRetriever(vectorStore, embedClient, retrievalConfig)

		// Gemini client
		geminiClient, err := analyzer.NewGeminiClient(
			m.config.Gemini.APIKey,
			m.config.Gemini.Model,
			m.config.Gemini.Temperature,
			m.config.Gemini.MaxTokens,
		)
		if err != nil {
			return ChatResponseMsg{Err: fmt.Errorf("failed to create Gemini client: %w", err)}
		}
		defer geminiClient.Close()

		// Chat engine
		chatEngine := chat.NewChatEngine(retriever, geminiClient, redisClient)

		// Start session (chatSelectedPapers now contains paper titles, not paths)
		session, err := chatEngine.StartSession(ctx, m.chatSelectedPapers)
		if err != nil {
			return ChatResponseMsg{Err: err}
		}

		// Store session ID
		return ChatSessionStarted{SessionID: session.ID}
	}
}

// ChatSessionStarted indicates chat session has started
type ChatSessionStarted struct {
	SessionID string
}

// SendChatMessage sends a message to the chat engine
func SendChatMessage(config interface{}, sessionID string, userMessage string, selectedPapers []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		cfg, ok := config.(*app.Config)
		if !ok {
			return ChatResponseMsg{Err: fmt.Errorf("invalid config type")}
		}

		// Initialize components
		redisClient := redis.NewClient(&redis.Options{
			Addr:     cfg.Cache.Redis.Addr,
			Password: cfg.Cache.Redis.Password,
			DB:       cfg.Cache.Redis.DB,
		})
		defer redisClient.Close()

		embedClient, err := rag.NewEmbeddingClient(cfg.Gemini.APIKey)
		if err != nil {
			return ChatResponseMsg{Err: err}
		}
		defer embedClient.Close()

		// Use FAISS vector store
		indexDir := filepath.Join(".metadata", "vector_index")
		vectorStore, err := rag.NewFAISSVectorStore(indexDir)
		if err != nil {
			return ChatResponseMsg{Err: err}
		}

		retrievalConfig := rag.DefaultRetrievalConfig()
		retriever := rag.NewRetriever(vectorStore, embedClient, retrievalConfig)

		geminiClient, err := analyzer.NewGeminiClient(
			cfg.Gemini.APIKey,
			cfg.Gemini.Model,
			cfg.Gemini.Temperature,
			cfg.Gemini.MaxTokens,
		)
		if err != nil {
			return ChatResponseMsg{Err: err}
		}
		defer geminiClient.Close()

		chatEngine := chat.NewChatEngine(retriever, geminiClient, redisClient)

		// Get session
		session, err := chatEngine.GetSession(ctx, sessionID)
		if err != nil {
			// Create new session if not found (selectedPapers now contains paper titles, not paths)
			session, err = chatEngine.StartSession(ctx, selectedPapers)
			if err != nil {
				return ChatResponseMsg{Err: err}
			}
		}

		// Send message
		response, err := chatEngine.Chat(ctx, session, userMessage)
		if err != nil {
			return ChatResponseMsg{Err: err}
		}

		return ChatResponseMsg{Message: response}
	}
}

// loadChatMenu loads the chat submenu
func (m *Model) loadChatMenu() {
	items := []list.Item{
		item{
			title:       "💬 Chat with Processed Papers",
			description: "Select from papers you've already processed",
			action:      "chat_processed",
		},
		item{
			title:       "🚀 Process & Chat with Any Paper",
			description: "Pick any paper from library, process it, and start chatting",
			action:      "chat_any",
		},
	}

	delegate := createStyledDelegate()
	chatMenu := list.New(items, delegate, 0, 0)
	chatMenu.Title = "💬 Chat Options"
	chatMenu.SetShowStatusBar(false)
	chatMenu.SetFilteringEnabled(false)
	chatMenu.Styles.Title = titleStyle

	m.chatMenu = chatMenu
}

// loadPapersForChat loads papers for chat selection
func (m *Model) loadPapersForChat() {
	// Load FAISS vector store to check which papers are indexed
	indexDir := filepath.Join(".metadata", "vector_index")
	vectorStore, err := rag.NewFAISSVectorStore(indexDir)
	if err != nil {
		log.Printf("⚠️  Warning: Could not load vector store: %v", err)
		vectorStore = nil
	}

	// Get list of indexed papers
	indexedPapersMap := make(map[string]bool)
	if vectorStore != nil {
		indexedPapers := vectorStore.GetIndexedPapers()
		log.Printf("📊 Found %d indexed papers", len(indexedPapers))
		for _, paper := range indexedPapers {
			indexedPapersMap[paper] = true
		}
	}

	// Get all processed papers from reports folder
	processedFiles, err := fileutil.GetPDFFiles(m.config.ReportOutputDir)
	if err != nil {
		log.Printf("❌ Error loading processed papers: %v", err)
		// Create empty list
		delegate := createStyledDelegate()
		chatList := list.New([]list.Item{}, delegate, 0, 0)
		chatList.Title = "💬 Error loading papers"
		m.chatPaperList = chatList
		return
	}

	if len(processedFiles) == 0 {
		log.Println("⚠️  No processed papers found")
		delegate := createStyledDelegate()
		chatList := list.New([]list.Item{}, delegate, 0, 0)
		chatList.Title = "💬 No processed papers found"
		m.chatPaperList = chatList
		return
	}

	// Create items list
	m.allPapersForSelect = make([]string, 0, len(processedFiles))
	m.multiSelectIndexes = make(map[int]bool)

	items := make([]list.Item, 0, len(processedFiles))
	for _, reportFile := range processedFiles {
		basename := filepath.Base(reportFile)
		// Extract paper title from report filename (remove "_Student_Guide.pdf" suffix)
		paperTitle := strings.TrimSuffix(basename, ".pdf")
		paperTitle = strings.TrimSuffix(paperTitle, "_Student_Guide")
		paperTitle = strings.ReplaceAll(paperTitle, "_", " ")

		m.allPapersForSelect = append(m.allPapersForSelect, paperTitle)

		// Check if indexed
		description := "📖 Not indexed - will be indexed before chat"
		if indexedPapersMap[paperTitle] {
			description = "✅ Indexed and ready for chat"
		}

		log.Printf("  ✓ Adding paper: %s (%s)", paperTitle, description)
		items = append(items, item{
			title:       paperTitle,
			description: description,
			action:      paperTitle,
		})
	}

	// Create list
	delegate := createStyledDelegate()
	chatList := list.New(items, delegate, 0, 0)
	chatList.Title = fmt.Sprintf("💬 Select Papers to Chat About (Space to toggle, Enter to confirm) - 0 selected (%d available)", len(items))
	chatList.SetShowStatusBar(false)
	chatList.SetFilteringEnabled(false)
	chatList.Styles.Title = titleStyle

	log.Printf("✅ Chat list created with %d items (%d indexed, %d need indexing)",
		len(items), len(indexedPapersMap), len(items)-len(indexedPapersMap))
	m.chatPaperList = chatList
}

// loadAnyPaperForChat loads all papers from library for processing and chat
func (m *Model) loadAnyPaperForChat() {
	// Get all PDF files from library
	allFiles, err := fileutil.GetPDFFiles(m.config.InputDir)
	if err != nil {
		log.Printf("❌ Error loading papers from library: %v", err)
		delegate := createStyledDelegate()
		chatList := list.New([]list.Item{}, delegate, 0, 0)
		chatList.Title = "💬 Error loading papers"
		m.chatPaperList = chatList
		return
	}

	if len(allFiles) == 0 {
		log.Println("⚠️  No papers found in library")
		delegate := createStyledDelegate()
		chatList := list.New([]list.Item{}, delegate, 0, 0)
		chatList.Title = "💬 No papers in library"
		m.chatPaperList = chatList
		return
	}

	// Create items list
	m.allPapersForSelect = make([]string, 0, len(allFiles))
	m.multiSelectIndexes = make(map[int]bool)

	items := make([]list.Item, 0, len(allFiles))
	for _, pdfFile := range allFiles {
		basename := filepath.Base(pdfFile)
		paperTitle := strings.TrimSuffix(basename, ".pdf")
		paperTitle = strings.ReplaceAll(paperTitle, "_", " ")

		m.allPapersForSelect = append(m.allPapersForSelect, pdfFile)

		items = append(items, item{
			title:       paperTitle,
			description: "📄 Will be processed and indexed for chat",
			action:      pdfFile,
		})
	}

	// Create list
	delegate := createStyledDelegate()
	chatList := list.New(items, delegate, 0, 0)
	chatList.Title = fmt.Sprintf("🚀 Select Paper to Process & Chat - 0 selected (%d available)", len(items))
	chatList.SetShowStatusBar(false)
	chatList.SetFilteringEnabled(false)
	chatList.Styles.Title = titleStyle

	log.Printf("✅ Chat list created with %d papers from library", len(items))
	m.chatPaperList = chatList
}

// renderChatScreen renders the chat interface
func (m Model) renderChatScreen() string {
	if m.chatLoading {
		return m.renderChatLoading()
	}

	// Chat container style
	chatContainer := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(m.width - 4)

	// Build chat history
	var chatHistory strings.Builder
	chatHistory.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("💬 Chat Session") + "\n\n")

	// Show selected papers
	if len(m.chatSelectedPapers) > 0 {
		chatHistory.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("242")).
			Render("Papers: "))

		for i, paperTitle := range m.chatSelectedPapers {
			if i > 0 {
				chatHistory.WriteString(", ")
			}
			chatHistory.WriteString(paperTitle)
		}
		chatHistory.WriteString("\n")
		chatHistory.WriteString(strings.Repeat("─", m.width-8) + "\n\n")
	}

	// Render messages
	maxHeight := m.height - 15
	visibleMessages := m.chatMessages
	if len(visibleMessages) > 10 {
		visibleMessages = visibleMessages[len(visibleMessages)-10:]
	}

	for _, msg := range visibleMessages {
		if msg.Role == "user" {
			// User message
			chatHistory.WriteString(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39")).
				Render("You: "))
			chatHistory.WriteString(msg.Content + "\n\n")
		} else {
			// Assistant message
			chatHistory.WriteString(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("86")).
				Render("🤖 Archivist: "))
			chatHistory.WriteString(msg.Content + "\n")

			// Citations
			if len(msg.Citations) > 0 {
				chatHistory.WriteString(lipgloss.NewStyle().
					Foreground(lipgloss.Color("242")).
					Italic(true).
					Render("\n📚 Sources: " + strings.Join(msg.Citations, ", ")) + "\n")
			}
			chatHistory.WriteString("\n")
		}
	}

	// Limit height
	historyLines := strings.Split(chatHistory.String(), "\n")
	if len(historyLines) > maxHeight {
		historyLines = historyLines[len(historyLines)-maxHeight:]
	}
	history := strings.Join(historyLines, "\n")

	// Input box
	inputBox := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Width(m.width - 8).
		Render("> " + m.chatInput + "█")

	// Combine
	content := chatContainer.Render(history) + "\n\n" + inputBox

	// Help text
	helpText := helpStyle.Render("Enter: Send • ESC: Back • Ctrl+C: Quit")

	return content + "\n\n" + helpText
}

// renderChatLoading renders loading state
func (m Model) renderChatLoading() string {
	loading := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("🤖 Archivist is thinking...")

	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := spinner[int(time.Now().UnixMilli()/100)%len(spinner)]

	return "\n\n" + loading + " " + frame + "\n\n" + helpStyle.Render("Please wait...")
}
