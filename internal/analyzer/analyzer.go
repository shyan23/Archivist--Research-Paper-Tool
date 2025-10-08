package analyzer

import (
	"archivist/internal/app"
	"context"
	"fmt"
	"strings"
)

type Analyzer struct {
	client *GeminiClient
	config *app.Config
}

// NewAnalyzer creates a new analyzer
func NewAnalyzer(config *app.Config) (*Analyzer, error) {
	client, err := NewGeminiClient(
		config.Gemini.APIKey,
		config.Gemini.Model,
		config.Gemini.Temperature,
		config.Gemini.MaxTokens,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &Analyzer{
		client: client,
		config: config,
	}, nil
}

// Close closes the analyzer
func (a *Analyzer) Close() error {
	return a.client.Close()
}

// GetClient returns the underlying Gemini client
func (a *Analyzer) GetClient() *GeminiClient {
	return a.client
}

// AnalyzePaper performs multi-stage agentic analysis of a research paper
func (a *Analyzer) AnalyzePaper(ctx context.Context, pdfPath string) (string, error) {
	if !a.config.Gemini.Agentic.Enabled {
		// Simple single-stage analysis
		return a.simplAnalysis(ctx, pdfPath)
	}

	// Multi-stage agentic workflow
	return a.agenticAnalysis(ctx, pdfPath)
}

// simplAnalysis performs a single-stage analysis
func (a *Analyzer) simplAnalysis(ctx context.Context, pdfPath string) (string, error) {
	latexContent, err := a.client.AnalyzePDFWithVision(ctx, pdfPath, AnalysisPrompt)
	if err != nil {
		return "", fmt.Errorf("analysis failed: %w", err)
	}

	return cleanLatexOutput(latexContent), nil
}

// agenticAnalysis performs multi-stage analysis with self-reflection
func (a *Analyzer) agenticAnalysis(ctx context.Context, pdfPath string) (string, error) {
	var latexContent string
	var err error

	// Stage 1: Initial analysis with appropriate model
	stage1Config := a.config.Gemini.Agentic.Stages.MethodologyAnalysis
	stage1Client, err := NewGeminiClient(
		a.config.Gemini.APIKey,
		stage1Config.Model,
		stage1Config.Temperature,
		a.config.Gemini.MaxTokens,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create stage 1 client: %w", err)
	}
	defer stage1Client.Close()

	latexContent, err = stage1Client.AnalyzePDFWithVision(ctx, pdfPath, AnalysisPrompt)
	if err != nil {
		return "", fmt.Errorf("stage 1 analysis failed: %w", err)
	}

	latexContent = cleanLatexOutput(latexContent)

	// Stage 2: Self-reflection and refinement
	if a.config.Gemini.Agentic.SelfReflection {
		for i := 0; i < a.config.Gemini.Agentic.MaxIterations; i++ {
			reflectionPrompt := fmt.Sprintf(`Review this LaTeX document for a research paper analysis. Check for:
1. Clarity and student-friendliness
2. Technical accuracy
3. Completeness of explanations
4. Proper LaTeX syntax

Current document:
%s

If improvements are needed, output the IMPROVED LaTeX document (complete, not just changes).
If it's already excellent, output: APPROVED

Output:`, latexContent)

			reflection, err := a.client.GenerateText(ctx, reflectionPrompt)
			if err != nil {
				// If reflection fails, continue with current version
				break
			}

			if strings.Contains(reflection, "APPROVED") {
				break
			}

			// Use the improved version
			improvedLatex := cleanLatexOutput(reflection)
			if len(improvedLatex) > 100 { // Sanity check
				latexContent = improvedLatex
			}
		}
	}

	// Stage 3: Validation
	if a.config.Gemini.Agentic.Stages.LatexGeneration.Validation {
		validated, err := a.validateLatex(ctx, latexContent)
		if err == nil && len(validated) > 100 {
			latexContent = validated
		}
	}

	return latexContent, nil
}

// validateLatex validates and fixes LaTeX syntax
func (a *Analyzer) validateLatex(ctx context.Context, latexContent string) (string, error) {
	validationPrompt := fmt.Sprintf(ValidationPrompt, latexContent)

	result, err := a.client.GenerateText(ctx, validationPrompt)
	if err != nil {
		return latexContent, err
	}

	if strings.Contains(result, "VALID") {
		return latexContent, nil
	}

	// Return the corrected version
	return cleanLatexOutput(result), nil
}

// cleanLatexOutput removes markdown code blocks and trims whitespace
func cleanLatexOutput(content string) string {
	// Remove markdown code blocks if present
	content = strings.ReplaceAll(content, "```latex", "")
	content = strings.ReplaceAll(content, "```tex", "")
	content = strings.ReplaceAll(content, "```", "")

	// Trim whitespace
	content = strings.TrimSpace(content)

	return content
}
