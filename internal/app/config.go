package app

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	InputDir         string           `mapstructure:"input_dir"`
	TexOutputDir     string           `mapstructure:"tex_output_dir"`
	ReportOutputDir  string           `mapstructure:"report_output_dir"`
	MetadataDir      string           `mapstructure:"metadata_dir"`
	Processing       ProcessingConfig `mapstructure:"processing"`
	Gemini           GeminiConfig     `mapstructure:"gemini"`
	Latex            LatexConfig      `mapstructure:"latex"`
	Cache            CacheConfig      `mapstructure:"cache"`
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

	// Load API key from environment
	config.Gemini.APIKey = os.Getenv("GEMINI_API_KEY")
	if config.Gemini.APIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not found in environment")
	}

	// Validate and create directories
	if err := ensureDirectories(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func ensureDirectories(config *Config) error {
	dirs := []string{
		config.InputDir,
		config.TexOutputDir,
		config.ReportOutputDir,
		config.MetadataDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
