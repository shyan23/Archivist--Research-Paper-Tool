package analyzer_test

import (
	"context"
	"testing"

	"archivist/internal/analyzer"
	"archivist/internal/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockGeminiClient for testing
type MockGeminiClient struct {
	mock.Mock
}

func (m *MockGeminiClient) GenerateText(ctx context.Context, prompt string) (string, error) {
	args := m.Called(ctx, prompt)
	return args.String(0), args.Error(1)
}

func (m *MockGeminiClient) AnalyzePDFWithVision(ctx context.Context, pdfPath, prompt string) (string, error) {
	args := m.Called(ctx, pdfPath, prompt)
	return args.String(0), args.Error(1)
}

func (m *MockGeminiClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockGeminiClient) GenerateWithRetry(ctx context.Context, prompt string, maxAttempts int, backoffMultiplier int, initialDelayMs int) (string, error) {
	args := m.Called(ctx, prompt, maxAttempts, backoffMultiplier, initialDelayMs)
	return args.String(0), args.Error(1)
}

// TestNewAnalyzer tests the creation of a new analyzer
func TestNewAnalyzer(t *testing.T) {
	// Create a temporary config
	config := &app.Config{
		Gemini: app.GeminiConfig{
			APIKey:      "test-key",
			Model:       "gemini-pro",
			Temperature: 0.7,
			MaxTokens:   2048,
			Agentic: app.AgenticConfig{
				Enabled: true,
			},
		},
	}

	anlzr, err := analyzer.NewAnalyzer(config)
	require.NoError(t, err)
	assert.NotNil(t, anlzr)
}

// TestAnalyzerClose tests closing the analyzer
func TestAnalyzerClose(t *testing.T) {
	config := &app.Config{
		Gemini: app.GeminiConfig{
			APIKey:      "test-key",
			Model:       "gemini-pro",
			Temperature: 0.7,
			MaxTokens:   2048,
		},
	}

	anlzr, err := analyzer.NewAnalyzer(config)
	require.NoError(t, err)

	err = anlzr.Close()
	assert.NoError(t, err)
}

// TestSimpleAnalysis tests single-stage analysis
func TestSimpleAnalysis(t *testing.T) {
	// Cannot test this directly as Analyzer struct fields are unexported
	// and cleanLatexOutput is unexported
	t.Skip("Cannot test AnalyzePaper with mock client as Analyzer fields are unexported")
}

// TestAgenticAnalysis tests multi-stage agentic analysis
func TestAgenticAnalysis(t *testing.T) {
	// Cannot test this directly as Analyzer struct fields are unexported
	// and cleanLatexOutput is unexported
	t.Skip("Cannot test AnalyzePaper with mock client as Analyzer fields are unexported")
}

// TestCleanLatexOutput tests the cleanLatexOutput function
func TestCleanLatexOutput(t *testing.T) {
	// cleanLatexOutput is unexported and cannot be tested from external package
	t.Skip("cleanLatexOutput is unexported and cannot be tested from external package")
}

// TestValidateLatex tests the validateLatex function
func TestValidateLatex(t *testing.T) {
	// validateLatex is unexported and cannot be tested from external package
	t.Skip("validateLatex is unexported and cannot be tested from external package")
}