package commands

import (
	"archivist/internal/analyzer"
	"archivist/internal/app"
	"archivist/internal/chat"
	"archivist/internal/rag"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

var (
	chatPapers    []string
	chatInteractive bool
	chatExport     string
)

var chatCmd = &cobra.Command{
	Use:   "chat [paper.pdf]",
	Short: "Interactive Q&A chat with your papers",
	Long: `Start an interactive chat session to ask questions about your research papers.
The chat uses RAG (Retrieval Augmented Generation) to provide context-aware answers.

Examples:
  archivist chat paper.pdf                    # Chat with a single paper
  archivist chat --papers lib/*.pdf           # Chat with multiple papers
  archivist chat                              # Interactive paper selection`,
	RunE: runChat,
}

// NewChatCommand creates the chat command
func NewChatCommand() *cobra.Command {
	chatCmd.Flags().StringSliceVar(&chatPapers, "papers", []string{}, "Papers to chat about (comma-separated)")
	chatCmd.Flags().BoolVarP(&chatInteractive, "interactive", "i", true, "Interactive mode")
	chatCmd.Flags().StringVarP(&chatExport, "export", "e", "", "Export chat to LaTeX file")
	return chatCmd
}

func runChat(cmd *cobra.Command, args []string) error {
	config, err := app.LoadConfig(ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()

	// Determine which papers to chat about
	var paperPaths []string

	if len(args) > 0 {
		// Paper specified as argument
		paperPaths = append(paperPaths, args[0])
	} else if len(chatPapers) > 0 {
		// Papers specified via flag
		paperPaths = chatPapers
	} else {
		// Interactive selection
		selected, err := selectPapersForChat(config.InputDir)
		if err != nil {
			return err
		}
		paperPaths = selected
	}

	if len(paperPaths) == 0 {
		return fmt.Errorf("no papers selected")
	}

	fmt.Printf("\n🤖 Starting chat with %d paper(s)...\n", len(paperPaths))

	// Initialize components
	fmt.Println("⚙️  Initializing chat engine...")

	// Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.Cache.Redis.Addr,
		Password: config.Cache.Redis.Password,
		DB:       config.Cache.Redis.DB,
	})
	defer redisClient.Close()

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w (make sure Redis Stack is running)", err)
	}

	// Initialize RAG components
	embedClient, err := rag.NewEmbeddingClient(config.Gemini.APIKey)
	if err != nil {
		return fmt.Errorf("failed to create embedding client: %w", err)
	}
	defer embedClient.Close()

	vectorStore, err := rag.NewVectorStore(redisClient)
	if err != nil {
		return fmt.Errorf("failed to create vector store: %w", err)
	}

	retrievalConfig := rag.DefaultRetrievalConfig()
	retrievalConfig.TopK = 5
	retriever := rag.NewRetriever(vectorStore, embedClient, retrievalConfig)

	// Gemini client for chat
	geminiClient, err := analyzer.NewGeminiClient(
		config.Gemini.APIKey,
		config.Gemini.Model,
		config.Gemini.Temperature,
		config.Gemini.MaxTokens,
	)
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer geminiClient.Close()

	// Chat engine
	chatEngine := chat.NewChatEngine(retriever, geminiClient, redisClient)

	// Extract paper titles from paths
	paperTitles := make([]string, len(paperPaths))
	for i, path := range paperPaths {
		paperTitles[i] = extractPaperTitle(path)
	}

	// Check if papers are indexed
	fmt.Println("\n📚 Checking paper indices...")
	indexer := rag.NewIndexer(
		rag.NewChunker(rag.DefaultChunkSize, rag.DefaultChunkOverlap),
		embedClient,
		vectorStore,
	)

	for i, title := range paperTitles {
		indexed, numChunks, err := indexer.CheckIfIndexed(ctx, title)
		if err != nil {
			return fmt.Errorf("failed to check index status: %w", err)
		}

		if !indexed {
			fmt.Printf("⚠️  Paper not indexed: %s\n", title)
			fmt.Printf("   Run 'archivist process %s' first to index this paper.\n", paperPaths[i])
			return fmt.Errorf("paper not indexed")
		}

		fmt.Printf("✓ %s (%d chunks indexed)\n", title, numChunks)
	}

	// Start chat session
	session, err := chatEngine.StartSession(ctx, paperTitles)
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	fmt.Printf("\n✅ Chat session started (ID: %s)\n", session.ID)
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("💬 Chat Mode - Ask questions about your papers")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\nTips:")
	fmt.Println("  - Ask specific questions about methodologies, results, etc.")
	fmt.Println("  - Type 'exit' or 'quit' to end the session")
	fmt.Println("  - Type 'export' to save the conversation to LaTeX")
	fmt.Println("")

	// Interactive chat loop
	for {
		// Prompt for user input
		prompt := promptui.Prompt{
			Label: "You",
		}

		userInput, err := prompt.Run()
		if err != nil {
			fmt.Println("\n👋 Goodbye!")
			break
		}

		userInput = strings.TrimSpace(userInput)

		// Handle special commands
		if userInput == "" {
			continue
		}

		if userInput == "exit" || userInput == "quit" {
			fmt.Println("\n👋 Goodbye!")
			break
		}

		if userInput == "export" {
			exportPath := chatExport
			if exportPath == "" {
				exportPath = fmt.Sprintf("chat_session_%s.tex", session.ID)
			}

			latex := chatEngine.ExportSessionToLatex(session)
			if err := os.WriteFile(exportPath, []byte(latex), 0644); err != nil {
				fmt.Printf("❌ Failed to export: %v\n", err)
			} else {
				fmt.Printf("✅ Chat exported to: %s\n", exportPath)
			}
			continue
		}

		// Process chat message
		fmt.Println("")
		response, err := chatEngine.Chat(ctx, session, userInput)
		if err != nil {
			fmt.Printf("❌ Error: %v\n\n", err)
			continue
		}

		// Display response
		fmt.Println(strings.Repeat("-", 60))
		fmt.Println("🤖 Archivist:")
		fmt.Println("")
		fmt.Println(response.Content)

		if len(response.Citations) > 0 {
			fmt.Println("")
			fmt.Println("📚 Sources:")
			for _, citation := range response.Citations {
				fmt.Printf("   - %s\n", citation)
			}
		}

		fmt.Println(strings.Repeat("-", 60))
		fmt.Println("")
	}

	// Ask if user wants to export
	if chatExport == "" {
		exportPrompt := promptui.Prompt{
			Label:     "Export conversation to LaTeX? (y/n)",
			IsConfirm: true,
		}

		result, err := exportPrompt.Run()
		if err == nil && (result == "y" || result == "Y") {
			exportPath := fmt.Sprintf("chat_session_%d.tex", time.Now().Unix())
			latex := chatEngine.ExportSessionToLatex(session)
			if err := os.WriteFile(exportPath, []byte(latex), 0644); err != nil {
				fmt.Printf("❌ Failed to export: %v\n", err)
			} else {
				fmt.Printf("✅ Chat exported to: %s\n", exportPath)
			}
		}
	}

	return nil
}

// selectPapersForChat allows interactive selection of papers
func selectPapersForChat(libDir string) ([]string, error) {
	// Find all PDF files
	pdfFiles, err := findPDFFiles(libDir)
	if err != nil {
		return nil, err
	}

	if len(pdfFiles) == 0 {
		return nil, fmt.Errorf("no PDF files found in %s", libDir)
	}

	// Simple prompt for selection
	fmt.Println("\n📚 Available papers:")
	for i, file := range pdfFiles {
		fmt.Printf("  %d. %s\n", i+1, filepath.Base(file))
	}

	prompt := promptui.Prompt{
		Label: "Select papers (comma-separated numbers, or 'all')",
	}

	input, err := prompt.Run()
	if err != nil {
		return nil, err
	}

	input = strings.TrimSpace(input)

	if input == "all" {
		return pdfFiles, nil
	}

	// Parse selections
	selections := strings.Split(input, ",")
	var selected []string

	for _, sel := range selections {
		sel = strings.TrimSpace(sel)
		var idx int
		if _, err := fmt.Sscanf(sel, "%d", &idx); err == nil && idx > 0 && idx <= len(pdfFiles) {
			selected = append(selected, pdfFiles[idx-1])
		}
	}

	return selected, nil
}

func extractPaperTitle(pdfPath string) string {
	base := filepath.Base(pdfPath)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func findPDFFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".pdf" {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}
