package analyzer

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiClient struct {
	client      *genai.Client
	model       string
	temperature float64
	maxTokens   int
}

// NewGeminiClient creates a new Gemini API client
func NewGeminiClient(apiKey, model string, temperature float64, maxTokens int) (*GeminiClient, error) {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &GeminiClient{
		client:      client,
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
	}, nil
}

// Close closes the Gemini client
func (gc *GeminiClient) Close() error {
	return gc.client.Close()
}

// GenerateText generates text from a prompt
func (gc *GeminiClient) GenerateText(ctx context.Context, prompt string) (string, error) {
	model := gc.client.GenerativeModel(gc.model)

	// Configure model parameters
	model.SetTemperature(float32(gc.temperature))
	model.SetMaxOutputTokens(int32(gc.maxTokens))

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates returned")
	}

	if resp.Candidates[0].Content == nil {
		return "", fmt.Errorf("no content in response")
	}

	// Extract text from response
	var result string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			result += string(txt)
		}
	}

	return result, nil
}

// AnalyzePDFWithVision analyzes a PDF using multimodal capabilities
func (gc *GeminiClient) AnalyzePDFWithVision(ctx context.Context, pdfPath, prompt string) (string, error) {
	model := gc.client.GenerativeModel(gc.model)

	// Configure model parameters
	model.SetTemperature(float32(gc.temperature))
	model.SetMaxOutputTokens(int32(gc.maxTokens))

	// Read PDF file
	pdfData, err := os.ReadFile(pdfPath)
	if err != nil {
		return "", fmt.Errorf("failed to read PDF: %w", err)
	}

	// Create multimodal prompt
	resp, err := model.GenerateContent(ctx,
		genai.Text(prompt),
		genai.Blob{
			MIMEType: "application/pdf",
			Data:     pdfData,
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to analyze PDF: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates returned")
	}

	if resp.Candidates[0].Content == nil {
		return "", fmt.Errorf("no content in response")
	}

	// Extract text from response
	var result string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			result += string(txt)
		}
	}

	return result, nil
}

// GenerateWithRetry generates content with retry logic
func (gc *GeminiClient) GenerateWithRetry(ctx context.Context, prompt string, maxAttempts int, backoffMultiplier int, initialDelayMs int) (string, error) {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err := gc.GenerateText(ctx, prompt)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if attempt < maxAttempts {
			delay := time.Duration(initialDelayMs*(1<<uint(attempt-1))) * time.Millisecond
			time.Sleep(delay)
		}
	}

	return "", fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}

// ListAvailableModels lists all available Gemini models
func (gc *GeminiClient) ListAvailableModels(ctx context.Context) ([]string, error) {
	iter := gc.client.ListModels(ctx)
	var models []string

	for {
		model, err := iter.Next()
		if err != nil {
			break
		}
		models = append(models, model.Name)
	}

	return models, nil
}

// FindThinkingModel finds the best available thinking model
func (gc *GeminiClient) FindThinkingModel(ctx context.Context) (string, error) {
	models, err := gc.ListAvailableModels(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list models: %w", err)
	}

	// Priority list of thinking models to try (in order of preference)
	preferredModels := []string{
		"models/gemini-2.0-flash-thinking-exp",
		"models/gemini-2.0-flash-thinking-exp-1219",
		"models/gemini-exp-1206",
		"models/gemini-2.0-flash-exp",
		"models/gemini-1.5-pro-latest",
		"models/gemini-1.5-pro-002",
		"models/gemini-1.5-pro",
		"models/gemini-2.0-flash-latest",
		"models/gemini-2.0-flash",
		"models/gemini-1.5-flash-latest",
		"models/gemini-1.5-flash",
	}

	// Find first available preferred model
	for _, preferred := range preferredModels {
		for _, available := range models {
			if available == preferred {
				return preferred, nil
			}
		}
	}

	// If no preferred model found, return the first available model
	if len(models) > 0 {
		return models[0], nil
	}

	return "", fmt.Errorf("no models available")
}
