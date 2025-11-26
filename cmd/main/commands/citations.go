package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"archivist/internal/analyzer"
	"archivist/internal/app"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	citationsFoundational bool
	citationsTopN         int
	citationsFormat       string
)

// NewCitationsCommand creates the citations extraction command
func NewCitationsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "citations [paper-path]",
		Short: "Extract all citations from a research paper",
		Long: `Extract and list all papers referenced throughout a research paper.

This command:
1. Scans the entire document for paper citations
2. Identifies foundational papers that directly influenced the work
3. Counts citation frequency
4. Provides context for why each paper was cited

Modes:
  --foundational  : Show only foundational/influential papers
  --top N         : Show top N most cited papers

Output formats:
  text (default) : Human-readable colored output
  json          : JSON format for programmatic use
  markdown      : Markdown table format`,
		Args: cobra.ExactArgs(1),
		RunE: runCitations,
	}

	cmd.Flags().BoolVarP(&citationsFoundational, "foundational", "f", false, "Show only foundational papers")
	cmd.Flags().IntVarP(&citationsTopN, "top", "t", 0, "Show top N most cited papers")
	cmd.Flags().StringVar(&citationsFormat, "format", "text", "Output format (text, json, markdown)")

	return cmd
}

func runCitations(cmd *cobra.Command, args []string) error {
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

	// Create citation extractor
	extractor := analyzer.NewCitationExtractor(analyzerInstance)

	// Print header
	color.Cyan("\n📚 Extracting Citations\n")
	color.White("   Paper: %s\n\n", filepath.Base(paperPath))

	// Extract citations based on flags
	var citations []analyzer.CitedPaper

	if citationsFoundational {
		color.Cyan("   Mode: Foundational papers only\n\n")
		citations, err = extractor.ExtractFoundationalPapers(ctx, paperPath)
	} else if citationsTopN > 0 {
		color.Cyan("   Mode: Top %d most cited papers\n\n", citationsTopN)
		citations, err = extractor.ExtractMostCitedPapers(ctx, paperPath, citationsTopN)
	} else {
		color.Cyan("   Mode: All citations\n\n")
		citations, err = extractor.ExtractAllCitations(ctx, paperPath)
	}

	if err != nil {
		return fmt.Errorf("failed to extract citations: %w", err)
	}

	if len(citations) == 0 {
		color.Yellow("No citations found.\n")
		return nil
	}

	// Display results based on format
	switch citationsFormat {
	case "json":
		printCitationsJSON(citations)
	case "markdown":
		printCitationsMarkdown(citations)
	default:
		printCitationsText(citations)
	}

	return nil
}

func printCitationsText(citations []analyzer.CitedPaper) {
	color.Green("\n✓ Found %d citations:\n\n", len(citations))

	for i, cit := range citations {
		// Header with index and title
		color.New(color.Bold, color.FgWhite).Printf("[%d] %s\n", i+1, cit.Title)

		// Authors and year
		authorsStr := strings.Join(cit.Authors[:min(3, len(cit.Authors))], ", ")
		if len(cit.Authors) > 3 {
			authorsStr += fmt.Sprintf(" +%d more", len(cit.Authors)-3)
		}
		color.White("    Authors: %s (%s)\n", authorsStr, cit.Year)

		// Venue if available
		if cit.Venue != "" {
			color.Cyan("    Venue: %s\n", cit.Venue)
		}

		// Citation count and foundational status
		badges := []string{}
		if cit.CitationCount > 1 {
			badges = append(badges, fmt.Sprintf("Cited %dx", cit.CitationCount))
		}
		if cit.IsFoundational {
			badges = append(badges, "FOUNDATIONAL")
		}
		if len(badges) > 0 {
			color.Green("    📊 %s\n", strings.Join(badges, " | "))
		}

		// Context
		if cit.Context != "" {
			color.Yellow("    Context: %s\n", cit.Context)
		}

		fmt.Println()
	}
}

func printCitationsMarkdown(citations []analyzer.CitedPaper) {
	fmt.Println("# Citations\n")
	fmt.Println("| # | Title | Authors | Year | Venue | Cited | Foundational | Context |")
	fmt.Println("|---|-------|---------|------|-------|-------|--------------|---------|")

	for i, cit := range citations {
		authorsStr := strings.Join(cit.Authors[:min(2, len(cit.Authors))], ", ")
		if len(cit.Authors) > 2 {
			authorsStr += " et al."
		}

		foundationalMark := ""
		if cit.IsFoundational {
			foundationalMark = "✓"
		}

		citCount := fmt.Sprintf("%dx", cit.CitationCount)
		if cit.CitationCount == 0 {
			citCount = "-"
		}

		context := cit.Context
		if len(context) > 50 {
			context = context[:50] + "..."
		}

		fmt.Printf("| %d | %s | %s | %s | %s | %s | %s | %s |\n",
			i+1, cit.Title, authorsStr, cit.Year, cit.Venue, citCount, foundationalMark, context)
	}
}

func printCitationsJSON(citations []analyzer.CitedPaper) {
	// Simple JSON output
	fmt.Println("[")
	for i, cit := range citations {
		fmt.Printf("  {\n")
		fmt.Printf("    \"title\": %q,\n", cit.Title)
		fmt.Printf("    \"authors\": [%s],\n", formatAuthorsJSON(cit.Authors))
		fmt.Printf("    \"year\": %q,\n", cit.Year)
		fmt.Printf("    \"venue\": %q,\n", cit.Venue)
		fmt.Printf("    \"citation_count\": %d,\n", cit.CitationCount)
		fmt.Printf("    \"is_foundational\": %v,\n", cit.IsFoundational)
		fmt.Printf("    \"context\": %q\n", cit.Context)

		if i < len(citations)-1 {
			fmt.Printf("  },\n")
		} else {
			fmt.Printf("  }\n")
		}
	}
	fmt.Println("]")
}

func formatAuthorsJSON(authors []string) string {
	quoted := make([]string, len(authors))
	for i, author := range authors {
		quoted[i] = fmt.Sprintf("%q", author)
	}
	return strings.Join(quoted, ", ")
}
