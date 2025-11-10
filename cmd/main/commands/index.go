package commands

import (
	"archivist/internal/app"
	"archivist/internal/cache"
	"archivist/internal/rag"
	"archivist/pkg/fileutil"
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Index processed papers for chat feature",
	Long: `Index all processed papers into the vector database for chat functionality.
This command reads the cached LaTeX content and creates embeddings for semantic search.

Run this if:
- You processed papers before the chat feature was added
- You want to rebuild the vector index

Example:
  archivist index                    # Index all processed papers
  archivist index --force            # Reindex even if already indexed`,
	RunE: runIndex,
}

var forceReindex bool

func NewIndexCommand() *cobra.Command {
	indexCmd.Flags().BoolVarP(&forceReindex, "force", "f", false, "Force reindexing even if already indexed")
	return indexCmd
}

func runIndex(cmd *cobra.Command, args []string) error {
	config, err := app.LoadConfig(ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()

	fmt.Println("\n🔌 Connecting to Redis Stack...")

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.Cache.Redis.Addr,
		Password: config.Cache.Redis.Password,
		DB:       config.Cache.Redis.DB,
	})
	defer redisClient.Close()

	// Test connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis Stack at %s: %w\n\nMake sure Redis Stack is running on port 6380", config.Cache.Redis.Addr, err)
	}

	fmt.Printf("✅ Connected to Redis Stack at %s\n\n", config.Cache.Redis.Addr)

	// Initialize cache
	redisCache, err := cache.NewRedisCache(
		config.Cache.Redis.Addr,
		config.Cache.Redis.Password,
		config.Cache.Redis.DB,
		time.Duration(config.Cache.TTL)*time.Hour,
	)
	if err != nil {
		return fmt.Errorf("failed to create cache: %w", err)
	}
	defer redisCache.Close()

	// Initialize embedding client
	fmt.Println("🧮 Initializing Gemini embeddings...")
	embedClient, err := rag.NewEmbeddingClient(config.Gemini.APIKey)
	if err != nil {
		return fmt.Errorf("failed to create embedding client: %w", err)
	}
	defer embedClient.Close()

	// Initialize vector store
	vectorStore, err := rag.NewVectorStore(redisClient)
	if err != nil {
		return fmt.Errorf("failed to create vector store: %w", err)
	}

	// Create indexer
	chunker := rag.NewChunker(rag.DefaultChunkSize, rag.DefaultChunkOverlap)
	indexer := rag.NewIndexer(chunker, embedClient, vectorStore)

	fmt.Println("✅ Indexer ready")

	// Get all PDF files from lib
	fmt.Println("📚 Finding papers in library...")
	pdfFiles, err := fileutil.GetPDFFiles(config.InputDir)
	if err != nil {
		return fmt.Errorf("failed to get PDF files: %w", err)
	}

	if len(pdfFiles) == 0 {
		fmt.Println("⚠️  No PDF files found in library")
		return nil
	}

	fmt.Printf("Found %d papers\n\n", len(pdfFiles))

	// Index each paper
	var indexed, skipped, failed int

	for i, pdfPath := range pdfFiles {
		basename := filepath.Base(pdfPath)
		paperTitle := strings.TrimSuffix(basename, filepath.Ext(basename))

		fmt.Printf("[%d/%d] Processing: %s\n", i+1, len(pdfFiles), paperTitle)

		// Check if already indexed
		if !forceReindex {
			isIndexed, numChunks, err := indexer.CheckIfIndexed(ctx, paperTitle)
			if err != nil {
				log.Printf("  ⚠️  Error checking index status: %v", err)
			} else if isIndexed {
				fmt.Printf("  ⏭️  Already indexed (%d chunks) - skipping\n", numChunks)
				skipped++
				continue
			}
		}

		// Get cached LaTeX content
		fileHash, err := fileutil.ComputeFileHash(pdfPath)
		if err != nil {
			fmt.Printf("  ❌ Failed to compute hash: %v\n", err)
			failed++
			continue
		}

		cached, err := redisCache.Get(ctx, fileHash)
		if err != nil || cached == nil {
			fmt.Printf("  ⚠️  No cached analysis found - paper needs to be processed first\n")
			fmt.Printf("      Run: archivist process %s\n", pdfPath)
			skipped++
			continue
		}

		// Index the paper
		if forceReindex {
			err = indexer.ReindexPaper(ctx, paperTitle, cached.LatexContent, pdfPath)
		} else {
			err = indexer.IndexPaper(ctx, paperTitle, cached.LatexContent, pdfPath)
		}

		if err != nil {
			fmt.Printf("  ❌ Indexing failed: %v\n", err)
			failed++
			continue
		}

		indexed++
		fmt.Println()
	}

	// Summary
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\n📊 Indexing Summary:")
	fmt.Printf("  ✅ Successfully indexed: %d\n", indexed)
	fmt.Printf("  ⏭️  Skipped: %d\n", skipped)
	if failed > 0 {
		fmt.Printf("  ❌ Failed: %d\n", failed)
	}
	fmt.Println()

	if indexed > 0 {
		fmt.Println("✨ Papers are now ready for chat!")
		fmt.Println("\nUsage:")
		fmt.Println("  archivist chat <paper.pdf>    # Start chatting")
		fmt.Println("  archivist run                 # Use TUI chat interface")
	}

	return nil
}
