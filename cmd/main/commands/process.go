package commands

import (
	"archivist/internal/app"
	"archivist/internal/compiler"
	"archivist/internal/profiler"
	"archivist/internal/tui"
	"archivist/internal/ui"
	"archivist/internal/worker"
	"archivist/pkg/fileutil"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	force       bool
	parallel    int
	mode        string
	interactive bool
	selectPapers bool
)

// NewProcessCommand creates the process command
func NewProcessCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "process [file|directory]",
		Short: "Process research paper(s)",
		Long:  "Process a single PDF file, all PDF files in a directory, or interactively select papers to process",
		Args:  cobra.MaximumNArgs(1),
		Run:   runProcess,
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "reprocess even if already processed")
	cmd.Flags().IntVarP(&parallel, "parallel", "p", 0, "number of parallel workers (default: config value)")
	cmd.Flags().StringVarP(&mode, "mode", "m", "", "processing mode: 'fast' (default: interactive)")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", true, "enable interactive mode selection")
	cmd.Flags().BoolVarP(&selectPapers, "select", "s", false, "interactively select papers to process from library")

	return cmd
}

func runProcess(cmd *cobra.Command, args []string) {
	// Initialize profiler if enabled
	if EnableProfile {
		profConfig := &profiler.ProfileConfig{
			Enabled:    true,
			OutputDir:  ProfileDir,
			CPUProfile: true,
			MemProfile: true,
			FuncTiming: true,
		}
		prof, err := profiler.NewProfiler(profConfig)
		if err != nil {
			fmt.Printf("Failed to initialize profiler: %v\n", err)
			os.Exit(1)
		}
		defer func() {
			if err := prof.Stop(); err != nil {
				fmt.Printf("Failed to stop profiler: %v\n", err)
			}
			prof.PrintTimings()
			profiler.PrintMemoryStats()
		}()
		if err := prof.Start(); err != nil {
			fmt.Printf("Failed to start profiler: %v\n", err)
			os.Exit(1)
		}
	}

	// Show banner
	ui.ShowBanner()

	// Load config
	config, err := app.LoadConfig(ConfigPath)
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
		selectedMode = ui.ProcessingMode(mode)
	} else if interactive {
		selectedMode, err = ui.PromptMode()
		if err != nil {
			ui.PrintError(fmt.Sprintf("Mode selection cancelled: %v", err))
			os.Exit(1)
		}
	} else {
		selectedMode = ui.ModeFast
	}

	// Apply mode configuration
	applyModeConfig(config, selectedMode)

	// Show mode details
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

	// Get files to process
	var files []string

	if selectPapers {
		// Interactive paper selection mode
		if len(args) > 0 {
			ui.PrintWarning("Ignoring file/directory argument when using --select flag")
		}

		// Get all papers from library
		allPapers, err := fileutil.GetPDFFiles(config.InputDir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to get PDF files: %v", err))
			os.Exit(1)
		}

		if len(allPapers) == 0 {
			ui.PrintWarning("No PDF files found in library")
			return
		}

		// Let user select papers
		basenames := make([]string, len(allPapers))
		for i, paper := range allPapers {
			basenames[i] = filepath.Base(paper)
		}

		selectedBasenames, err := ui.PromptSelectPapers(basenames)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Paper selection failed: %v", err))
			os.Exit(1)
		}

		// Map selected basenames back to full paths
		for _, basename := range selectedBasenames {
			for _, paper := range allPapers {
				if filepath.Base(paper) == basename {
					files = append(files, paper)
					break
				}
			}
		}
	} else if len(args) == 0 {
		// No arguments and no --select flag: process all papers in library
		files, err = fileutil.GetPDFFiles(config.InputDir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to get PDF files: %v", err))
			os.Exit(1)
		}
	} else {
		// Process specified file or directory
		inputPath := args[0]

		// Determine if it's a file or directory
		info, err := os.Stat(inputPath)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to access path: %v", err))
			os.Exit(1)
		}

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
		if err := tui.Run(ConfigPath); err != nil {
			ui.PrintError(fmt.Sprintf("TUI error: %v", err))
			os.Exit(1)
		}
	}
}

func applyModeConfig(config *app.Config, mode ui.ProcessingMode) {
	modes := ui.GetModeConfigs()
	modeConfig := modes[mode]

	config.Gemini.Agentic.Enabled = modeConfig.AgenticEnabled
	config.Gemini.Agentic.SelfReflection = modeConfig.SelfReflection
	config.Gemini.Agentic.MaxIterations = modeConfig.MaxIterations
	config.Gemini.Agentic.MultiStageAnalysis = modeConfig.MultiStageAnalysis
	config.Gemini.Agentic.Stages.LatexGeneration.Validation = modeConfig.ValidationEnabled
	config.Gemini.Model = modeConfig.Model

	// Use fast model for methodology analysis (only one mode now)
	config.Gemini.Agentic.Stages.MethodologyAnalysis.Model = "models/gemini-2.0-flash-exp"
}
