package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// NewSetupCommand creates the setup command
func NewSetupCommand() *cobra.Command {
	var skipServices bool
	var autoYes bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Bootstrap and setup Archivist with all dependencies",
		Long: `Automatically checks for and installs all required dependencies:
  • Go, Python, Docker, LaTeX, Git, Make
  • Go modules and Python packages
  • Docker images (Neo4j, Qdrant, Redis, Kafka)
  • Project structure and configuration files

This command handles everything from scratch!`,
		Aliases: []string{"bootstrap", "init"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetup(skipServices, autoYes)
		},
	}

	cmd.Flags().BoolVar(&skipServices, "skip-services", false, "skip starting Docker services")
	cmd.Flags().BoolVarP(&autoYes, "yes", "y", false, "automatically answer yes to all prompts")

	return cmd
}

func runSetup(skipServices, autoYes bool) error {
	// Check if bootstrap script exists
	scriptPath := "./scripts/bootstrap.sh"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("bootstrap script not found at %s", scriptPath)
	}

	// Show welcome message
	printSetupBanner()

	// Confirm before proceeding
	if !autoYes {
		fmt.Println("\n⚠️  This will install system dependencies and may require sudo password.")
		fmt.Print("Do you want to continue? [Y/n]: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "n" || response == "no" {
			fmt.Println("\n❌ Setup cancelled")
			return nil
		}
	}

	// Set environment variables for the script
	env := os.Environ()
	if skipServices {
		env = append(env, "SKIP_SERVICES=true")
	}
	if autoYes {
		env = append(env, "AUTO_YES=true")
	}

	// Execute bootstrap script
	cmdExec := exec.Command("/bin/bash", scriptPath)
	cmdExec.Env = env
	cmdExec.Stdout = os.Stdout
	cmdExec.Stderr = os.Stderr
	cmdExec.Stdin = os.Stdin

	fmt.Println("\n🚀 Starting bootstrap process...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	startTime := time.Now()

	if err := cmdExec.Run(); err != nil {
		fmt.Printf("\n❌ Setup failed: %v\n", err)
		return err
	}

	duration := time.Since(startTime)

	// Show completion message
	printSetupComplete(duration)

	return nil
}

func printSetupBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║     █████╗ ██████╗  ██████╗██╗  ██╗██╗██╗   ██╗██╗███████╗   ║
║    ██╔══██╗██╔══██╗██╔════╝██║  ██║██║██║   ██║██║██╔════╝   ║
║    ███████║██████╔╝██║     ███████║██║██║   ██║██║███████╗   ║
║    ██╔══██║██╔══██╗██║     ██╔══██║██║╚██╗ ██╔╝██║╚════██║   ║
║    ██║  ██║██║  ██║╚██████╗██║  ██║██║ ╚████╔╝ ██║███████║   ║
║    ╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═══╝  ╚═╝╚══════╝   ║
║                                                               ║
║               AUTOMATED SETUP & BOOTSTRAP                     ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
`
	fmt.Print(banner)

	fmt.Println("\n📦 What will be installed:")
	fmt.Println("   ✓ System dependencies (Go, Python, Docker, LaTeX)")
	fmt.Println("   ✓ Go modules and packages")
	fmt.Println("   ✓ Python virtual environment and packages")
	fmt.Println("   ✓ Docker images (Neo4j, Qdrant, Redis, Kafka)")
	fmt.Println("   ✓ Project structure and configuration files")
	fmt.Println("   ✓ Archivist binary compilation")
}

func printSetupComplete(duration time.Duration) {
	complete := `
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║                   ✅ SETUP COMPLETE! ✅                        ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
`
	fmt.Println(complete)
	fmt.Printf("⏱️  Total setup time: %s\n\n", duration.Round(time.Second))

	fmt.Println("🎯 Next Steps:")
	fmt.Println()
	fmt.Println("1. Add your Gemini API key to .env:")
	fmt.Println("   export GEMINI_API_KEY='your-api-key-here'")
	fmt.Println("   Or edit: nano .env")
	fmt.Println()
	fmt.Println("2. Verify installation:")
	fmt.Println("   ./archivist check")
	fmt.Println()
	fmt.Println("3. Try the interactive TUI:")
	fmt.Println("   ./archivist run")
	fmt.Println()
	fmt.Println("4. Or process a paper:")
	fmt.Println("   ./archivist process lib/your_paper.pdf")
	fmt.Println()
	fmt.Println("📚 Documentation:")
	fmt.Println("   • README.md - Quick start guide")
	fmt.Println("   • docs/features/KNOWLEDGE_GRAPH_GUIDE.md - Graph database")
	fmt.Println("   • docs/features/SEARCH_ENGINE_GUIDE.md - Search engine")
	fmt.Println()
	fmt.Println("🌐 Services:")
	fmt.Println("   • Neo4j: http://localhost:7474 (neo4j/password)")
	fmt.Println("   • Qdrant: http://localhost:6333/dashboard")
	fmt.Println("   • Redis: localhost:6379")
	fmt.Println()
}
