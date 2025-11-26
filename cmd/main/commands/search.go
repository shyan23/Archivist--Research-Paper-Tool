package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"archivist/internal/app"
	"archivist/internal/search"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var (
	searchMaxResults int
	searchSources    []string
	searchDownload   bool
	searchServiceURL string
)

// NewSearchCommand creates the search command
func NewSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for research papers from arXiv, OpenReview, and ACL Anthology",
		Long: `Search for academic papers across multiple sources including:
- arXiv (AI/ML papers)
- OpenReview (ICLR, NeurIPS conferences)
- ACL Anthology (EMNLP, ACL, NLP papers)

The search microservice must be running for this command to work.
Start it with: cd services/search-engine && python run.py`,
		Args: cobra.MinimumNArgs(1),
		RunE: runSearch,
	}

	cmd.Flags().IntVarP(&searchMaxResults, "max-results", "n", 20, "Maximum number of results")
	cmd.Flags().StringSliceVarP(&searchSources, "sources", "s", []string{}, "Filter by sources (arxiv, openreview, acl)")
	cmd.Flags().BoolVarP(&searchDownload, "download", "d", false, "Download selected papers to lib/")
	cmd.Flags().StringVar(&searchServiceURL, "service-url", "http://localhost:8000", "Search service URL")

	return cmd
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	// Create search client
	client := search.NewClient(searchServiceURL)

	// Check if service is running
	if !client.IsServiceRunning() {
		return fmt.Errorf(`search service is not running

Please start the search microservice:
  cd services/search-engine
  python3 -m venv venv
  source venv/bin/activate
  pip install -r requirements.txt
  python run.py

Then try your search again.`)
	}

	// Load config for lib directory
	config, err := app.LoadConfig(ConfigPath)
	if err != nil {
		color.Yellow("Warning: Could not load config, using default lib directory")
		config = &app.Config{InputDir: "./lib"}
	}

	// Print search info
	color.Cyan("\n🔍 Searching for: %s\n", query)
	if len(searchSources) > 0 {
		color.Cyan("   Sources: %v\n", searchSources)
	} else {
		color.Cyan("   Sources: all (arXiv, OpenReview, ACL)\n")
	}
	color.Cyan("   Max results: %d\n\n", searchMaxResults)

	// Perform search
	searchQuery := &search.SearchQuery{
		Query:      query,
		MaxResults: searchMaxResults,
		Sources:    searchSources,
	}

	results, err := client.Search(searchQuery)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if results.Total == 0 {
		color.Yellow("No results found for: %s\n", query)
		return nil
	}

	// Display results
	color.Green("✓ Found %d papers\n", results.Total)
	color.Cyan("  Sources searched: %v\n\n", results.SourcesSearched)

	// Print results
	for i, result := range results.Results {
		printSearchResult(i+1, &result)
	}

	// Offer to download if flag is set
	if searchDownload {
		return handleDownload(client, results.Results, config.InputDir)
	}

	return nil
}

func printSearchResult(index int, result *search.SearchResult) {
	// Header with index and title
	color.New(color.Bold, color.FgWhite).Printf("[%d] %s\n", index, result.Title)

	// Source and venue
	color.Cyan("    Source: %s | Venue: %s | Published: %s\n",
		result.Source, result.Venue, result.PublishedAt.Format("2006-01-02"))

	// Relevance scores (new!)
	if result.RelevanceScore != nil || result.FuzzyScore != nil || result.SimilarityScore != nil {
		scores := []string{}
		if result.RelevanceScore != nil {
			percentage := *result.RelevanceScore * 100
			scores = append(scores, fmt.Sprintf("Relevance: %.1f%%", percentage))
		}
		if result.FuzzyScore != nil {
			scores = append(scores, fmt.Sprintf("Fuzzy Match: %.1f%%", *result.FuzzyScore))
		}
		if result.SimilarityScore != nil {
			percentage := *result.SimilarityScore * 100
			scores = append(scores, fmt.Sprintf("Similarity: %.1f%%", percentage))
		}
		if len(scores) > 0 {
			color.Green("    📊 %s\n", strings.Join(scores, " | "))
		}
	}

	// Authors
	authorsStr := strings.Join(result.Authors[:min(3, len(result.Authors))], ", ")
	if len(result.Authors) > 3 {
		authorsStr += fmt.Sprintf(" +%d more", len(result.Authors)-3)
	}
	color.White("    Authors: %s\n", authorsStr)

	// Abstract (truncated)
	abstract := result.Abstract
	if len(abstract) > 200 {
		abstract = abstract[:200] + "..."
	}
	color.White("    %s\n", abstract)

	// URLs
	color.Blue("    PDF: %s\n", result.PDFURL)
	color.Blue("    Source: %s\n", result.SourceURL)

	// Categories if available
	if len(result.Categories) > 0 {
		color.Yellow("    Categories: %v\n", result.Categories[:min(5, len(result.Categories))])
	}

	fmt.Println()
}

func handleDownload(client *search.Client, results []search.SearchResult, libDir string) error {
	if len(results) == 0 {
		return nil
	}

	color.Cyan("\n📥 Download Papers\n\n")

	// Create selection prompt
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "▸ {{ .Title | cyan }} ({{ .Source }})",
		Inactive: "  {{ .Title }} ({{ .Source }})",
		Selected: "✓ {{ .Title | green }}",
	}

	// Add "Download All" and "Exit" options
	type option struct {
		Title  string
		Source string
		Index  int
	}

	options := []option{
		{Title: "Download All Papers", Source: "Action", Index: -1},
		{Title: "Exit (no download)", Source: "Action", Index: -2},
	}

	for i, result := range results {
		options = append(options, option{
			Title:  result.Title,
			Source: result.Source,
			Index:  i,
		})
	}

	prompt := promptui.Select{
		Label:     "Select papers to download (use arrows and Enter)",
		Items:     options,
		Templates: templates,
		Size:      15,
	}

	selectedIndices := make(map[int]bool)

	for {
		idx, _, err := prompt.Run()
		if err != nil {
			return nil
		}

		selected := options[idx]

		// Handle special options
		if selected.Index == -2 {
			// Exit
			break
		} else if selected.Index == -1 {
			// Download all
			for i := range results {
				selectedIndices[i] = true
			}
			break
		} else {
			// Toggle selection
			if selectedIndices[selected.Index] {
				delete(selectedIndices, selected.Index)
				color.Yellow("Removed: %s\n", selected.Title)
			} else {
				selectedIndices[selected.Index] = true
				color.Green("Added: %s\n", selected.Title)
			}

			// Ask if done
			continuePrompt := promptui.Prompt{
				Label:     "Continue selecting? (y/n)",
				IsConfirm: true,
			}

			_, err := continuePrompt.Run()
			if err != nil {
				// User said no or Ctrl+C
				break
			}
		}
	}

	if len(selectedIndices) == 0 {
		color.Yellow("No papers selected for download.\n")
		return nil
	}

	// Download selected papers
	color.Cyan("\n📥 Downloading %d papers to %s...\n\n", len(selectedIndices), libDir)

	successCount := 0
	for idx := range selectedIndices {
		result := results[idx]

		color.White("Downloading [%d/%d]: %s\n", successCount+1, len(selectedIndices), result.Title)

		// Generate filename from paper ID
		filename := sanitizeFilename(result.Title)
		if filename == "" {
			filename = result.ID
		}
		filename = filename + ".pdf"

		// Download to temporary location first
		downloadResp, err := client.DownloadPaper(result.PDFURL, filename)
		if err != nil {
			color.Red("  ✗ Failed: %v\n", err)
			continue
		}

		// Move from temp location to lib directory
		// The Python service downloads to /tmp/archivist_downloads
		tempPath := filepath.Join("/tmp/archivist_downloads", downloadResp.Filename)
		finalPath := filepath.Join(libDir, downloadResp.Filename)

		if err := os.Rename(tempPath, finalPath); err != nil {
			// If rename fails, try copying
			if err := copyFile(tempPath, finalPath); err != nil {
				color.Red("  ✗ Failed to move file: %v\n", err)
				continue
			}
			os.Remove(tempPath)
		}

		color.Green("  ✓ Downloaded: %s (%.2f MB)\n", downloadResp.Filename, float64(downloadResp.SizeBytes)/(1024*1024))
		successCount++
	}

	color.Green("\n✓ Successfully downloaded %d/%d papers\n", successCount, len(selectedIndices))

	return nil
}

func sanitizeFilename(filename string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "\n", "\r", "\t"}
	for _, char := range invalid {
		filename = strings.ReplaceAll(filename, char, "_")
	}
	filename = strings.TrimSpace(filename)
	filename = strings.Trim(filename, ".")

	if len(filename) > 200 {
		filename = filename[:200]
	}

	return filename
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0644)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
