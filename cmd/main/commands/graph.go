package commands

import (
	"archivist/internal/app"
	"archivist/internal/ui"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	graphServiceURL string
	priority        int
)

// NewGraphCommand creates the graph command
func NewGraphCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Knowledge graph operations",
		Long:  "Interact with the knowledge graph service for paper relationships, citations, and semantic search",
	}

	// Add subcommands
	cmd.AddCommand(
		newGraphAddCommand(),
		newGraphStatsCommand(),
		newGraphStatusCommand(),
	)

	// Global flags for graph commands
	cmd.PersistentFlags().StringVar(&graphServiceURL, "graph-url", "http://localhost:8081", "graph service URL")

	return cmd
}

// newGraphAddCommand creates the 'graph add' subcommand
func newGraphAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [paper.pdf]",
		Short: "Add a paper to the knowledge graph",
		Long:  "Process a paper and add it to the Neo4j knowledge graph with citations, authors, and relationships",
		Args:  cobra.ExactArgs(1),
		Run:   runGraphAdd,
	}

	cmd.Flags().IntVarP(&priority, "priority", "p", 0, "processing priority (higher = processed first)")

	return cmd
}

// newGraphStatsCommand creates the 'graph stats' subcommand
func newGraphStatsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show knowledge graph statistics",
		Long:  "Display statistics about papers, authors, citations, and methods in the graph",
		Run:   runGraphStats,
	}

	return cmd
}

// newGraphStatusCommand creates the 'graph status' subcommand
func newGraphStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check graph service status",
		Long:  "Check if the graph service is running and healthy",
		Run:   runGraphStatus,
	}

	return cmd
}

func runGraphAdd(cmd *cobra.Command, args []string) {
	pdfPath := args[0]

	// Load config
	config, err := app.LoadConfig(ConfigPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	// Check if file exists
	if !fileExists(pdfPath) {
		ui.PrintError(fmt.Sprintf("File not found: %s", pdfPath))
		os.Exit(1)
	}

	// Get absolute path
	absPath, err := filepath.Abs(pdfPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to get absolute path: %v", err))
		os.Exit(1)
	}

	// Check if paper has been processed (LaTeX exists)
	paperName := filepath.Base(absPath)
	paperName = paperName[:len(paperName)-len(filepath.Ext(paperName))]
	latexPath := filepath.Join(config.ReportOutputDir, paperName+".tex")

	if !fileExists(latexPath) {
		ui.PrintWarning(fmt.Sprintf("Paper not yet processed: %s", paperName))
		ui.PrintInfo("Please process the paper first using: ./archivist process " + pdfPath)
		os.Exit(1)
	}

	// Read LaTeX content
	ui.PrintInfo(fmt.Sprintf("Reading LaTeX report: %s", latexPath))
	latexContent, err := os.ReadFile(latexPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to read LaTeX: %v", err))
		os.Exit(1)
	}

	// Prepare request
	requestBody := map[string]interface{}{
		"paper_title": paperName,
		"latex_content": string(latexContent),
		"pdf_path": absPath,
		"priority": priority,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to create request: %v", err))
		os.Exit(1)
	}

	// Send to graph service
	ui.PrintInfo(fmt.Sprintf("Submitting to graph service: %s", graphServiceURL))

	resp, err := http.Post(
		graphServiceURL+"/api/graph/add-paper",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to connect to graph service: %v", err))
		ui.PrintWarning("Make sure the graph service is running:")
		ui.PrintInfo("  docker-compose -f docker-compose-graph.yml up -d")
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to read response: %v", err))
		os.Exit(1)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to parse response: %v", err))
		os.Exit(1)
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		ui.PrintError(fmt.Sprintf("Graph service error: %s", result["detail"]))
		os.Exit(1)
	}

	// Success!
	ui.PrintSuccess(fmt.Sprintf("Paper queued for graph building: %s", paperName))
	ui.PrintInfo(fmt.Sprintf("Job ID: %s", result["job_id"]))
	ui.PrintInfo(fmt.Sprintf("Queue position: %v", result["queue_position"]))

	fmt.Println()
	ui.PrintInfo("Track progress:")
	ui.PrintInfo(fmt.Sprintf("  curl %s/api/graph/job/%s", graphServiceURL, result["job_id"]))
}

func runGraphStats(cmd *cobra.Command, args []string) {
	// Get stats from graph service
	ui.PrintInfo(fmt.Sprintf("Fetching graph statistics from: %s", graphServiceURL))

	resp, err := http.Get(graphServiceURL + "/api/graph/stats")
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to connect to graph service: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to read response: %v", err))
		os.Exit(1)
	}

	// Parse response
	var stats map[string]interface{}
	if err := json.Unmarshal(body, &stats); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to parse response: %v", err))
		os.Exit(1)
	}

	// Display stats
	fmt.Println()
	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	ui.ColorBold.Println("            KNOWLEDGE GRAPH STATISTICS                         ")
	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	ui.ColorInfo.Printf("  📄 Papers:        %v\n", stats["paper_count"])
	ui.ColorInfo.Printf("  👥 Authors:       %v\n", stats["author_count"])
	ui.ColorInfo.Printf("  🔗 Citations:     %v\n", stats["citation_count"])
	ui.ColorInfo.Printf("  🔬 Methods:       %v\n", stats["method_count"])
	ui.ColorInfo.Printf("  📊 Datasets:      %v\n", stats["dataset_count"])
	ui.ColorInfo.Printf("  📚 Venues:        %v\n", stats["venue_count"])
	ui.ColorInfo.Printf("  🏛️  Institutions: %v\n", stats["institution_count"])
	fmt.Println()

	ui.PrintInfo("View graph in Neo4j browser: http://localhost:7474")
	fmt.Println()
}

func runGraphStatus(cmd *cobra.Command, args []string) {
	ui.PrintInfo(fmt.Sprintf("Checking graph service: %s", graphServiceURL))

	resp, err := http.Get(graphServiceURL + "/health")
	if err != nil {
		ui.PrintError(fmt.Sprintf("Graph service is not running: %v", err))
		ui.PrintWarning("Start the graph service:")
		ui.PrintInfo("  docker-compose -f docker-compose-graph.yml up -d")
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to read response: %v", err))
		os.Exit(1)
	}

	// Parse response
	var health map[string]interface{}
	if err := json.Unmarshal(body, &health); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to parse response: %v", err))
		os.Exit(1)
	}

	// Display status
	fmt.Println()
	status := health["status"]
	if status == "healthy" {
		ui.PrintSuccess("Graph service is healthy")
	} else {
		ui.PrintWarning(fmt.Sprintf("Graph service status: %v", status))
	}

	ui.PrintInfo(fmt.Sprintf("Neo4j: %v", health["neo4j"]))
	ui.PrintInfo(fmt.Sprintf("Worker queue: %v", health["worker_queue"]))
	ui.PrintInfo(fmt.Sprintf("Queue size: %v", health["queue_size"]))
	fmt.Println()
}
