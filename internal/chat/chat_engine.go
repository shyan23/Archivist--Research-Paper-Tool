package chat

import (
	"archivist/internal/analyzer"
	"archivist/internal/rag"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// ChatHistoryPrefix is the Redis key prefix for chat histories
	ChatHistoryPrefix = "archivist:chat:history:"
	// ChatHistoryTTL is the TTL for chat histories (24 hours)
	ChatHistoryTTL = 24 * time.Hour
)

// Message represents a single message in a conversation
type Message struct {
	Role      string    `json:"role"`       // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Citations []string  `json:"citations"`  // Source citations for assistant messages
}

// ChatSession represents an ongoing chat session
type ChatSession struct {
	ID            string    `json:"id"`
	PaperTitles   []string  `json:"paper_titles"`
	Messages      []Message `json:"messages"`
	CreatedAt     time.Time `json:"created_at"`
	LastUpdated   time.Time `json:"last_updated"`
}

// ChatEngine handles RAG-powered chat interactions
type ChatEngine struct {
	retriever    *rag.Retriever
	geminiClient *analyzer.GeminiClient
	redisClient  *redis.Client
}

// NewChatEngine creates a new chat engine
func NewChatEngine(retriever *rag.Retriever, geminiClient *analyzer.GeminiClient, redisClient *redis.Client) *ChatEngine {
	return &ChatEngine{
		retriever:    retriever,
		geminiClient: geminiClient,
		redisClient:  redisClient,
	}
}

// StartSession starts a new chat session
func (ce *ChatEngine) StartSession(ctx context.Context, paperTitles []string) (*ChatSession, error) {
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())

	session := &ChatSession{
		ID:          sessionID,
		PaperTitles: paperTitles,
		Messages:    []Message{},
		CreatedAt:   time.Now(),
		LastUpdated: time.Now(),
	}

	// Save to Redis
	if err := ce.saveSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	log.Printf("✓ Started chat session %s with %d papers", sessionID, len(paperTitles))

	return session, nil
}

// Chat processes a user message and generates a response
func (ce *ChatEngine) Chat(ctx context.Context, session *ChatSession, userMessage string) (*Message, error) {
	if userMessage == "" {
		return nil, fmt.Errorf("empty message")
	}

	// Add user message to session
	userMsg := Message{
		Role:      "user",
		Content:   userMessage,
		Timestamp: time.Now(),
	}
	session.Messages = append(session.Messages, userMsg)

	log.Printf("  💬 User: %s", truncateString(userMessage, 60))

	// Retrieve relevant context
	log.Println("  🔍 Retrieving relevant context...")
	var retrievedContext *rag.RetrievedContext
	var err error

	if len(session.PaperTitles) == 0 {
		// Search across all papers
		retrievedContext, err = ce.retriever.Retrieve(ctx, userMessage, nil)
	} else if len(session.PaperTitles) == 1 {
		// Single paper context
		retrievedContext, err = ce.retriever.RetrieveFromPaper(ctx, userMessage, session.PaperTitles[0])
	} else {
		// Multi-paper context
		retrievedContext, err = ce.retriever.RetrieveMultiPaper(ctx, userMessage, session.PaperTitles)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve context: %w", err)
	}

	log.Printf("  ✓ Retrieved %d relevant chunks", len(retrievedContext.Chunks))

	// Build prompt with context and conversation history
	prompt := ce.buildPrompt(session, userMessage, retrievedContext)

	// Generate response using Gemini
	log.Println("  🤖 Generating response...")
	response, err := ce.geminiClient.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	// Extract citations
	citations := ce.extractCitations(retrievedContext)

	// Create assistant message
	assistantMsg := Message{
		Role:      "assistant",
		Content:   response,
		Timestamp: time.Now(),
		Citations: citations,
	}

	// Add to session
	session.Messages = append(session.Messages, assistantMsg)
	session.LastUpdated = time.Now()

	// Save updated session
	if err := ce.saveSession(ctx, session); err != nil {
		log.Printf("Warning: failed to save session: %v", err)
	}

	log.Printf("  ✓ Response generated (%d chars)", len(response))

	return &assistantMsg, nil
}

// GetSession retrieves a session from Redis
func (ce *ChatEngine) GetSession(ctx context.Context, sessionID string) (*ChatSession, error) {
	key := ChatHistoryPrefix + sessionID

	data, err := ce.redisClient.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session ChatSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// ListSessions lists all active chat sessions
func (ce *ChatEngine) ListSessions(ctx context.Context) ([]*ChatSession, error) {
	pattern := ChatHistoryPrefix + "*"

	keys, err := ce.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	sessions := make([]*ChatSession, 0, len(keys))

	for _, key := range keys {
		data, err := ce.redisClient.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var session ChatSession
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// DeleteSession deletes a chat session
func (ce *ChatEngine) DeleteSession(ctx context.Context, sessionID string) error {
	key := ChatHistoryPrefix + sessionID

	err := ce.redisClient.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// ExportSessionToLatex exports a chat session to LaTeX format
func (ce *ChatEngine) ExportSessionToLatex(session *ChatSession) string {
	latex := "\\section{Q\\&A Session}\n\n"

	if len(session.PaperTitles) > 0 {
		latex += "\\subsection{Papers Discussed}\n"
		latex += "\\begin{itemize}\n"
		for _, title := range session.PaperTitles {
			latex += fmt.Sprintf("  \\item %s\n", escapeLatex(title))
		}
		latex += "\\end{itemize}\n\n"
	}

	latex += "\\subsection{Conversation}\n\n"

	for i, msg := range session.Messages {
		if msg.Role == "user" {
			latex += fmt.Sprintf("\\textbf{Question %d:} %s\n\n", (i/2)+1, escapeLatex(msg.Content))
		} else {
			latex += fmt.Sprintf("\\textbf{Answer:} %s\n\n", escapeLatex(msg.Content))

			if len(msg.Citations) > 0 {
				latex += "\\textit{Sources:} "
				for j, citation := range msg.Citations {
					if j > 0 {
						latex += ", "
					}
					latex += escapeLatex(citation)
				}
				latex += "\n\n"
			}
		}
	}

	return latex
}

// saveSession saves a session to Redis
func (ce *ChatEngine) saveSession(ctx context.Context, session *ChatSession) error {
	key := ChatHistoryPrefix + session.ID

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	err = ce.redisClient.Set(ctx, key, data, ChatHistoryTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to save session to Redis: %w", err)
	}

	return nil
}

// buildPrompt builds the RAG prompt with context and history
func (ce *ChatEngine) buildPrompt(session *ChatSession, userMessage string, context *rag.RetrievedContext) string {
	prompt := "You are a helpful AI research assistant for CS students studying AI/ML papers.\n\n"

	// Add paper context
	if len(session.PaperTitles) > 0 {
		prompt += "You are discussing the following papers:\n"
		for _, title := range session.PaperTitles {
			prompt += fmt.Sprintf("- %s\n", title)
		}
		prompt += "\n"
	}

	// Add retrieved context
	prompt += "RELEVANT CONTEXT FROM PAPERS:\n"
	prompt += "---\n"
	prompt += context.Context
	prompt += "---\n\n"

	// Add conversation history (last 3 exchanges to keep context manageable)
	if len(session.Messages) > 1 {
		prompt += "CONVERSATION HISTORY:\n"
		startIdx := len(session.Messages) - 6 // Last 3 Q&A pairs
		if startIdx < 0 {
			startIdx = 0
		}

		for i := startIdx; i < len(session.Messages)-1; i++ {
			msg := session.Messages[i]
			if msg.Role == "user" {
				prompt += fmt.Sprintf("User: %s\n", msg.Content)
			} else {
				prompt += fmt.Sprintf("Assistant: %s\n", msg.Content)
			}
		}
		prompt += "\n"
	}

	// Add current question
	prompt += "CURRENT QUESTION:\n"
	prompt += userMessage + "\n\n"

	// Add instructions
	prompt += "INSTRUCTIONS:\n"
	prompt += "- Answer the question using the provided context from the papers.\n"
	prompt += "- Be clear, concise, and student-friendly.\n"
	prompt += "- Cite specific sections when referencing information (e.g., 'According to Section 3.2...').\n"
	prompt += "- If the context doesn't contain enough information, say so.\n"
	prompt += "- Use technical terms but explain them when first introduced.\n"
	prompt += "- If comparing multiple papers, clearly distinguish between them.\n\n"

	prompt += "ANSWER:"

	return prompt
}

// extractCitations extracts citation information from retrieved context
func (ce *ChatEngine) extractCitations(context *rag.RetrievedContext) []string {
	citations := []string{}
	seen := make(map[string]bool)

	for _, chunk := range context.Chunks {
		citation := chunk.Document.Source
		if chunk.Document.Section != "" {
			citation += fmt.Sprintf(" (Section: %s)", chunk.Document.Section)
		}

		if !seen[citation] {
			seen[citation] = true
			citations = append(citations, citation)
		}
	}

	return citations
}

// Helper functions

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func escapeLatex(text string) string {
	replacements := map[string]string{
		"\\": "\\textbackslash{}",
		"&":  "\\&",
		"%":  "\\%",
		"$":  "\\$",
		"#":  "\\#",
		"_":  "\\_",
		"{":  "\\{",
		"}":  "\\}",
		"~":  "\\textasciitilde{}",
		"^":  "\\textasciicircum{}",
	}

	for old, new := range replacements {
		text = replaceAll(text, old, new)
	}

	return text
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		found := false
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old) - 1
			found = true
		}
		if !found {
			result += string(s[i])
		}
	}
	return result
}
