package tui

import (
	"archivist/internal/app"
	"archivist/internal/rag"
	"archivist/internal/worker"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// indexPaperIfNeeded checks if a paper is indexed, and indexes it if not
func indexPaperIfNeeded(ctx context.Context, config *app.Config, paperTitle string) error {
	// Check if already indexed
	indexDir := filepath.Join(".metadata", "vector_index")
	vectorStore, err := rag.NewFAISSVectorStore(indexDir)
	if err != nil {
		return fmt.Errorf("failed to load vector store: %w", err)
	}

	indexedPapers := vectorStore.GetIndexedPapers()
	for _, indexed := range indexedPapers {
		if indexed == paperTitle {
			log.Printf("✅ Paper already indexed: %s", paperTitle)
			return nil // Already indexed
		}
	}

	// Not indexed - need to index it
	log.Printf("📇 Paper not indexed, indexing now: %s", paperTitle)

	// Find the LaTeX file for this paper
	texFile := findTexFileForPaper(config.TexOutputDir, paperTitle)
	if texFile == "" {
		return fmt.Errorf("could not find LaTeX file for paper: %s", paperTitle)
	}

	// Read LaTeX content
	latexContent, err := os.ReadFile(texFile)
	if err != nil {
		return fmt.Errorf("failed to read LaTeX file: %w", err)
	}

	// Find the original PDF
	pdfFile := findPDFFileForPaper(config.InputDir, paperTitle)

	// Index the paper
	log.Printf("  📇 Indexing from: %s", texFile)
	if err := worker.IndexPaperAfterProcessing(ctx, config, paperTitle, string(latexContent), pdfFile); err != nil {
		return fmt.Errorf("indexing failed: %w", err)
	}

	log.Printf("✅ Paper indexed successfully: %s", paperTitle)
	return nil
}

// findTexFileForPaper finds the LaTeX file for a given paper title
func findTexFileForPaper(texDir, paperTitle string) string {
	// Convert paper title to potential filename
	// "Paper Title" -> "Paper_Title.tex" or "Paper_Title_Student_Guide.tex"
	baseFilename := strings.ReplaceAll(paperTitle, " ", "_")

	// Try different variations
	variations := []string{
		filepath.Join(texDir, baseFilename+".tex"),
		filepath.Join(texDir, baseFilename+"_Student_Guide.tex"),
	}

	for _, path := range variations {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Try finding any .tex file that contains similar words
	files, err := filepath.Glob(filepath.Join(texDir, "*.tex"))
	if err != nil {
		return ""
	}

	// Match by similarity
	titleWords := strings.Fields(strings.ToLower(paperTitle))
	for _, file := range files {
		basename := filepath.Base(file)
		basename = strings.ToLower(strings.TrimSuffix(basename, ".tex"))
		basename = strings.ReplaceAll(basename, "_", " ")

		// Count matching words
		matchCount := 0
		for _, word := range titleWords {
			if len(word) > 3 && strings.Contains(basename, word) {
				matchCount++
			}
		}

		// If more than half the words match, consider it a match
		if matchCount > len(titleWords)/2 {
			return file
		}
	}

	return ""
}

// findPDFFileForPaper finds the original PDF file for a given paper title
func findPDFFileForPaper(inputDir, paperTitle string) string {
	// Convert paper title to potential filename
	baseFilename := strings.ReplaceAll(paperTitle, " ", "_")

	// Try different variations
	variations := []string{
		filepath.Join(inputDir, baseFilename+".pdf"),
	}

	for _, path := range variations {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Try finding any .pdf file that contains similar words
	files, err := filepath.Glob(filepath.Join(inputDir, "*.pdf"))
	if err != nil {
		return ""
	}

	// Match by similarity
	titleWords := strings.Fields(strings.ToLower(paperTitle))
	for _, file := range files {
		basename := filepath.Base(file)
		basename = strings.ToLower(strings.TrimSuffix(basename, ".pdf"))
		basename = strings.ReplaceAll(basename, "_", " ")

		// Count matching words
		matchCount := 0
		for _, word := range titleWords {
			if len(word) > 3 && strings.Contains(basename, word) {
				matchCount++
			}
		}

		// If more than half the words match, consider it a match
		if matchCount > len(titleWords)/2 {
			return file
		}
	}

	return ""
}

// indexMultiplePapersIfNeeded indexes multiple papers if needed
func indexMultiplePapersIfNeeded(ctx context.Context, config *app.Config, paperTitles []string) error {
	for _, paperTitle := range paperTitles {
		indexCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		err := indexPaperIfNeeded(indexCtx, config, paperTitle)
		cancel()

		if err != nil {
			log.Printf("⚠️  Failed to index %s: %v", paperTitle, err)
			// Continue with other papers even if one fails
		}
	}
	return nil
}
