package internal

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"archivist/internal/analyzer"
	"archivist/internal/app"
	"archivist/internal/compiler"
	"archivist/internal/generator"
	"archivist/internal/parser"
	"archivist/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockGeminiClient for integration testing
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

// TestEndToEndWorkflow tests a complete processing workflow
func TestEndToEndWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Setup directories
	inputDir := filepath.Join(tmpDir, "input")
	texDir := filepath.Join(tmpDir, "tex")
	reportsDir := filepath.Join(tmpDir, "reports")
	metadataDir := filepath.Join(tmpDir, "metadata")
	
	for _, dir := range []string{inputDir, texDir, reportsDir, metadataDir} {
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)
	}

	// Create a test PDF file
	pdfPath := filepath.Join(inputDir, "test_paper.pdf")
	err := os.WriteFile(pdfPath, []byte("fake pdf content for testing"), 0644)
	require.NoError(t, err)

	// Setup configuration
	config := &app.Config{
		InputDir:        inputDir,
		TexOutputDir:    texDir,
		ReportOutputDir: reportsDir,
		MetadataDir:     metadataDir,
		Processing: app.ProcessingConfig{
			MaxWorkers:      1,
			BatchSize:       10,
			TimeoutPerPaper: 300,
		},
		Gemini: app.GeminiConfig{
			APIKey:      "test-key",
			Model:       "gemini-pro",
			Temperature: 0.7,
			MaxTokens:   2048,
			Agentic: app.AgenticConfig{
				Enabled: false, // Use simple analysis for this test
			},
		},
		Latex: app.LatexConfig{
			Compiler: "pdflatex",
			Engine:   "pdflatex",
			CleanAux: true,
		},
		HashAlgorithm: "md5",
	}

	// Create mock analyzer
	mockClient := new(MockGeminiClient)
	
	// Mock the metadata extraction
	metadataResponse := "TITLE: Test Research Paper\nAUTHORS: Author 1, Author 2\nYEAR: 2023\nABSTRACT: This is a test abstract."
	mockClient.On("AnalyzePDFWithVision", mock.Anything, pdfPath, mock.AnythingOfType("string")).Return(metadataResponse, nil)
	
	// Mock the full analysis
	latexOutput := `\\documentclass{article}
\\begin{document}
\\title{Test Research Paper}
\\section{Introduction}
This is a test introduction.
\\end{document}`
	mockClient.On("AnalyzePDFWithVision", mock.Anything, pdfPath, analyzer.AnalysisPrompt).Return(latexOutput, nil)

	analyzerObj := &analyzer.Analyzer{
		client: mockClient,
		config: config,
	}
	defer analyzerObj.Close()

	// Step 1: Parse metadata
	pdfParser := parser.NewPDFParser(mockClient)
	metadata, err := pdfParser.ExtractMetadata(context.Background(), pdfPath)
	require.NoError(t, err)
	assert.Equal(t, "Test Research Paper", metadata.Title)

	// Step 2: Analyze paper
	latexContent, err := analyzerObj.AnalyzePaper(context.Background(), pdfPath)
	require.NoError(t, err)
	assert.Contains(t, latexContent, "Test Research Paper")

	// Step 3: Generate LaTeX file
	latexGen := generator.NewLatexGenerator(texDir)
	texPath, err := latexGen.GenerateLatexFile(metadata.Title, latexContent)
	require.NoError(t, err)
	
	// Check that the file was created
	_, err = os.Stat(texPath)
	assert.NoError(t, err)

	// Step 4: Try to compile to PDF (this will likely fail without LaTeX installed)
	// For this test, we'll just validate that the compiler can be created
	latexCompiler := compiler.NewLatexCompiler(
		config.Latex.Compiler,
		config.Latex.Engine == "latexmk",
		config.Latex.CleanAux,
		reportsDir,
	)
	
	// Step 5: Store metadata
	metadataStore, err := storage.NewMetadataStore(metadataDir)
	require.NoError(t, err)

	record := storage.ProcessingRecord{
		FilePath:    pdfPath,
		FileHash:    "test-hash", // In real usage, this would come from fileutil.ComputeFileHash
		PaperTitle:  metadata.Title,
		ProcessedAt: time.Now(),
		TexFilePath: texPath,
		ReportPath:  filepath.Join(reportsDir, "test_research_paper.pdf"),
		Status:      storage.StatusCompleted,
	}
	err = metadataStore.MarkCompleted(record)
	require.NoError(t, err)

	// Verify the record was stored
	storedRecord, exists := metadataStore.GetRecord("test-hash")
	assert.True(t, exists)
	assert.Equal(t, metadata.Title, storedRecord.PaperTitle)
	assert.Equal(t, storage.StatusCompleted, storedRecord.Status)

	mockClient.AssertExpectations(t)
}

// TestDeduplicationWorkflow tests the deduplication functionality
func TestDeduplicationWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Setup directories
	metadataDir := filepath.Join(tmpDir, "metadata")
	err := os.MkdirAll(metadataDir, 0755)
	require.NoError(t, err)

	// Create metadata store
	metadataStore, err := storage.NewMetadataStore(metadataDir)
	require.NoError(t, err)

	// Add a completed record
	hash := "test-file-hash"
	firstRecord := storage.ProcessingRecord{
		FilePath:    "/path/to/paper.pdf",
		FileHash:    hash,
		PaperTitle:  "Test Paper",
		ProcessedAt: time.Now(),
		TexFilePath: "/path/to/output.tex",
		ReportPath:  "/path/to/output.pdf",
		Status:      storage.StatusCompleted,
	}
	err = metadataStore.MarkCompleted(firstRecord)
	require.NoError(t, err)

	// Check that the file is marked as processed
	assert.True(t, metadataStore.IsProcessed(hash))

	// Add the same file with failed status
	failedRecord := storage.ProcessingRecord{
		FilePath:    "/path/to/paper.pdf",
		FileHash:    hash,
		PaperTitle:  "Test Paper",
		ProcessedAt: time.Now(),
		Status:      storage.StatusFailed,
		Error:       "API error",
	}
	err = metadataStore.MarkFailed(hash, "API error")
	require.NoError(t, err)

	// The file should still not be considered processed since last status is failed
	assert.False(t, metadataStore.IsProcessed(hash))
	
	// Mark it as completed again
	err = metadataStore.MarkCompleted(firstRecord)
	require.NoError(t, err)
	
	// Now it should be processed again
	assert.True(t, metadataStore.IsProcessed(hash))
}

// TestFileDiscoveryIntegration tests file discovery functionality
func TestFileDiscoveryIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create a directory structure with PDFs
	inputDir := filepath.Join(tmpDir, "input")
	err := os.MkdirAll(inputDir, 0755)
	require.NoError(t, err)
	
	// Create various files including PDFs
	files := []struct {
		name string
		isPDF bool
	}{
		{"paper1.pdf", true},
		{"paper2.PDF", true},  // Test case sensitivity
		{"document.txt", false},
		{"paper3.pdf", true},
		{"image.png", false},
		{"report.PDF", true},  // Test uppercase extension
	}
	
	for _, file := range files {
		path := filepath.Join(inputDir, file.name)
		content := "content"
		if file.isPDF {
			content = "%PDF-1.4 content"
		}
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create subdirectory with more PDFs
	subDir := filepath.Join(inputDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)
	
	subPdfPath := filepath.Join(subDir, "nested_paper.pdf")
	err = os.WriteFile(subPdfPath, []byte("%PDF-1.4 content"), 0644)
	require.NoError(t, err)

	// Simulate file discovery (we'll use fileutil function in real usage)
	// For this test, we'll just verify that we can find the PDF files
	allFiles := []string{}
	err = filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".pdf" {
			allFiles = append(allFiles, path)
		}
		return nil
	})
	require.NoError(t, err)
	
	// Should find 5 PDFs: 3 in root + 1 in subdir + 1 with uppercase extension
	assert.Equal(t, 5, len(allFiles))
	
	// Verify specific files are found
	pdfPaths := make(map[string]bool)
	for _, path := range allFiles {
		pdfPaths[filepath.Base(path)] = true
	}
	
	assert.True(t, pdfPaths["paper1.pdf"])
	assert.True(t, pdfPaths["paper2.PDF"])
	assert.True(t, pdfPaths["paper3.pdf"])
	assert.True(t, pdfPaths["report.PDF"])
	assert.True(t, pdfPaths["nested_paper.pdf"])
}

// TestErrorHandlingWorkflow tests error handling in the workflow
func TestErrorHandlingWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Setup directories
	metadataDir := filepath.Join(tmpDir, "metadata")
	err := os.MkdirAll(metadataDir, 0755)
	require.NoError(t, err)

	// Create metadata store
	metadataStore, err := storage.NewMetadataStore(metadataDir)
	require.NoError(t, err)

	// Add a failed record
	hash := "failed-hash"
	errMsg := "API timeout error"
	err = metadataStore.MarkFailed(hash, errMsg)
	require.NoError(t, err)

	// Verify the failed record exists
	record, exists := metadataStore.GetRecord(hash)
	assert.True(t, exists)
	assert.Equal(t, storage.StatusFailed, record.Status)
	assert.Equal(t, errMsg, record.Error)
	
	// Should not be considered processed
	assert.False(t, metadataStore.IsProcessed(hash))
}

// TestSpecialCharacterHandling tests handling of special characters in titles
func TestSpecialCharacterHandling(t *testing.T) {
	tmpDir := t.TempDir()
	texDir := filepath.Join(tmpDir, "tex")
	err := os.MkdirAll(texDir, 0755)
	require.NoError(t, err)

	// Test various special titles
	testTitles := []struct {
		rawTitle     string
		expectedFile string
	}{
		{
			rawTitle:     "Attention Is All You Need",
			expectedFile: "Attention_Is_All_You_Need.tex",
		},
		{
			rawTitle:     "Learning Deep Features: A Study with α, β, γ",
			expectedFile: "Learning_Deep_Features_ A_Study_with_____.tex", // Greek letters become underscores
		},
		{
			rawTitle:     "CNN & RNN: A Comparative Study",
			expectedFile: "CNN__RNN_ A_Comparative_Study.tex",
		},
		{
			rawTitle:     "Transformers: What's Next?",
			expectedFile: "Transformers_ What_s_Next_.tex",
		},
		{
			rawTitle:     "My Paper/v1", // Contains forward slash
			expectedFile: "My_Paper_v1.tex",
		},
	}

	for _, test := range testTitles {
		t.Run(test.rawTitle, func(t *testing.T) {
			latexGen := generator.NewLatexGenerator(texDir)
			latexContent := "\\documentclass{article}\n\\begin{document}\nTest\n\\end{document}"
			
			resultPath, err := latexGen.GenerateLatexFile(test.rawTitle, latexContent)
			require.NoError(t, err)
			
			expectedPath := filepath.Join(texDir, test.expectedFile)
			assert.Equal(t, expectedPath, resultPath)
			
			// Check that file was created
			_, err = os.Stat(resultPath)
			assert.NoError(t, err)
		})
	}
}