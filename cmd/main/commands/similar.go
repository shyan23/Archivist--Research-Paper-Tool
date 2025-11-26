package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"archivist/internal/analyzer"
	"archivist/internal/app"
	"archivist/internal/search"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	similarMaxResults  int
	similarServiceURL  string
	similarDownload    bool
)

// NewSimilarCommand creates the similar papers command
func NewSimilarCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "similar [paper-path]",
		Short: "Find papers similar to a given paper",
		Long: `Analyze a research paper and find similar papers based on its methodology and key concepts.

This command:
1. Extracts the essence of the paper (methodology, key concepts, techniques)
2. Searches arXiv, OpenReview, and ACL for similar papers
3. Ranks results by relevance

The search microservice must be running for this command to work.
Start it with: cd services/search-engine && python run.py`,
		Args: cobra.ExactArgs(1),
		RunE: runSimilar,
	}

	cmd.Flags().IntVarP(&similarMaxResults, "max-results", "n", 10, "Maximum number of similar papers to find")
	cmd.Flags().StringVar(&similarServiceURL, "service-url", "http://localhost:8000", "Search service URL")
	cmd.Flags().BoolVarP(&similarDownload, "download", "d", false, "Download similar papers to lib/")

	return cmd
}

func runSimilar(cmd *cobra.Command, args []string) error {
	paperPath := args[0]

	// Validate file exists
	if !fileExists(paperPath) {
		return fmt.Errorf("file not found: %s", paperPath)
	}

	// Load config
	config, err := app.LoadConfig(ConfigPath)
	if err != nil {
		color.Yellow("Warning: Could not load config, using defaults")
		config = &app.Config{
			InputDir: "./lib",
			Gemini: app.GeminiConfig{
				APIKey:      "",
				Model:       "gemini-2.0-flash-exp",
				Temperature: 0.7,
				MaxTokens:   8192,
			},
		}
	}

	// Check if Gemini API key is set
	if config.Gemini.APIKey == "" {
		return fmt.Errorf("Gemini API key not configured. Set GEMINI_API_KEY environment variable or configure in config.yaml")
	}

	ctx := context.Background()

	// Create analyzer
	analyzerInstance, err := analyzer.NewAnalyzer(config)
	if err != nil {
		return fmt.Errorf("failed to create analyzer: %w", err)
	}
	defer analyzerInstance.Close()

	// Create similar paper finder
	finder := analyzer.NewSimilarPaperFinder(analyzerInstance, similarServiceURL)

	// Print header
	color.Cyan("\n🔍 Finding Similar Papers\n")
	color.White("   Paper: %s\n", filepath.Base(paperPath))
	color.White("   Max results: %d\n\n", similarMaxResults)

	// Extract essence and search
	color.Cyan("📊 Analyzing paper...\n")
	essence, results, err := finder.FindSimilarPapersFromPDF(ctx, paperPath, similarMaxResults)
	if err != nil {
		return fmt.Errorf("failed to find similar papers: %w", err)
	}

	// Display essence
	color.Green("\n✓ Paper Essence Extracted:\n\n")
	color.White("   Main Methodology: %s\n", essence.MainMethodology)
	color.White("   Problem Domain: %s\n", essence.ProblemDomain)

	if len(essence.KeyConcepts) > 0 {
		color.White("   Key Concepts: %s\n", strings.Join(essence.KeyConcepts[:min(5, len(essence.KeyConcepts))], ", "))
	}

	if len(essence.Techniques) > 0 {
		color.White("   Techniques: %s\n", strings.Join(essence.Techniques[:min(3, len(essence.Techniques))], ", "))
	}

	// Display results
	if results.Total == 0 {
		color.Yellow("\nNo similar papers found.\n")
		return nil
	}

	color.Green("\n✓ Found %d similar papers:\n\n", results.Total)

	for i, result := range results.Results {
		printSearchResult(i+1, &result)
	}

	// Offer to download if flag is set
	if similarDownload {
		searchClient := search.NewClient(similarServiceURL)
		return handleDownload(searchClient, results.Results, config.InputDir)
	}

	return nil
}
