package commands

import (
	"archivist/internal/app"
	"archivist/internal/ui"
	"archivist/pkg/fileutil"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var showReports bool

// NewListCommand creates the list command
func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List papers",
		Long:  "Display a list of input PDFs or generated reports",
		Run:   runList,
	}

	cmd.Flags().BoolVarP(&showReports, "reports", "r", false, "show generated reports instead of input files")

	return cmd
}

func runList(cmd *cobra.Command, args []string) {
	ui.ShowBanner()

	config, err := app.LoadConfig(ConfigPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	if showReports {
		// Show generated reports
		files, err := fileutil.GetPDFFiles(config.ReportOutputDir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to get reports: %v", err))
			os.Exit(1)
		}

		ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
		ui.ColorBold.Printf("              GENERATED REPORTS (%d)                        \n", len(files))
		ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
		fmt.Println()

		if len(files) == 0 {
			ui.PrintWarning("No reports have been generated yet")
			return
		}

		for i, file := range files {
			basename := filepath.Base(file)
			ui.ColorTitle.Printf("%d. %s\n", i+1, basename)
			ui.ColorSubtle.Printf("   Path: %s\n", file)
			fmt.Println()
		}
	} else {
		// Show input files
		files, err := fileutil.GetPDFFiles(config.InputDir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to get PDF files: %v", err))
			os.Exit(1)
		}

		ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
		ui.ColorBold.Printf("              INPUT PAPERS (%d)                        \n", len(files))
		ui.ColorBold.Println("═══════════════════════════════════════════════════════════════")
		fmt.Println()

		if len(files) == 0 {
			ui.PrintWarning("No PDF files found in library")
			return
		}

		for i, file := range files {
			basename := filepath.Base(file)
			ui.ColorTitle.Printf("%d. %s\n", i+1, basename)
			ui.ColorSubtle.Printf("   Path: %s\n", file)
			fmt.Println()
		}
	}
}
