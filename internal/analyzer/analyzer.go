package analyzer

import (
	"archivist/internal/app"
	"context"
	"fmt"
	"log"
	"strings"
	"time"
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
	log.Println("     📝 Using simple analysis workflow (single API call)")
	log.Printf("     → Calling Gemini API (%s)...", a.config.Gemini.Model)
	startTime := time.Now()

	latexContent, err := a.client.AnalyzePDFWithVision(ctx, pdfPath, AnalysisPrompt)
	if err != nil {
		return "", fmt.Errorf("analysis failed: %w", err)
	}

	log.Printf("     ✓ Analysis complete (%.2fs, %d chars generated)", time.Since(startTime).Seconds(), len(latexContent))
	return cleanLatexOutput(latexContent), nil
}

// agenticAnalysis performs multi-stage analysis with self-reflection
func (a *Analyzer) agenticAnalysis(ctx context.Context, pdfPath string) (string, error) {
	var latexContent string
	var err error

	log.Println("     📊 Using agentic analysis workflow (multi-stage)")

	// Stage 1: Initial analysis with appropriate model
	log.Println("     🔬 Stage 1: Initial deep analysis")
	stage1Start := time.Now()
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

	log.Printf("     → Calling Gemini API (%s) for paper analysis...", stage1Config.Model)
	latexContent, err = stage1Client.AnalyzePDFWithVision(ctx, pdfPath, AnalysisPrompt)
	if err != nil {
		return "", fmt.Errorf("stage 1 analysis failed: %w", err)
	}

	latexContent = cleanLatexOutput(latexContent)
	log.Printf("     ✓ Stage 1 complete (%.2fs, %d chars generated)", time.Since(stage1Start).Seconds(), len(latexContent))

	// Stage 2: Self-reflection and refinement
	if a.config.Gemini.Agentic.SelfReflection {
		log.Printf("     🔄 Stage 2: Self-reflection (max %d iterations)", a.config.Gemini.Agentic.MaxIterations)
		for i := 0; i < a.config.Gemini.Agentic.MaxIterations; i++ {
			iterStart := time.Now()
			log.Printf("       → Iteration %d/%d: Reviewing for improvements...", i+1, a.config.Gemini.Agentic.MaxIterations)

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
				log.Printf("       ⚠️  Reflection iteration %d failed: %v (continuing with current version)", i+1, err)
				break
			}

			if strings.Contains(reflection, "APPROVED") {
				log.Printf("       ✓ Iteration %d: APPROVED (no changes needed) (%.2fs)", i+1, time.Since(iterStart).Seconds())
				break
			}

			// Use the improved version
			improvedLatex := cleanLatexOutput(reflection)
			if len(improvedLatex) > 100 { // Sanity check
				latexContent = improvedLatex
				log.Printf("       ✓ Iteration %d: Improved (%.2fs)", i+1, time.Since(iterStart).Seconds())
			} else {
				log.Printf("       ⚠️  Iteration %d: Invalid improvement, keeping previous version", i+1)
			}
		}
		log.Println("     ✓ Stage 2 complete")
	}

	// Stage 3: Validation removed to reduce API calls
	// LaTeX validation is now skipped to keep API usage to minimum

	return latexContent, nil
}

// validateLatex validates and fixes LaTeX syntax
func (a *Analyzer) validateLatex(ctx context.Context, latexContent string) (string, error) {
	log.Println("     → Calling Gemini API for LaTeX validation...")
	validationPrompt := fmt.Sprintf(ValidationPrompt, latexContent)

	result, err := a.client.GenerateText(ctx, validationPrompt)
	if err != nil {
		return latexContent, err
	}

	if strings.Contains(result, "VALID") {
		log.Println("     ✓ LaTeX syntax validated (no issues found)")
		return latexContent, nil
	}

	log.Println("     ✓ LaTeX syntax corrected")
	// Return the corrected version
	return cleanLatexOutput(result), nil
}

// cleanLatexOutput removes markdown code blocks and trims whitespace
func cleanLatexOutput(content string) string {
	// Remove markdown code blocks if present
	content = strings.ReplaceAll(content, "```latex", "")
	content = strings.ReplaceAll(content, "```tex", "")
	content = strings.ReplaceAll(content, "```", "")

	// Remove common feedback phrases and markdown headers
	lines := strings.Split(content, "\n")
	var filteredLines []string
	skipFeedback := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip lines that look like feedback or markdown headers
		if strings.HasPrefix(trimmed, "An excellent") ||
		   strings.HasPrefix(trimmed, "However,") ||
		   strings.HasPrefix(trimmed, "Here is") ||
		   strings.HasPrefix(trimmed, "###") ||
		   strings.HasPrefix(trimmed, "IMPROVED") ||
		   strings.HasPrefix(trimmed, "APPROVED") {
			skipFeedback = true
			continue
		}

		// Once we hit LaTeX content, stop skipping
		if strings.HasPrefix(trimmed, "\\documentclass") {
			skipFeedback = false
		}

		if !skipFeedback || strings.HasPrefix(trimmed, "\\") {
			filteredLines = append(filteredLines, line)
		}
	}

	content = strings.Join(filteredLines, "\n")

	// Find the start of the actual LaTeX document
	latexStart := strings.Index(content, "\\documentclass")
	if latexStart == -1 {
		// If no \documentclass found, try to find other LaTeX commands
		latexStart = strings.Index(content, "\\begin{document}")
		if latexStart == -1 {
			latexStart = strings.Index(content, "\\title{")
		}
	}

	// If we found LaTeX content, extract from that point
	if latexStart != -1 {
		content = content[latexStart:]
	}

	// Final pass: only keep lines that are part of LaTeX document
	lines = strings.Split(content, "\n")
	var cleanedLines []string
	inLatexContent := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines at the beginning
		if !inLatexContent && trimmed == "" {
			continue
		}

		// Start collecting LaTeX content from first LaTeX command
		if strings.HasPrefix(trimmed, "\\") || inLatexContent {
			inLatexContent = true
			cleanedLines = append(cleanedLines, line)
		}
	}

	content = strings.Join(cleanedLines, "\n")

	// Trim whitespace
	content = strings.TrimSpace(content)

	// Ensure document is properly closed
	content = ensureLatexComplete(content)

	return content
}

// ensureLatexComplete checks and fixes incomplete LaTeX documents
func ensureLatexComplete(content string) string {
	// Check if document ends with \end{document}
	if !strings.Contains(content, "\\end{document}") {
		content += "\n\\end{document}"
	}

	// Count and close any unclosed environments
	// Track common environments
	environments := []string{"itemize", "enumerate", "description", "keyinsight", "prerequisite"}

	for _, env := range environments {
		beginCount := strings.Count(content, "\\begin{"+env+"}")
		endCount := strings.Count(content, "\\end{"+env+"}")

		// Add missing \end{} tags before \end{document}
		if beginCount > endCount {
			missing := beginCount - endCount
			// Find position of \end{document}
			docEndPos := strings.LastIndex(content, "\\end{document}")
			if docEndPos != -1 {
				// Insert missing closing tags before \end{document}
				closingTags := ""
				for i := 0; i < missing; i++ {
					closingTags += "\\end{" + env + "}\n"
				}
				content = content[:docEndPos] + closingTags + "\n" + content[docEndPos:]
			}
		}
	}

	return content
}
