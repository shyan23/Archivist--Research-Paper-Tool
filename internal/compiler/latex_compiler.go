package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type LatexCompiler struct {
	engine     string // "pdflatex", "xelatex", "lualatex"
	useLatexmk bool
	cleanAux   bool
	outputDir  string
}

// NewLatexCompiler creates a new LaTeX compiler
func NewLatexCompiler(engine string, useLatexmk, cleanAux bool, outputDir string) *LatexCompiler {
	return &LatexCompiler{
		engine:     engine,
		useLatexmk: useLatexmk,
		cleanAux:   cleanAux,
		outputDir:  outputDir,
	}
}

// Compile compiles a .tex file to PDF
func (lc *LatexCompiler) Compile(texPath string) (string, error) {
	workDir := filepath.Dir(texPath)
	texFile := filepath.Base(texPath)
	baseName := strings.TrimSuffix(texFile, ".tex")

	// Output PDF path
	outputPDF := filepath.Join(lc.outputDir, baseName+".pdf")

	// Ensure output directory exists
	if err := os.MkdirAll(lc.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	var err error
	if lc.useLatexmk {
		err = lc.compileWithLatexmk(workDir, texFile)
	} else {
		err = lc.compileManual(workDir, texFile)
	}

	if err != nil {
		return "", err
	}

	// Move compiled PDF to output directory
	compiledPDF := filepath.Join(workDir, baseName+".pdf")
	if err := os.Rename(compiledPDF, outputPDF); err != nil {
		return "", fmt.Errorf("failed to move PDF to output directory: %w", err)
	}

	// Clean auxiliary files
	if lc.cleanAux {
		lc.cleanAuxiliaryFiles(workDir, baseName)
	}

	return outputPDF, nil
}

// compileWithLatexmk compiles using latexmk
func (lc *LatexCompiler) compileWithLatexmk(workDir, texFile string) error {
	cmd := exec.Command("latexmk",
		"-pdf",
		"-interaction=nonstopmode",
		"-halt-on-error",
		texFile,
	)
	cmd.Dir = workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("latexmk compilation failed: %w\nOutput: %s", err, output)
	}

	return nil
}

// compileManual performs manual compilation with multiple passes
func (lc *LatexCompiler) compileManual(workDir, texFile string) error {
	// Usually need 2-3 passes for references and TOC
	for i := 0; i < 3; i++ {
		cmd := exec.Command(lc.engine,
			"-interaction=nonstopmode",
			"-halt-on-error",
			texFile,
		)
		cmd.Dir = workDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("compilation pass %d failed: %w\nOutput: %s", i+1, err, output)
		}
	}

	return nil
}

// cleanAuxiliaryFiles removes auxiliary LaTeX files
func (lc *LatexCompiler) cleanAuxiliaryFiles(workDir, baseName string) {
	extensions := []string{".aux", ".log", ".out", ".toc", ".fdb_latexmk", ".fls", ".synctex.gz"}

	for _, ext := range extensions {
		auxFile := filepath.Join(workDir, baseName+ext)
		os.Remove(auxFile) // Ignore errors
	}

	// Also clean latexmk files
	if lc.useLatexmk {
		cmd := exec.Command("latexmk", "-c", baseName+".tex")
		cmd.Dir = workDir
		cmd.Run() // Ignore errors
	}
}

// CheckDependencies verifies that LaTeX is installed
func CheckDependencies(useLatexmk bool, engine string) error {
	if useLatexmk {
		if err := checkCommand("latexmk"); err != nil {
			return fmt.Errorf("latexmk not found: %w", err)
		}
	}

	if err := checkCommand(engine); err != nil {
		return fmt.Errorf("%s not found: %w", engine, err)
	}

	return nil
}

func checkCommand(cmd string) error {
	_, err := exec.LookPath(cmd)
	return err
}
