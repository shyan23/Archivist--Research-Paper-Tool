package analyzer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

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

	analyzer, err := NewAnalyzer(config)
	require.NoError(t, err)
	assert.NotNil(t, analyzer)
	assert.NotNil(t, analyzer.client)
	assert.Equal(t, config, analyzer.config)
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

	analyzer, err := NewAnalyzer(config)
	require.NoError(t, err)

	err = analyzer.Close()
	assert.NoError(t, err)
}

// TestSimpleAnalysis tests single-stage analysis
func TestSimpleAnalysis(t *testing.T) {
	// Create a mock client
	mockClient := new(MockGeminiClient)
	
	// Create a temporary PDF file for testing
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "test.pdf")
	err := os.WriteFile(pdfPath, []byte("fake pdf content"), 0644)
	require.NoError(t, err)

	// Mock the AnalyzePDFWithVision call
	expectedOutput := "test LaTeX output"
	mockClient.On("AnalyzePDFWithVision", mock.Anything, pdfPath, AnalysisPrompt).Return(expectedOutput, nil)

	// Create analyzer with agentic disabled
	config := &app.Config{
		Gemini: app.GeminiConfig{
			APIKey:      "test-key",
			Model:       "gemini-pro",
			Temperature: 0.7,
			MaxTokens:   2048,
			Agentic: app.AgenticConfig{
				Enabled: false, // Simple analysis
			},
		},
	}

	analyzer := &Analyzer{
		client: mockClient,
		config: config,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := analyzer.AnalyzePaper(ctx, pdfPath)
	assert.NoError(t, err)
	assert.Equal(t, cleanLatexOutput(expectedOutput), result)

	mockClient.AssertExpectations(t)
}

// TestAgenticAnalysis tests multi-stage agentic analysis
func TestAgenticAnalysis(t *testing.T) {
	// Create a mock client
	mockClient := new(MockGeminiClient)
	
	// Create a temporary PDF file for testing
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "test.pdf")
	err := os.WriteFile(pdfPath, []byte("fake pdf content"), 0644)
	require.NoError(t, err)

	// Mock the initial analysis call
	analysisOutput := "test LaTeX output"
	mockClient.On("AnalyzePDFWithVision", mock.Anything, pdfPath, AnalysisPrompt).Return(analysisOutput, nil)

	// Mock the validation call
	validationOutput := "VALID"
	validationPrompt := "Review this LaTeX code for syntax errors. Check:\n1. All environments are properly opened and closed (\\begin{} and \\end{} match)\n2. All special characters are properly escaped\n3. All equations are properly formatted\n4. No markdown syntax mixed in\n\nLaTeX code to validate:\n" + analysisOutput + "\n\nIf there are errors, output ONLY the corrected LaTeX code. If it's valid, output: VALID"
	mockClient.On("GenerateText", mock.Anything, validationPrompt).Return(validationOutput, nil)

	// Create analyzer with agentic enabled
	config := &app.Config{
		Gemini: app.GeminiConfig{
			APIKey:      "test-key",
			Model:       "gemini-pro",
			Temperature: 0.7,
			MaxTokens:   2048,
			Agentic: app.AgenticConfig{
				Enabled:        true,
				SelfReflection: false, // Disable self-reflection for this test
				Stages: app.StagesConfig{
					LatexGeneration: app.StageConfig{
						Validation: true,
					},
				},
			},
		},
	}

	analyzer := &Analyzer{
		client: mockClient,
		config: config,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := analyzer.AnalyzePaper(ctx, pdfPath)
	assert.NoError(t, err)
	assert.Equal(t, cleanLatexOutput(analysisOutput), result)

	mockClient.AssertExpectations(t)
}

// TestCleanLatexOutput tests the cleanLatexOutput function
func TestCleanLatexOutput(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Remove markdown code blocks",
			input:    "```latex\n\\documentclass{article}\n\\begin{document}\nTest\n\\end{document}\n```",
			expected: "\\documentclass{article}\n\\begin{document}\nTest\n\\end{document}",
		},
		{
			name:     "Remove tex code blocks",
			input:    "```tex\n\\title{Test}\n```",
			expected: "\\title{Test}",
		},
		{
			name:     "Remove generic code blocks",
			input:    "```\n\\section{Test}\n```",
			expected: "\\section{Test}",
		},
		{
			name:     "Trim whitespace",
			input:    "  \\title{Test}  \n\n",
			expected: "\\title{Test}",
		},
		{
			name:     "No changes needed",
			input:    "\\title{Test}",
			expected: "\\title{Test}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cleanLatexOutput(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestValidateLatex tests the validateLatex function
func TestValidateLatex(t *testing.T) {
	// Create a mock client
	mockClient := new(MockGeminiClient)

	config := &app.Config{
		Gemini: app.GeminiConfig{
			APIKey:      "test-key",
			Model:       "gemini-pro",
			Temperature: 0.7,
			MaxTokens:   2048,
		},
	}

	analyzer := &Analyzer{
		client: mockClient,
		config: config,
	}

	latexContent := "\\documentclass{article}\n\\begin{document}\nTest\n\\end{document}"
	expectedValidationPrompt := "Review this LaTeX code for syntax errors. Check:\n1. All environments are properly opened and closed (\\begin{} and \\end{} match)\n2. All special characters are properly escaped\n3. All equations are properly formatted\n4. No markdown syntax mixed in\n\nLaTeX code to validate:\n" + latexContent + "\n\nIf there are errors, output ONLY the corrected LaTeX code. If it's valid, output: VALID"

	// Test with valid LaTeX
	mockClient.On("GenerateText", mock.Anything, expectedValidationPrompt).Return("VALID", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := analyzer.validateLatex(ctx, latexContent)
	assert.NoError(t, err)
	assert.Equal(t, latexContent, result)
	mockClient.AssertExpectations(t)

	// Test with invalid LaTeX that gets corrected
	mockClient.On("GenerateText", mock.Anything, expectedValidationPrompt).Return("corrected LaTeX content", nil)

	result, err = analyzer.validateLatex(ctx, latexContent)
	assert.NoError(t, err)
	assert.Equal(t, "corrected LaTeX content", result)
}