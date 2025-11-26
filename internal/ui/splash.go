package ui

import (
	"fmt"
	"math"
	"os"
	"time"
)

// AnimatedSplash displays a Japanese elegant retro-style animated splash screen
func AnimatedSplash() {
	// Clear screen
	fmt.Print("\033[H\033[2J")

	// Hide cursor
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h") // Show cursor on exit

	// Matte Japanese-inspired color palette
	// Using 256-color ANSI codes for better terminal compatibility
	colors := map[string]string{
		"cream":      "\033[38;5;230m", // Soft cream background
		"beige":      "\033[38;5;180m", // Muted beige
		"sage":       "\033[38;5;108m", // Sage green
		"dustyBlue":  "\033[38;5;109m", // Dusty blue
		"softBrown":  "\033[38;5;137m", // Soft brown
		"charcoal":   "\033[38;5;240m", // Charcoal gray
		"warmGray":   "\033[38;5;145m", // Warm gray
		"mutedRust":  "\033[38;5;173m", // Muted rust/terracotta
		"paleGreen":  "\033[38;5;151m", // Pale green
		"reset":      "\033[0m",
	}

	// Simple animation frames - minimalist landscape
	totalFrames := 24

	for frame := range totalFrames {
		fmt.Print("\033[H") // Move cursor to top

		// Render the minimalist Japanese landscape
		renderMinimalistLandscape(colors, frame)

		time.Sleep(100 * time.Millisecond)
	}

	// Final clear
	time.Sleep(400 * time.Millisecond)
	fmt.Print("\033[H\033[2J")
}

// renderMinimalistLandscape creates a zen-like landscape with geometric mountains
func renderMinimalistLandscape(colors map[string]string, frame int) {
	width := 76
	height := 24

	// Sun/moon position (simple circle that fades in)
	sunVisible := frame > 8
	sunOpacity := int(math.Min(float64(frame-8)*20, 100))

	// Print top border
	fmt.Print(colors["charcoal"] + "┌" + repeat("─", width-2) + "┐" + colors["reset"] + "\n")

	// Sky section (top third)
	for y := 0; y < height/3; y++ {
		fmt.Print(colors["charcoal"] + "│" + colors["reset"])

		// Simple sun/moon in upper right
		if sunVisible && y == 3 && sunOpacity > 50 {
			sunLine := repeat(" ", width-18) + colors["warmGray"] + "◯" + colors["reset"] + repeat(" ", 15)
			fmt.Print(sunLine[:width-2])
		} else {
			fmt.Print(repeat(" ", width-2))
		}

		fmt.Print(colors["charcoal"] + "│" + colors["reset"] + "\n")
	}

	// Mountain layers (middle section) - geometric and minimal
	mountainHeight := height / 3
	for y := range mountainHeight {
		fmt.Print(colors["charcoal"] + "│" + colors["reset"])

		// Build mountain layers with proper priority (near > mid > far)
		nearMountain := []bool{}
		midMountain := []bool{}
		farMountain := []bool{}

		// Generate all layers
		if y >= 0 {
			nearMountain = generateMountainMask(width-2, y, mountainHeight, 55, 10)
		}
		if y >= 1 {
			midMountain = generateMountainMask(width-2, y-1, mountainHeight-1, 45, 15)
		}
		if y >= 2 {
			farMountain = generateMountainMask(width-2, y-2, mountainHeight-2, 30, 20)
		}

		// Render each position with correct color priority
		for x := 0; x < width-2; x++ {
			if len(nearMountain) > x && nearMountain[x] {
				fmt.Print(colors["sage"] + "█" + colors["reset"])
			} else if len(midMountain) > x && midMountain[x] {
				fmt.Print(colors["dustyBlue"] + "█" + colors["reset"])
			} else if len(farMountain) > x && farMountain[x] {
				fmt.Print(colors["warmGray"] + "▓" + colors["reset"])
			} else {
				fmt.Print(" ")
			}
		}

		fmt.Print(colors["charcoal"] + "│" + colors["reset"] + "\n")
	}

	// Title section (lower third)
	titleStart := height/3 + mountainHeight
	currentLine := titleStart

	for y := 0; y < height-titleStart-1; y++ {
		fmt.Print(colors["charcoal"] + "│" + colors["reset"])

		switch y {
		case 2:
			title := center("ARCHIVIST", width-2)
			fmt.Print(colors["softBrown"] + title + colors["reset"])
		case 4:
			subtitle := center("research paper helper", width-2)
			fmt.Print(colors["beige"] + subtitle + colors["reset"])
		case 6:
			// Animated loading indicator (minimalist)
			dotCount := (frame % 4)
			dots := ""
			for range dotCount {
				dots += "·"
			}
			loading := center(dots, width-2)
			fmt.Print(colors["dustyBlue"] + loading + colors["reset"])
		default:
			fmt.Print(repeat(" ", width-2))
		}

		fmt.Print(colors["charcoal"] + "│" + colors["reset"] + "\n")
		currentLine++
	}

	// Bottom border
	fmt.Print(colors["charcoal"] + "└" + repeat("─", width-2) + "┘" + colors["reset"] + "\n")
}

// generateMountainMask creates a boolean mask for mountain positions
func generateMountainMask(width, currentY, totalHeight, peakX, spread int) []bool {
	mask := make([]bool, width)

	// Calculate mountain slope
	leftEdge := peakX - (currentY * spread / totalHeight)
	rightEdge := peakX + (currentY * spread / totalHeight)

	if leftEdge < 0 {
		leftEdge = 0
	}
	if rightEdge >= width {
		rightEdge = width - 1
	}

	// Fill mountain area
	for x := leftEdge; x <= rightEdge; x++ {
		if x >= 0 && x < width {
			mask[x] = true
		}
	}

	return mask
}

// Helper functions
func repeat(s string, count int) string {
	result := ""
	for range count {
		result += s
	}
	return result
}

func center(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	leftPad := (width - len(s)) / 2
	rightPad := width - len(s) - leftPad
	return repeat(" ", leftPad) + s + repeat(" ", rightPad)
}

// ShowWelcomeMessage displays a welcome message after splash
func ShowWelcomeMessage(inputDir, outputDir string) {
	fmt.Print("\033[H\033[2J") // Clear screen

	// Muted color palette matching the splash screen
	charcoal := "\033[38;5;240m"
	softBrown := "\033[38;5;137m"
	beige := "\033[38;5;180m"
	sage := "\033[38;5;108m"
	dustyBlue := "\033[38;5;109m"
	warmGray := "\033[38;5;145m"
	reset := "\033[0m"

	// Print minimalist welcome box
	fmt.Println()
	fmt.Println(charcoal + "┌" + repeat("─", 76) + "┐" + reset)
	fmt.Println(charcoal + "│" + reset + center("", 76) + charcoal + "│" + reset)
	fmt.Println(charcoal + "│" + reset + center(softBrown+"Welcome to ARCHIVIST"+reset, 90) + charcoal + "│" + reset)
	fmt.Println(charcoal + "│" + reset + center("", 76) + charcoal + "│" + reset)
	fmt.Println(charcoal + "└" + repeat("─", 76) + "┘" + reset)
	fmt.Println()

	// Show configured directories
	fmt.Println(sage + "  Configuration Loaded" + reset)
	fmt.Println()

	// Shorten paths for display
	displayInput := inputDir
	if len(displayInput) > 50 {
		displayInput = "..." + displayInput[len(displayInput)-47:]
	}

	displayOutput := outputDir
	if len(displayOutput) > 50 {
		displayOutput = "..." + displayOutput[len(displayOutput)-47:]
	}

	fmt.Printf("  "+dustyBlue+"Input Directory:"+reset+"  %s\n", displayInput)
	fmt.Printf("  "+beige+"Output Directory:"+reset+" %s\n", displayOutput)
	fmt.Println()

	// Check if directories exist
	inputExists := fileExists(inputDir)
	outputExists := fileExists(outputDir)

	if !inputExists || !outputExists {
		fmt.Println(warmGray + "  Notice:" + reset)
		if !inputExists {
			fmt.Printf("  · Input directory will be created: "+charcoal+"%s"+reset+"\n", inputDir)
		}
		if !outputExists {
			fmt.Printf("  · Output directory will be created: "+charcoal+"%s"+reset+"\n", outputDir)
		}
		fmt.Println()
	}

	fmt.Println(charcoal + "  Tip: Change directories anytime in Settings → Directory Configuration" + reset)
	fmt.Println()
	fmt.Println(charcoal + repeat("─", 76) + reset)
	fmt.Println()
}

// fileExists checks if a file or directory exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
