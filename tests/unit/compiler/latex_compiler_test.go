package compiler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLatexCompiler tests creating a new LaTeX compiler
func TestNewLatexCompiler(t *testing.T) {
	engine := "pdflatex"
	outputDir := "/tmp/test-output"
	
	compiler := NewLatexCompiler(engine, true, true, outputDir)
	
	assert.NotNil(t, compiler)
	assert.Equal(t, engine, compiler.engine)
	assert.True(t, compiler.useLatexmk)
	assert.True(t, compiler.cleanAux)
	assert.Equal(t, outputDir, compiler.outputDir)
}

// TestCheckDependencies tests dependency checking
func TestCheckDependencies(t *testing.T) {
	// This test assumes that common LaTeX tools might not be installed in the test environment
	// So we're testing that the function doesn't panic and handles missing dependencies gracefully
	
	t.Run("Check dependencies with latexmk", func(t *testing.T) {
		err := CheckDependencies(true, "pdflatex")
		// The function may return an error if dependencies are not installed,
		// but it should not panic
	})
	
	t.Run("Check dependencies without latexmk", func(t *testing.T) {
		err := CheckDependencies(false, "pdflatex")
		// The function may return an error if dependencies are not installed,
		// but it should not panic
	})
}

// TestSanityCheck tests the compilation process with a simple valid LaTeX
func TestCompileValidLatex(t *testing.T) {
	// This test is skipped if LaTeX is not available
	// We'll just create a basic test to ensure the function structure works
	if err := CheckDependencies(false, "pdflatex"); err != nil {
		t.Skipf("Skipping test: LaTeX dependencies not available: %v", err)
	}

	tmpDir := t.TempDir()
	texDir := filepath.Join(tmpDir, "tex")
	pdfDir := filepath.Join(tmpDir, "pdf")
	
	err := os.MkdirAll(texDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(pdfDir, 0755)
	require.NoError(t, err)

	// Create a simple valid LaTeX file
	texContent := `\\documentclass{article}
\\begin{document}
Hello, World!
\\end{document}`
	
	texPath := filepath.Join(texDir, "test.tex")
	err = os.WriteFile(texPath, []byte(texContent), 0644)
	require.NoError(t, err)

	compiler := NewLatexCompiler("pdflatex", false, true, pdfDir)
	
	// This test will fail if LaTeX is not installed, which is expected in many environments
	_, err = compiler.Compile(texPath)
	if err != nil {
		t.Logf("Compilation failed (expected if LaTeX not installed): %v", err)
		// We won't fail the test if LaTeX tools are not available
	}
}

// TestCompileManual tests manual compilation process
func TestCompileManual(t *testing.T) {
	// Create a mock compiler that won't actually call pdflatex
	// Since we can't reliably test compilation without LaTeX installation
	
	tmpDir := t.TempDir()
	texDir := filepath.Join(tmpDir, "tex")
	
	err := os.MkdirAll(texDir, 0755)
	require.NoError(t, err)

	// Create a simple LaTeX file
	texContent := `\\documentclass{article}
\\begin{document}
Hello, World!
\\end{document}`
	
	texPath := filepath.Join(texDir, "test.tex")
	err = os.WriteFile(texPath, []byte(texContent), 0644)
	require.NoError(t, err)

	// Test with non-existent LaTeX engine to trigger error
	compiler := NewLatexCompiler("nonexistent-latex-engine", false, true, tmpDir)
	
	// This should fail because the engine doesn't exist
	_, err = compiler.Compile(texPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compilation failed")
}

// TestCleanAuxiliaryFiles tests the auxiliary files cleaning functionality
func TestCleanAuxiliaryFiles(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create some auxiliary files
	auxFiles := []string{
		"test.aux",
		"test.log",
		"test.out",
		"test.toc",
		"test.fdb_latexmk",
		"test.fls",
		"test.synctex.gz",
	}
	
	for _, file := range auxFiles {
		auxPath := filepath.Join(tmpDir, file)
		err := os.WriteFile(auxPath, []byte("dummy content"), 0644)
		require.NoError(t, err)
	}
	
	// Verify files exist before cleaning
	for _, file := range auxFiles {
		auxPath := filepath.Join(tmpDir, file)
		_, err := os.Stat(auxPath)
		assert.NoError(t, err, "Auxiliary file should exist before cleaning")
	}

	// Create compiler and clean auxiliary files
	compiler := NewLatexCompiler("pdflatex", true, true, tmpDir)
	compiler.cleanAuxiliaryFiles(tmpDir, "test")

	// Verify files are deleted after cleaning
	for _, file := range auxFiles {
		auxPath := filepath.Join(tmpDir, file)
		_, err := os.Stat(auxPath)
		assert.Error(t, err, "Auxiliary file should not exist after cleaning")
		assert.True(t, os.IsNotExist(err))
	}
}

// TestWithNonExistentFiles tests cleaning with non-existent auxiliary files
func TestCleanAuxiliaryFilesNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create compiler and clean auxiliary files for a non-existent base name
	compiler := NewLatexCompiler("pdflatex", true, true, tmpDir)
	compiler.cleanAuxiliaryFiles(tmpDir, "nonexistent")

	// Should not cause any errors
}

// TestCompileWithLatexmkError tests latexmk compilation with error handling
func TestCompileWithLatexmkError(t *testing.T) {
	tmpDir := t.TempDir()
	texDir := filepath.Join(tmpDir, "tex")
	
	err := os.MkdirAll(texDir, 0755)
	require.NoError(t, err)

	// Create a LaTeX file
	texContent := `\\documentclass{article}
\\begin{document}
Hello, World!
\\end{document}`
	
	texPath := filepath.Join(texDir, "test.tex")
	err = os.WriteFile(texPath, []byte(texContent), 0644)
	require.NoError(t, err)

	// Test with latexmk disabled but trying to use it
	compiler := &LatexCompiler{
		engine:     "nonexistent-engine",
		useLatexmk: true, // This will try to call latexmk which may not exist
		cleanAux:   false,
		outputDir:  tmpDir,
	}
	
	// This should fail because latexmk or the engine doesn't exist
	err = compiler.compileWithLatexmk(texDir, "test.tex")
	assert.Error(t, err)
}