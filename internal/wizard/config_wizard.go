package wizard

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/manifoldco/promptui"
	"gopkg.in/yaml.v3"
)

// ConfigWizard represents the configuration wizard
type ConfigWizard struct {
	config map[string]interface{}
}

// NewConfigWizard creates a new configuration wizard
func NewConfigWizard() *ConfigWizard {
	return &ConfigWizard{
		config: make(map[string]interface{}),
	}
}

// Run starts the interactive configuration wizard
func (cw *ConfigWizard) Run(configPath string) error {
	fmt.Println()
	fmt.Println("🧙 Archivist Configuration Wizard")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println()
	fmt.Println("This wizard will help you set up your config.yaml file.")
	fmt.Println()

	// Step 1: Directory Configuration
	if err := cw.configureDirectories(); err != nil {
		return err
	}

	// Step 2: Processing Configuration
	if err := cw.configureProcessing(); err != nil {
		return err
	}

	// Step 3: Gemini Configuration
	if err := cw.configureGemini(); err != nil {
		return err
	}

	// Step 4: LaTeX Configuration
	if err := cw.configureLatex(); err != nil {
		return err
	}

	// Step 5: Cache Configuration
	if err := cw.configureCache(); err != nil {
		return err
	}

	// Step 6: Logging Configuration
	if err := cw.configureLogging(); err != nil {
		return err
	}

	// Save configuration
	return cw.saveConfig(configPath)
}

func (cw *ConfigWizard) configureDirectories() error {
	fmt.Println("📁 Directory Configuration")
	fmt.Println("───────────────────────────")
	fmt.Println()

	inputDir, err := cw.promptString("Input directory for PDF papers", "./lib")
	if err != nil {
		return err
	}

	texOutputDir, err := cw.promptString("Output directory for LaTeX files", "./tex_files")
	if err != nil {
		return err
	}

	reportOutputDir, err := cw.promptString("Output directory for PDF reports", "./reports")
	if err != nil {
		return err
	}

	cw.config["input_dir"] = inputDir
	cw.config["tex_output_dir"] = texOutputDir
	cw.config["report_output_dir"] = reportOutputDir

	fmt.Println()
	return nil
}

func (cw *ConfigWizard) configureProcessing() error {
	fmt.Println("⚙️  Processing Configuration")
	fmt.Println("───────────────────────────")
	fmt.Println()

	maxWorkers, err := cw.promptInt("Maximum parallel workers", 8)
	if err != nil {
		return err
	}

	batchSize, err := cw.promptInt("Batch size", 10)
	if err != nil {
		return err
	}

	timeoutPerPaper, err := cw.promptInt("Timeout per paper (seconds)", 600)
	if err != nil {
		return err
	}

	cw.config["processing"] = map[string]interface{}{
		"max_workers":       maxWorkers,
		"batch_size":        batchSize,
		"timeout_per_paper": timeoutPerPaper,
	}

	fmt.Println()
	return nil
}

func (cw *ConfigWizard) configureGemini() error {
	fmt.Println("🤖 Gemini AI Configuration")
	fmt.Println("───────────────────────────")
	fmt.Println()

	model, err := cw.promptSelect("Default Gemini model",
		[]string{"gemini-1.5-flash", "gemini-1.5-pro", "gemini-2.0-flash-exp"})
	if err != nil {
		return err
	}

	maxTokens, err := cw.promptInt("Maximum tokens", 8000)
	if err != nil {
		return err
	}

	temperature, err := cw.promptFloat("Temperature (0.0-1.0)", 0.3)
	if err != nil {
		return err
	}

	agenticEnabled, err := cw.promptConfirm("Enable agentic workflow? (multi-stage analysis)")
	if err != nil {
		return err
	}

	cw.config["gemini"] = map[string]interface{}{
		"model":       model,
		"max_tokens":  maxTokens,
		"temperature": temperature,
		"agentic": map[string]interface{}{
			"enabled":             agenticEnabled,
			"max_iterations":      1,
			"self_reflection":     false,
			"multi_stage_analysis": false,
			"stages": map[string]interface{}{
				"metadata_extraction": map[string]interface{}{
					"model":       "gemini-1.5-flash",
					"temperature": 1.0,
				},
				"methodology_analysis": map[string]interface{}{
					"model":           "models/gemini-2.0-flash-thinking-exp",
					"temperature":     1.0,
					"thinking_budget": 10000,
				},
				"latex_generation": map[string]interface{}{
					"model":       "gemini-1.5-flash",
					"temperature": 1.0,
					"validation":  true,
				},
			},
			"retry": map[string]interface{}{
				"max_attempts":        3,
				"backoff_multiplier":  2,
				"initial_delay_ms":    1000,
			},
		},
	}

	fmt.Println()
	return nil
}

func (cw *ConfigWizard) configureLatex() error {
	fmt.Println("📄 LaTeX Configuration")
	fmt.Println("───────────────────────────")
	fmt.Println()

	compiler, err := cw.promptSelect("LaTeX compiler",
		[]string{"pdflatex", "xelatex", "lualatex"})
	if err != nil {
		return err
	}

	engine, err := cw.promptSelect("Build engine",
		[]string{"latexmk", "direct"})
	if err != nil {
		return err
	}

	cleanAux, err := cw.promptConfirm("Clean auxiliary files after compilation?")
	if err != nil {
		return err
	}

	cw.config["latex"] = map[string]interface{}{
		"compiler":  compiler,
		"engine":    engine,
		"clean_aux": cleanAux,
	}

	fmt.Println()
	return nil
}

func (cw *ConfigWizard) configureCache() error {
	fmt.Println("💾 Cache Configuration")
	fmt.Println("───────────────────────────")
	fmt.Println()

	enabled, err := cw.promptConfirm("Enable Redis caching?")
	if err != nil {
		return err
	}

	var redisAddr, redisPassword string
	var redisDB, ttl int

	if enabled {
		redisAddr, err = cw.promptString("Redis address", "localhost:6379")
		if err != nil {
			return err
		}

		redisPassword, err = cw.promptString("Redis password (leave empty for no auth)", "")
		if err != nil {
			return err
		}

		redisDB, err = cw.promptInt("Redis database number", 0)
		if err != nil {
			return err
		}

		ttl, err = cw.promptInt("Cache TTL in hours", 720)
		if err != nil {
			return err
		}
	}

	cw.config["cache"] = map[string]interface{}{
		"enabled": enabled,
		"type":    "redis",
		"ttl":     ttl,
		"redis": map[string]interface{}{
			"addr":     redisAddr,
			"password": redisPassword,
			"db":       redisDB,
		},
	}

	cw.config["hash_algorithm"] = "md5"

	fmt.Println()
	return nil
}

func (cw *ConfigWizard) configureLogging() error {
	fmt.Println("📝 Logging Configuration")
	fmt.Println("───────────────────────────")
	fmt.Println()

	level, err := cw.promptSelect("Log level",
		[]string{"debug", "info", "warn", "error"})
	if err != nil {
		return err
	}

	logFile, err := cw.promptString("Log file path", "./logs/processing.log")
	if err != nil {
		return err
	}

	console, err := cw.promptConfirm("Enable console logging?")
	if err != nil {
		return err
	}

	cw.config["logging"] = map[string]interface{}{
		"level":   level,
		"file":    logFile,
		"console": console,
	}

	fmt.Println()
	return nil
}

func (cw *ConfigWizard) promptString(label, defaultValue string) (string, error) {
	prompt := promptui.Prompt{
		Label:   label,
		Default: defaultValue,
	}

	return prompt.Run()
}

func (cw *ConfigWizard) promptInt(label string, defaultValue int) (int, error) {
	prompt := promptui.Prompt{
		Label:   label,
		Default: fmt.Sprintf("%d", defaultValue),
		Validate: func(s string) error {
			_, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("invalid number")
			}
			return nil
		},
	}

	result, err := prompt.Run()
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(result)
}

func (cw *ConfigWizard) promptFloat(label string, defaultValue float64) (float64, error) {
	prompt := promptui.Prompt{
		Label:   label,
		Default: fmt.Sprintf("%.1f", defaultValue),
		Validate: func(s string) error {
			_, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return fmt.Errorf("invalid number")
			}
			return nil
		},
	}

	result, err := prompt.Run()
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(result, 64)
}

func (cw *ConfigWizard) promptConfirm(label string) (bool, error) {
	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
	}

	_, err := prompt.Run()
	if err != nil {
		// Promptui returns an error for "No" responses
		if strings.Contains(err.Error(), "^C") {
			return false, err
		}
		return false, nil
	}

	return true, nil
}

func (cw *ConfigWizard) promptSelect(label string, items []string) (string, error) {
	prompt := promptui.Select{
		Label: label,
		Items: items,
	}

	_, result, err := prompt.Run()
	return result, err
}

func (cw *ConfigWizard) saveConfig(configPath string) error {
	fmt.Println("💾 Saving Configuration")
	fmt.Println("───────────────────────────")
	fmt.Println()

	data, err := yaml.Marshal(cw.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("✅ Configuration saved to: %s\n", configPath)
	fmt.Println()
	fmt.Println("You can now run:")
	fmt.Println("  rph run      # Launch interactive TUI")
	fmt.Println("  rph process ./lib  # Process papers")
	fmt.Println()

	return nil
}
