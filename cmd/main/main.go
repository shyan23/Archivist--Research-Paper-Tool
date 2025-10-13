package main

import (
	"archivist/internal/analyzer"
	"archivist/internal/app"
	"archivist/internal/cache"
	"archivist/internal/compiler"
	"archivist/internal/storage"
	"archivist/internal/tui"
	"archivist/internal/ui"
	"archivist/internal/wizard"
	"archivist/internal/worker"
	"archivist/pkg/fileutil"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	configPath  string
	force       bool
	parallel    int
	outputDir   string
	unprocessed bool
	mode        string // Processing mode: "fast" or "quality"
	interactive bool   // Enable interactive mode selection
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "rph",
		Short: "Research Paper Helper - Convert research papers to student-friendly LaTeX reports",
		Long: `Research Paper Helper analyzes AI/ML research papers using Gemini AI
and generates comprehensive, student-friendly LaTeX reports with detailed
explanations of methodologies, breakthroughs, and results.`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config/config.yaml", "config file path")

	// Process command
	processCmd := &cobra.Command{
		Use:   "process [file|directory]",
		Short: "Process research paper(s)",
		Long:  "Process a single PDF file or all PDF files in a directory",
		Args:  cobra.ExactArgs(1),
		Run:   runProcess,
	}
	processCmd.Flags().BoolVarP(&force, "force", "f", false, "reprocess even if already processed")
	processCmd.Flags().IntVarP(&parallel, "parallel", "p", 0, "number of parallel workers (default: config value)")
	processCmd.Flags().StringVarP(&mode, "mode", "m", "", "processing mode: 'fast' or 'quality' (default: interactive)")
	processCmd.Flags().BoolVarP(&interactive, "interactive", "i", true, "enable interactive mode selection")

	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List processed papers",
		Long:  "Display a list of all processed papers",
		Run:   runList,
	}
	listCmd.Flags().BoolVarP(&unprocessed, "unprocessed", "u", false, "show unprocessed papers")

	// Status command
	statusCmd := &cobra.Command{
		Use:   "status [file]",
		Short: "Show processing status",
		Long:  "Display processing status for a specific file",
		Args:  cobra.ExactArgs(1),
		Run:   runStatus,
	}

	// Clean command
	cleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean temporary files",
		Long:  "Remove auxiliary LaTeX files",
		Run:   runClean,
	}

	// Check command
	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Check dependencies",
		Long:  "Verify that all required tools (pdflatex, latexmk) are installed",
		Run:   runCheck,
	}

	// Run command - Interactive TUI
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Launch interactive TUI",
		Long:  "Start the interactive terminal UI for browsing and processing papers",
		Run:   runInteractive,
	}

	// Models command - List available Gemini models
	modelsCmd := &cobra.Command{
		Use:   "models",
		Short: "List available Gemini AI models",
		Long:  "Query the Gemini API to list all available models and find the best thinking model",
		Run:   runModels,
	}

	// Cache command - Manage Redis cache
	cacheCmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage analysis cache",
		Long:  "Manage the Redis cache for paper analysis results",
	}

	// Cache clear subcommand
	cacheClearCmd := &cobra.Command{
		Use:   "clear [file1.pdf] [file2.pdf] ...",
		Short: "Clear cached analysis results",
		Long:  "Remove all cached analysis results from Redis, or specific papers if file paths are provided",
		Run:   runCacheClear,
	}

	// Cache stats subcommand
	cacheStatsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show cache statistics",
		Long:  "Display statistics about cached analysis results",
		Run:   runCacheStats,
	}

	// Cache list subcommand
	cacheListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all cached papers",
		Long:  "Display all papers currently cached in Redis",
		Run:   runCacheList,
	}

	cacheCmd.AddCommand(cacheClearCmd, cacheStatsCmd, cacheListCmd)

	// Configure command - Interactive configuration wizard
	configureCmd := &cobra.Command{
		Use:   "configure",
		Short: "Interactive configuration wizard",
		Long:  "Launch an interactive wizard to set up your config.yaml file",
		Run:   runConfigure,
	}

	rootCmd.AddCommand(processCmd, listCmd, statusCmd, cleanCmd, checkCmd, runCmd, modelsCmd, cacheCmd, configureCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runProcess(cmd *cobra.Command, args []string) {
	// Show banner
	ui.ShowBanner()

	// Load config
	config, err := app.LoadConfig(configPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	// Initialize logger
	if err := app.InitLogger(config); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to initialize logger: %v", err))
		os.Exit(1)
	}

	// Select processing mode
	var selectedMode ui.ProcessingMode
	if mode != "" {
		// Mode specified via flag
		selectedMode = ui.ProcessingMode(mode)
	} else if interactive {
		// Interactive mode selection
		selectedMode, err = ui.PromptMode()
		if err != nil {
			ui.PrintError(fmt.Sprintf("Mode selection cancelled: %v", err))
			os.Exit(1)
		}
	} else {
		// Default to fast mode
		selectedMode = ui.ModeFast
	}

	// Apply mode configuration
	applyModeConfig(config, selectedMode)

	// Show mode details with actual config values
	ui.ShowModeDetailsWithConfig(selectedMode, config)

	// Override parallel workers if specified
	if parallel > 0 {
		config.Processing.MaxWorkers = parallel
		ui.PrintInfo(fmt.Sprintf("Using %d parallel workers (overridden)", parallel))
	} else {
		ui.PrintInfo(fmt.Sprintf("Using %d parallel workers", config.Processing.MaxWorkers))
	}

	// Check dependencies
	ui.PrintStage("Checking Dependencies", "Verifying LaTeX installation")
	if err := compiler.CheckDependencies(config.Latex.Engine == "latexmk", config.Latex.Compiler); err != nil {
		ui.PrintError(fmt.Sprintf("Dependency check failed: %v", err))
		fmt.Println("\nPlease install the required LaTeX tools:")
		fmt.Println("  sudo apt install texlive-latex-extra latexmk")
		os.Exit(1)
	}
	ui.PrintSuccess("All dependencies installed")

	// Get input path
	inputPath := args[0]

	// Determine if it's a file or directory
	info, err := os.Stat(inputPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to access path: %v", err))
		os.Exit(1)
	}

	var files []string
	if info.IsDir() {
		files, err = fileutil.GetPDFFiles(inputPath)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to get PDF files: %v", err))
			os.Exit(1)
		}
	} else {
		if filepath.Ext(inputPath) != ".pdf" {
			ui.PrintError("File must be a PDF")
			os.Exit(1)
		}
		files = []string{inputPath}
	}

	if len(files) == 0 {
		ui.PrintWarning("No PDF files found")
		return
	}

	ui.PrintInfo(fmt.Sprintf("Found %d PDF file(s)", len(files)))

	// Confirm processing
	if interactive && !ui.ConfirmProcessing(len(files)) {
		ui.PrintWarning("Processing cancelled by user")
		return
	}

	// Process files
	fmt.Println()
	ui.PrintStage("Processing Papers", "Starting batch processing")
	ctx := context.Background()
	if err := worker.ProcessBatch(ctx, files, config, force); err != nil {
		ui.PrintError(fmt.Sprintf("Processing failed: %v", err))
		fmt.Println()
		ui.PrintInfo("Press Enter to continue...")
		fmt.Scanln()
		os.Exit(1)
	}

	// After successful processing, offer to launch TUI
	fmt.Println()
	ui.PrintSuccess("All processing complete!")
	ui.PrintInfo("Would you like to:")
	fmt.Println("  1. Launch TUI to view papers")
	fmt.Println("  2. Exit")
	fmt.Print("\nChoice (1/2): ")

	var choice string
	fmt.Scanln(&choice)

	if choice == "1" {
		// Launch TUI
		if err := tui.Run(configPath); err != nil {
			ui.PrintError(fmt.Sprintf("TUI error: %v", err))
			os.Exit(1)
		}
	}
}

// applyModeConfig applies the selected mode's configuration
func applyModeConfig(config *app.Config, mode ui.ProcessingMode) {
	modes := ui.GetModeConfigs()
	modeConfig := modes[mode]

	config.Gemini.Agentic.Enabled = modeConfig.AgenticEnabled
	config.Gemini.Agentic.SelfReflection = modeConfig.SelfReflection
	config.Gemini.Agentic.MaxIterations = modeConfig.MaxIterations
	config.Gemini.Agentic.MultiStageAnalysis = modeConfig.MultiStageAnalysis
	config.Gemini.Agentic.Stages.LatexGeneration.Validation = modeConfig.ValidationEnabled
	config.Gemini.Model = modeConfig.Model

	// Use appropriate model for methodology analysis
	if mode == ui.ModeQuality {
		config.Gemini.Agentic.Stages.MethodologyAnalysis.Model = "models/gemini-2.0-flash-thinking-exp"
	} else {
		config.Gemini.Agentic.Stages.MethodologyAnalysis.Model = "models/gemini-2.0-flash-exp"
	}
}

func runList(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(configPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	metadataStore, err := storage.NewMetadataStore(config.MetadataDir)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load metadata: %v", err))
		os.Exit(1)
	}

	records := metadataStore.GetAllRecords()

	if unprocessed {
		// Show unprocessed files
		files, err := fileutil.GetPDFFiles(config.InputDir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to get PDF files: %v", err))
			os.Exit(1)
		}

		ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
		ui.ColorBold.Println("                   UNPROCESSED FILES                           ")
		ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
		fmt.Println()

		count := 0
		for _, file := range files {
			hash, _ := fileutil.ComputeFileHash(file)
			if !metadataStore.IsProcessed(hash) {
				count++
				ui.ColorWarning.Printf("  %d. %s\n", count, file)
			}
		}

		if count == 0 {
			ui.PrintSuccess("All files have been processed!")
		} else {
			fmt.Println()
			ui.PrintInfo(fmt.Sprintf("Total unprocessed: %d", count))
		}
		return
	}

	// Show processed files
	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	ui.ColorBold.Printf("              PROCESSED PAPERS (%d)                        \n", len(records))
	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	if len(records) == 0 {
		ui.PrintWarning("No papers have been processed yet")
		return
	}

	for i, record := range records {
		ui.ColorTitle.Printf("%d. %s\n", i+1, record.PaperTitle)
		ui.ColorSubtle.Printf("   File:      %s\n", record.FilePath)

		// Status with color
		switch record.Status {
		case storage.StatusCompleted:
			ui.ColorSuccess.Printf("   Status:    %s\n", record.Status)
		case storage.StatusFailed:
			ui.ColorError.Printf("   Status:    %s\n", record.Status)
		case storage.StatusProcessing:
			ui.ColorWarning.Printf("   Status:    %s\n", record.Status)
		default:
			fmt.Printf("   Status:    %s\n", record.Status)
		}

		ui.ColorSubtle.Printf("   Processed: %s\n", record.ProcessedAt.Format("2006-01-02 15:04:05"))

		if record.ReportPath != "" {
			ui.ColorInfo.Printf("   Report:    %s\n", record.ReportPath)
		}
		if record.Error != "" {
			ui.ColorError.Printf("   Error:     %s\n", record.Error)
		}
		fmt.Println()
	}
}

func runStatus(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(configPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	filePath := args[0]
	hash, err := fileutil.ComputeFileHash(filePath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to compute hash: %v", err))
		os.Exit(1)
	}

	metadataStore, err := storage.NewMetadataStore(config.MetadataDir)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load metadata: %v", err))
		os.Exit(1)
	}

	record, exists := metadataStore.GetRecord(hash)
	if !exists {
		ui.PrintWarning("Status: Not processed")
		return
	}

	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	ui.ColorBold.Println("                      FILE STATUS                              ")
	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	ui.ColorTitle.Printf("📄 Paper:     %s\n", record.PaperTitle)
	fmt.Println()

	switch record.Status {
	case storage.StatusCompleted:
		ui.ColorSuccess.Printf("✅ Status:    %s\n", record.Status)
	case storage.StatusFailed:
		ui.ColorError.Printf("❌ Status:    %s\n", record.Status)
	case storage.StatusProcessing:
		ui.ColorWarning.Printf("⏳ Status:    %s\n", record.Status)
	default:
		fmt.Printf("Status:       %s\n", record.Status)
	}

	ui.ColorSubtle.Printf("📅 Processed: %s\n", record.ProcessedAt.Format("2006-01-02 15:04:05"))

	if record.TexFilePath != "" {
		ui.ColorInfo.Printf("📝 LaTeX:     %s\n", record.TexFilePath)
	}
	if record.ReportPath != "" {
		ui.ColorInfo.Printf("📊 Report:    %s\n", record.ReportPath)
	}
	if record.Error != "" {
		fmt.Println()
		ui.ColorError.Printf("❌ Error:     %s\n", record.Error)
	}
	fmt.Println()
}

func runClean(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(configPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	ui.PrintStage("Cleaning", "Removing auxiliary LaTeX files")

	// Clean aux files from tex_files directory
	extensions := []string{"*.aux", "*.log", "*.out", "*.toc", "*.fdb_latexmk", "*.fls", "*.synctex.gz"}
	totalCleaned := 0

	for _, ext := range extensions {
		matches, _ := filepath.Glob(filepath.Join(config.TexOutputDir, ext))
		for _, file := range matches {
			if err := os.Remove(file); err == nil {
				totalCleaned++
			}
		}
	}

	ui.PrintSuccess(fmt.Sprintf("Cleaned %d auxiliary files", totalCleaned))
}

func runCheck(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(configPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	ui.PrintStage("Dependency Check", "Verifying system requirements")

	useLatexmk := config.Latex.Engine == "latexmk"
	if err := compiler.CheckDependencies(useLatexmk, config.Latex.Compiler); err != nil {
		ui.PrintError(fmt.Sprintf("%v", err))
		fmt.Println()
		ui.ColorWarning.Println("Please install the following:")
		if useLatexmk {
			fmt.Println("  • latexmk:        sudo apt install latexmk")
		}
		fmt.Printf("  • %s:  sudo apt install texlive-latex-extra\n", config.Latex.Compiler)
		fmt.Println()
		os.Exit(1)
	}

	ui.PrintSuccess("All dependencies installed")
	fmt.Println()
	ui.ColorInfo.Printf("  📦 LaTeX Compiler:  %s\n", config.Latex.Compiler)
	if useLatexmk {
		ui.ColorInfo.Println("  ⚙️  Build Tool:      latexmk")
	}
	ui.ColorInfo.Printf("  🔧 Workers:         %d\n", config.Processing.MaxWorkers)
	ui.ColorInfo.Printf("  🤖 AI Model:        %s\n", config.Gemini.Model)
	fmt.Println()
}

func runInteractive(cmd *cobra.Command, args []string) {
	// Launch interactive TUI
	if err := tui.Run(configPath); err != nil {
		ui.PrintError(fmt.Sprintf("TUI error: %v", err))
		os.Exit(1)
	}
}

func runModels(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(configPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	ui.PrintStage("Querying Gemini API", "Finding available models")

	// Create a temporary Gemini client to query models
	analyzer, err := analyzer.NewAnalyzer(config)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to create analyzer: %v", err))
		os.Exit(1)
	}
	defer analyzer.Close()

	ctx := context.Background()

	// List all available models
	ui.PrintInfo("Fetching list of available models...")
	models, err := analyzer.GetClient().ListAvailableModels(ctx)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to list models: %v", err))
		os.Exit(1)
	}

	ui.ColorBold.Println("\n═══════════════════════════════════════════════════════════════")
	ui.ColorBold.Printf("           AVAILABLE GEMINI MODELS (%d)                    \n", len(models))
	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	for i, model := range models {
		ui.ColorInfo.Printf("  %d. %s\n", i+1, model)
	}

	// Find best thinking model
	fmt.Println()
	ui.PrintInfo("Finding best thinking model...")
	thinkingModel, err := analyzer.GetClient().FindThinkingModel(ctx)
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Could not find thinking model: %v", err))
	} else {
		fmt.Println()
		ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
		ui.ColorSuccess.Printf("  RECOMMENDED THINKING MODEL: %s\n", thinkingModel)
		ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
		fmt.Println()
		ui.PrintInfo("To use this model, update your config/config.yaml:")
		ui.ColorSubtle.Printf("  gemini:\n")
		ui.ColorSubtle.Printf("    agentic:\n")
		ui.ColorSubtle.Printf("      stages:\n")
		ui.ColorSubtle.Printf("        methodology_analysis:\n")
		ui.ColorSubtle.Printf("          model: \"%s\"\n", thinkingModel)
		fmt.Println()
	}
}

func runCacheClear(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(configPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	if !config.Cache.Enabled {
		ui.PrintWarning("Cache is not enabled in config")
		ui.PrintInfo("To enable cache, set cache.enabled: true in config/config.yaml")
		return
	}

	if config.Cache.Type != "redis" {
		ui.PrintError("Only Redis cache is supported for clearing")
		return
	}

	ui.PrintStage("Connecting to Redis", fmt.Sprintf("Connecting to %s", config.Cache.Redis.Addr))

	// Connect to Redis
	ctx := context.Background()
	ttl := time.Duration(config.Cache.TTL) * time.Hour
	redisCache, err := cache.NewRedisCache(
		config.Cache.Redis.Addr,
		config.Cache.Redis.Password,
		config.Cache.Redis.DB,
		ttl,
	)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to connect to Redis: %v", err))
		fmt.Println()
		ui.PrintInfo("Make sure Redis is running:")
		ui.ColorSubtle.Println("  sudo systemctl start redis")
		ui.ColorSubtle.Println("  # or")
		ui.ColorSubtle.Println("  redis-server")
		os.Exit(1)
	}
	defer redisCache.Close()

	ui.PrintSuccess("Connected to Redis")
	fmt.Println()

	// If specific files are provided, clear only those
	if len(args) > 0 {
		ui.PrintStage("Clearing Specific Papers", fmt.Sprintf("Removing cache for %d file(s)", len(args)))

		successCount := 0
		failCount := 0

		for _, filePath := range args {
			// Compute hash for the file
			hash, err := fileutil.ComputeFileHash(filePath)
			if err != nil {
				ui.PrintError(fmt.Sprintf("❌ Failed to hash %s: %v", filePath, err))
				failCount++
				continue
			}

			// Delete from cache
			err = redisCache.Delete(ctx, hash)
			if err != nil {
				ui.PrintError(fmt.Sprintf("❌ Failed to clear cache for %s: %v", filePath, err))
				failCount++
			} else {
				ui.PrintSuccess(fmt.Sprintf("✅ Cleared cache for: %s", filepath.Base(filePath)))
				successCount++
			}
		}

		fmt.Println()
		if successCount > 0 {
			ui.PrintSuccess(fmt.Sprintf("Successfully cleared %d cached entries", successCount))
		}
		if failCount > 0 {
			ui.PrintWarning(fmt.Sprintf("%d entries failed to clear", failCount))
		}
		fmt.Println()
		ui.ColorInfo.Println("💡 These papers will be analyzed fresh on next processing")
		fmt.Println()
		return
	}

	// Clear all cache entries
	count, err := redisCache.GetStats(ctx)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to get stats: %v", err))
		os.Exit(1)
	}

	if count == 0 {
		ui.PrintInfo("Cache is already empty")
		return
	}

	ui.PrintWarning(fmt.Sprintf("Found %d cached entries", count))
	fmt.Println()
	ui.ColorWarning.Println("⚠️  This will permanently delete ALL cached analysis results!")
	fmt.Print("\nAre you sure? (yes/no): ")

	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "yes" {
		ui.PrintInfo("Cache clear cancelled")
		return
	}

	// Clear the cache
	fmt.Println()
	ui.PrintStage("Clearing Cache", "Removing all cached entries")

	deleted, err := redisCache.Clear(ctx)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to clear cache: %v", err))
		os.Exit(1)
	}

	ui.PrintSuccess(fmt.Sprintf("Successfully cleared %d cached entries", deleted))
	fmt.Println()
	ui.ColorInfo.Println("💡 Next time you process papers, they will be analyzed fresh")
	fmt.Println()
}

func runCacheStats(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(configPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	if !config.Cache.Enabled {
		ui.PrintWarning("Cache is not enabled in config")
		ui.PrintInfo("To enable cache, set cache.enabled: true in config/config.yaml")
		return
	}

	if config.Cache.Type != "redis" {
		ui.PrintError("Only Redis cache is supported for stats")
		return
	}

	ui.PrintStage("Connecting to Redis", fmt.Sprintf("Connecting to %s", config.Cache.Redis.Addr))

	// Connect to Redis
	ctx := context.Background()
	ttl := time.Duration(config.Cache.TTL) * time.Hour
	redisCache, err := cache.NewRedisCache(
		config.Cache.Redis.Addr,
		config.Cache.Redis.Password,
		config.Cache.Redis.DB,
		ttl,
	)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to connect to Redis: %v", err))
		fmt.Println()
		ui.PrintInfo("Make sure Redis is running:")
		ui.ColorSubtle.Println("  sudo systemctl start redis")
		ui.ColorSubtle.Println("  # or")
		ui.ColorSubtle.Println("  redis-server")
		os.Exit(1)
	}
	defer redisCache.Close()

	ui.PrintSuccess("Connected to Redis")
	fmt.Println()

	// Get stats
	count, err := redisCache.GetStats(ctx)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to get stats: %v", err))
		os.Exit(1)
	}

	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	ui.ColorBold.Println("                      CACHE STATISTICS                         ")
	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	ui.ColorInfo.Printf("  📊 Total Cached Papers:  %d\n", count)
	ui.ColorInfo.Printf("  🕒 Cache TTL:            %d hours (%.0f days)\n",
		config.Cache.TTL, float64(config.Cache.TTL)/24)
	ui.ColorInfo.Printf("  🔗 Redis Address:        %s\n", config.Cache.Redis.Addr)
	ui.ColorInfo.Printf("  🗄️  Redis Database:       %d\n", config.Cache.Redis.DB)
	fmt.Println()

	if count == 0 {
		ui.ColorSubtle.Println("  💡 Cache is empty. Process some papers to populate the cache!")
	} else {
		ui.ColorSuccess.Println("  💡 Cache is active and saving API costs!")
		ui.ColorSubtle.Printf("     Estimated savings: %d Gemini API calls avoided\n", count)
	}
	fmt.Println()

	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	ui.PrintInfo("To clear the cache, run: rph cache clear")
	fmt.Println()
}

func runCacheList(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(configPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	if !config.Cache.Enabled {
		ui.PrintWarning("Cache is not enabled in config")
		ui.PrintInfo("To enable cache, set cache.enabled: true in config/config.yaml")
		return
	}

	if config.Cache.Type != "redis" {
		ui.PrintError("Only Redis cache is supported for listing")
		return
	}

	ui.PrintStage("Connecting to Redis", fmt.Sprintf("Connecting to %s", config.Cache.Redis.Addr))

	// Connect to Redis
	ctx := context.Background()
	ttl := time.Duration(config.Cache.TTL) * time.Hour
	redisCache, err := cache.NewRedisCache(
		config.Cache.Redis.Addr,
		config.Cache.Redis.Password,
		config.Cache.Redis.DB,
		ttl,
	)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to connect to Redis: %v", err))
		fmt.Println()
		ui.PrintInfo("Make sure Redis is running:")
		ui.ColorSubtle.Println("  sudo systemctl start redis")
		ui.ColorSubtle.Println("  # or")
		ui.ColorSubtle.Println("  redis-server")
		os.Exit(1)
	}
	defer redisCache.Close()

	ui.PrintSuccess("Connected to Redis")
	fmt.Println()

	// List all cached entries
	ui.PrintStage("Fetching Cache", "Retrieving all cached papers")

	entries, err := redisCache.ListAll(ctx)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to list cache: %v", err))
		os.Exit(1)
	}

	if len(entries) == 0 {
		ui.PrintInfo("Cache is empty")
		fmt.Println()
		ui.ColorSubtle.Println("  💡 Process some papers to populate the cache!")
		fmt.Println()
		return
	}

	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	ui.ColorBold.Printf("              CACHED PAPERS (%d)                           \n", len(entries))
	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	for i, entry := range entries {
		ui.ColorTitle.Printf("%d. %s\n", i+1, entry.PaperTitle)
		ui.ColorSubtle.Printf("   Hash:      %s...%s\n", entry.ContentHash[:8], entry.ContentHash[len(entry.ContentHash)-8:])
		ui.ColorSubtle.Printf("   Model:     %s\n", entry.ModelUsed)
		ui.ColorSubtle.Printf("   Cached:    %s (%s ago)\n",
			entry.CachedAt.Format("2006-01-02 15:04:05"),
			time.Since(entry.CachedAt).Round(time.Minute))
		ui.ColorInfo.Printf("   Size:      %d chars\n", len(entry.LatexContent))
		fmt.Println()
	}

	ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	ui.PrintInfo("To clear specific papers, run: rph cache clear <file1.pdf> <file2.pdf>")
	ui.PrintInfo("To clear all cache, run: rph cache clear")
	fmt.Println()
}

func runConfigure(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	wiz := wizard.NewConfigWizard()
	if err := wiz.Run(configPath); err != nil {
		ui.PrintError(fmt.Sprintf("Configuration wizard failed: %v", err))
		os.Exit(1)
	}
}
