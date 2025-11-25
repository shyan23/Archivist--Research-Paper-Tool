package app

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	InputDir         string           `mapstructure:"input_dir"`
	TexOutputDir     string           `mapstructure:"tex_output_dir"`
	ReportOutputDir  string           `mapstructure:"report_output_dir"`
	Processing       ProcessingConfig `mapstructure:"processing"`
	Gemini           GeminiConfig     `mapstructure:"gemini"`
	Latex            LatexConfig      `mapstructure:"latex"`
	Cache            CacheConfig      `mapstructure:"cache"`
	FAISS            FAISSConfig      `mapstructure:"faiss"`
	Graph            GraphConfig      `mapstructure:"graph"`
	Visualization    VisualizationConfig `mapstructure:"visualization"`
	HashAlgorithm    string           `mapstructure:"hash_algorithm"`
	Logging          LoggingConfig    `mapstructure:"logging"`
}

type ProcessingConfig struct {
	MaxWorkers       int `mapstructure:"max_workers"`
	BatchSize        int `mapstructure:"batch_size"`
	TimeoutPerPaper  int `mapstructure:"timeout_per_paper"`
}

type GeminiConfig struct {
	Model       string        `mapstructure:"model"`
	MaxTokens   int           `mapstructure:"max_tokens"`
	Temperature float64       `mapstructure:"temperature"`
	Agentic     AgenticConfig `mapstructure:"agentic"`
	APIKey      string        // Loaded from .env
}

type AgenticConfig struct {
	Enabled            bool          `mapstructure:"enabled"`
	MaxIterations      int           `mapstructure:"max_iterations"`
	SelfReflection     bool          `mapstructure:"self_reflection"`
	MultiStageAnalysis bool          `mapstructure:"multi_stage_analysis"`
	Stages             StagesConfig  `mapstructure:"stages"`
	Retry              RetryConfig   `mapstructure:"retry"`
}

type StagesConfig struct {
	MetadataExtraction   StageConfig `mapstructure:"metadata_extraction"`
	MethodologyAnalysis  StageConfig `mapstructure:"methodology_analysis"`
	LatexGeneration      StageConfig `mapstructure:"latex_generation"`
}

type StageConfig struct {
	Model           string  `mapstructure:"model"`
	Temperature     float64 `mapstructure:"temperature"`
	ThinkingBudget  int     `mapstructure:"thinking_budget"`
	Validation      bool    `mapstructure:"validation"`
}

type RetryConfig struct {
	MaxAttempts        int `mapstructure:"max_attempts"`
	BackoffMultiplier  int `mapstructure:"backoff_multiplier"`
	InitialDelayMs     int `mapstructure:"initial_delay_ms"`
}

type LatexConfig struct {
	Compiler  string `mapstructure:"compiler"`
	Engine    string `mapstructure:"engine"`
	CleanAux  bool   `mapstructure:"clean_aux"`
}

type LoggingConfig struct {
	Level   string `mapstructure:"level"`
	File    string `mapstructure:"file"`
	Console bool   `mapstructure:"console"`
}

type CacheConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Type     string `mapstructure:"type"`      // "redis" or "memory"
	Redis    RedisConfig `mapstructure:"redis"`
	TTL      int    `mapstructure:"ttl"`       // TTL in hours
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type FAISSConfig struct {
	IndexDir string `mapstructure:"index_dir"`
}

type GraphConfig struct {
	Enabled            bool                      `mapstructure:"enabled"`
	Neo4j              Neo4jConfig               `mapstructure:"neo4j"`
	AsyncBuilding      bool                      `mapstructure:"async_building"`
	MaxGraphWorkers    int                       `mapstructure:"max_graph_workers"`
	CitationExtraction CitationExtractionConfig  `mapstructure:"citation_extraction"`
	Search             SearchConfig              `mapstructure:"search"`
	Optimization       OptimizationConfig        `mapstructure:"optimization"`
}

type Neo4jConfig struct {
	URI      string `mapstructure:"uri"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
}

type CitationExtractionConfig struct {
	Enabled              bool     `mapstructure:"enabled"`
	PrioritizeInText     bool     `mapstructure:"prioritize_in_text"`
	ConfidenceThreshold  float64  `mapstructure:"confidence_threshold"`
	ImportanceFilter     []string `mapstructure:"importance_filter"`
}

type SearchConfig struct {
	DefaultTopK     int     `mapstructure:"default_top_k"`
	VectorWeight    float64 `mapstructure:"vector_weight"`
	GraphWeight     float64 `mapstructure:"graph_weight"`
	KeywordWeight   float64 `mapstructure:"keyword_weight"`
	TraversalDepth  int     `mapstructure:"traversal_depth"`
}

type OptimizationConfig struct {
	MaxPapersInMemory      int  `mapstructure:"max_papers_in_memory"`
	CacheGraphLayout       bool `mapstructure:"cache_graph_layout"`
	PrecomputeSimilarities bool `mapstructure:"precompute_similarities"`
}

type VisualizationConfig struct {
	Terminal TerminalVisualizationConfig `mapstructure:"terminal"`
	Web      WebVisualizationConfig      `mapstructure:"web"`
}

type TerminalVisualizationConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	MaxNodesDisplayed  int    `mapstructure:"max_nodes_displayed"`
	LayoutAlgorithm    string `mapstructure:"layout_algorithm"`
}

type WebVisualizationConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

// LoadConfig loads configuration from config.yaml and .env
func LoadConfig(configPath string) (*Config, error) {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: .env file not found, using environment variables")
	}

	// Setup Viper
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Load user preferences and override config directories
	prefs, err := LoadPreferences()
	if err == nil && prefs.ConfiguredOnce {
		// User has configured preferences, use them
		if prefs.InputDirectory != "" {
			config.InputDir = prefs.InputDirectory
		}
		if prefs.OutputDirectory != "" {
			config.ReportOutputDir = prefs.OutputDirectory
		}
	} else if err == nil && !prefs.ConfiguredOnce {
		// First time user - use defaults, they can configure in Settings
		// Save default preferences so we don't prompt again
		defaultPrefs := &UserPreferences{
			InputDirectory:  config.InputDir,
			OutputDirectory: config.ReportOutputDir,
			ConfiguredOnce:  true,
		}
		SavePreferences(defaultPrefs)
	}

	// Load API key from environment or prompt for it
	config.Gemini.APIKey = os.Getenv("GEMINI_API_KEY")
	if config.Gemini.APIKey == "" {
		fmt.Println()
		fmt.Println("═══════════════════════════════════════════════════════════════")
		fmt.Println("                    API KEY NOT FOUND                          ")
		fmt.Println("═══════════════════════════════════════════════════════════════")
		fmt.Println()
		fmt.Println("GEMINI_API_KEY not found in environment or .env file.")
		fmt.Println()
		fmt.Println("You can get your API key from:")
		fmt.Println("  https://aistudio.google.com/app/apikey")
		fmt.Println()

		apiKey, err := promptForAPIKey()
		if err != nil {
			return nil, fmt.Errorf("failed to get API key: %w", err)
		}

		config.Gemini.APIKey = apiKey

		// Ask if user wants to save to .env
		if shouldSave, _ := promptYesNo("Save API key to .env file? (y/n)"); shouldSave {
			if err := saveAPIKeyToEnv(apiKey); err != nil {
				fmt.Printf("Warning: Failed to save to .env: %v\n", err)
				fmt.Println("You can manually add it to .env file:")
				fmt.Printf("  GEMINI_API_KEY=%s\n", apiKey)
			} else {
				fmt.Println("✅ API key saved to .env file")
			}
		}
		fmt.Println()
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	// Validate and create directories
	if err := ensureDirectories(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// validateConfig validates the configuration values
func validateConfig(config *Config) error {
	// Validate MaxWorkers
	if config.Processing.MaxWorkers <= 0 {
		return fmt.Errorf("max_workers must be > 0, got %d", config.Processing.MaxWorkers)
	}
	if config.Processing.MaxWorkers > runtime.NumCPU() {
		return fmt.Errorf("max_workers (%d) exceeds available CPUs (%d)",
			config.Processing.MaxWorkers, runtime.NumCPU())
	}

	// Validate TimeoutPerPaper
	if config.Processing.TimeoutPerPaper <= 0 {
		return fmt.Errorf("timeout_per_paper must be > 0 seconds, got %d",
			config.Processing.TimeoutPerPaper)
	}

	// Validate Temperature
	if config.Gemini.Temperature < 0 || config.Gemini.Temperature > 2 {
		return fmt.Errorf("temperature must be in range [0, 2], got %.2f",
			config.Gemini.Temperature)
	}

	// Validate Model format
	if !strings.HasPrefix(config.Gemini.Model, "models/") {
		return fmt.Errorf("invalid model format: %s (must start with 'models/')",
			config.Gemini.Model)
	}

	// Validate MaxTokens
	if config.Gemini.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be > 0, got %d", config.Gemini.MaxTokens)
	}

	// Validate Cache TTL if caching is enabled
	if config.Cache.Enabled && config.Cache.TTL <= 0 {
		return fmt.Errorf("cache TTL must be > 0 hours when caching is enabled, got %d",
			config.Cache.TTL)
	}

	// Validate Cache Type
	if config.Cache.Enabled && config.Cache.Type != "redis" && config.Cache.Type != "memory" {
		return fmt.Errorf("cache type must be 'redis' or 'memory', got '%s'",
			config.Cache.Type)
	}

	// Validate Latex Compiler
	validCompilers := []string{"pdflatex", "xelatex", "lualatex"}
	isValidCompiler := false
	for _, valid := range validCompilers {
		if config.Latex.Compiler == valid {
			isValidCompiler = true
			break
		}
	}
	if !isValidCompiler {
		return fmt.Errorf("invalid latex compiler: %s (must be one of: %v)",
			config.Latex.Compiler, validCompilers)
	}

	// Validate Hash Algorithm
	validHashAlgos := []string{"sha256", "sha512", "md5"}
	isValidHash := false
	for _, valid := range validHashAlgos {
		if config.HashAlgorithm == valid {
			isValidHash = true
			break
		}
	}
	if !isValidHash {
		return fmt.Errorf("invalid hash_algorithm: %s (must be one of: %v)",
			config.HashAlgorithm, validHashAlgos)
	}

	return nil
}

func ensureDirectories(config *Config) error {
	dirs := []string{
		config.InputDir,
		config.TexOutputDir,
		config.ReportOutputDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// promptForAPIKey prompts the user to enter their API key
func promptForAPIKey() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your GEMINI_API_KEY: ")
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", fmt.Errorf("API key cannot be empty")
	}

	return apiKey, nil
}

// promptYesNo prompts for a yes/no answer
func promptYesNo(prompt string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt + " ")
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}

// saveAPIKeyToEnv saves the API key to the .env file
func saveAPIKeyToEnv(apiKey string) error {
	envPath := ".env"

	// Check if .env exists
	content := ""
	if data, err := os.ReadFile(envPath); err == nil {
		content = string(data)
	}

	// Check if GEMINI_API_KEY already exists in file
	lines := strings.Split(content, "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, "GEMINI_API_KEY=") {
			lines[i] = fmt.Sprintf("GEMINI_API_KEY=%s", apiKey)
			found = true
			break
		}
	}

	if !found {
		// Add new line if file doesn't end with newline
		if content != "" && !strings.HasSuffix(content, "\n") {
			lines = append(lines, "")
		}
		lines = append(lines, fmt.Sprintf("GEMINI_API_KEY=%s", apiKey))
	}

	// Write back to file
	newContent := strings.Join(lines, "\n")
	return os.WriteFile(envPath, []byte(newContent), 0644)
}
