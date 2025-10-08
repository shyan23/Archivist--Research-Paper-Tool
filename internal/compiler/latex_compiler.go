package compiler

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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
	log.Printf("     → Running latexmk (automatic multi-pass)...")
	startTime := time.Now()

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

	log.Printf("     ✓ latexmk complete (%.2fs)", time.Since(startTime).Seconds())
	return nil
}

// compileManual performs manual compilation with multiple passes
func (lc *LatexCompiler) compileManual(workDir, texFile string) error {
	// Usually need 2-3 passes for references and TOC
	log.Printf("     → Running %s (3 passes for references/TOC)...", lc.engine)
	for i := 0; i < 3; i++ {
		passStart := time.Now()
		log.Printf("       Pass %d/3...", i+1)

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

		log.Printf("       ✓ Pass %d complete (%.2fs)", i+1, time.Since(passStart).Seconds())
	}

	return nil
}

// cleanAuxiliaryFiles removes auxiliary LaTeX files
func (lc *LatexCompiler) cleanAuxiliaryFiles(workDir, baseName string) {
	log.Printf("     🧹 Cleaning auxiliary files...")
	extensions := []string{".aux", ".log", ".out", ".toc", ".fdb_latexmk", ".fls", ".synctex.gz"}

	cleaned := 0
	for _, ext := range extensions {
		auxFile := filepath.Join(workDir, baseName+ext)
		if err := os.Remove(auxFile); err == nil {
			cleaned++
		}
	}

	// Also clean latexmk files
	if lc.useLatexmk {
		cmd := exec.Command("latexmk", "-c", baseName+".tex")
		cmd.Dir = workDir
		cmd.Run() // Ignore errors
	}

	log.Printf("     ✓ Cleaned %d auxiliary files", cleaned)
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
