package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// UserPreferences stores user-specific settings
type UserPreferences struct {
	InputDirectory  string `json:"input_directory"`
	OutputDirectory string `json:"output_directory"`
	ConfiguredOnce  bool   `json:"configured_once"`
}

// GetPreferencesPath returns the path to the preferences file
func GetPreferencesPath() string {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "archivist")
	os.MkdirAll(configDir, 0755)
	return filepath.Join(configDir, "preferences.json")
}

// LoadPreferences loads user preferences from disk
func LoadPreferences() (*UserPreferences, error) {
	prefsPath := GetPreferencesPath()

	// If preferences don't exist, return defaults
	if _, err := os.Stat(prefsPath); os.IsNotExist(err) {
		return &UserPreferences{
			InputDirectory:  "",
			OutputDirectory: "",
			ConfiguredOnce:  false,
		}, nil
	}

	data, err := os.ReadFile(prefsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read preferences: %w", err)
	}

	var prefs UserPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return nil, fmt.Errorf("failed to parse preferences: %w", err)
	}

	return &prefs, nil
}

// SavePreferences saves user preferences to disk
func SavePreferences(prefs *UserPreferences) error {
	prefsPath := GetPreferencesPath()

	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	if err := os.WriteFile(prefsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write preferences: %w", err)
	}

	return nil
}

// PromptForInitialSetup prompts user for initial directory setup
func PromptForInitialSetup() (*UserPreferences, error) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("         🎓 ARCHIVIST - First Time Setup                       ")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("Welcome to Archivist! Let's set up your directories.")
	fmt.Println()

	prefs := &UserPreferences{
		ConfiguredOnce: true,
	}

	// Get input directory
	fmt.Println("📥 INPUT DIRECTORY (where your PDF papers are stored)")
	fmt.Println("───────────────────────────────────────────────────────")
	inputDir, err := promptForDirectory("Enter input directory path", "./lib")
	if err != nil {
		return nil, err
	}
	prefs.InputDirectory = inputDir

	fmt.Println()

	// Get output directory
	fmt.Println("📤 OUTPUT DIRECTORY (where processed reports will be saved)")
	fmt.Println("───────────────────────────────────────────────────────")
	outputDir, err := promptForDirectory("Enter output directory path", "./reports")
	if err != nil {
		return nil, err
	}
	prefs.OutputDirectory = outputDir

	// Create directories
	fmt.Println()
	fmt.Println("Creating directories...")
	if err := os.MkdirAll(prefs.InputDirectory, 0755); err != nil {
		fmt.Printf("⚠️  Warning: Could not create input directory: %v\n", err)
	} else {
		fmt.Printf("✅ Created input directory: %s\n", prefs.InputDirectory)
	}

	if err := os.MkdirAll(prefs.OutputDirectory, 0755); err != nil {
		fmt.Printf("⚠️  Warning: Could not create output directory: %v\n", err)
	} else {
		fmt.Printf("✅ Created output directory: %s\n", prefs.OutputDirectory)
	}

	// Save preferences
	if err := SavePreferences(prefs); err != nil {
		fmt.Printf("⚠️  Warning: Could not save preferences: %v\n", err)
	} else {
		fmt.Println()
		fmt.Printf("✅ Preferences saved to: %s\n", GetPreferencesPath())
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("Setup complete! You can change these settings anytime from")
	fmt.Println("the Settings menu in the TUI.")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	return prefs, nil
}

// promptForDirectory prompts user for a directory path
func promptForDirectory(prompt, defaultPath string) (string, error) {
	fmt.Printf("\n%s\n", prompt)
	fmt.Printf("Default: %s\n", defaultPath)
	fmt.Print("Path (press Enter for default): ")

	var input string
	fmt.Scanln(&input)

	if input == "" {
		input = defaultPath
	}

	// Expand home directory
	if len(input) >= 2 && input[:2] == "~/" {
		home, _ := os.UserHomeDir()
		input = filepath.Join(home, input[2:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(input)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	return absPath, nil
}

// UpdateDirectories updates the directories in preferences
func UpdateDirectories(inputDir, outputDir string) error {
	prefs, err := LoadPreferences()
	if err != nil {
		prefs = &UserPreferences{}
	}

	if inputDir != "" {
		prefs.InputDirectory = inputDir
	}
	if outputDir != "" {
		prefs.OutputDirectory = outputDir
	}

	prefs.ConfiguredOnce = true

	return SavePreferences(prefs)
}
