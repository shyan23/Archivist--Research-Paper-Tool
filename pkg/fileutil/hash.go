package fileutil

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ComputeFileHash computes SHA-256 hash of a file
func ComputeFileHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// SanitizeFilename removes invalid characters from filename
func SanitizeFilename(name string) string {
	// Remove file extension if present
	name = strings.TrimSuffix(name, filepath.Ext(name))

	// Replace invalid characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)

	sanitized := replacer.Replace(name)

	// Limit length
	if len(sanitized) > 200 {
		sanitized = sanitized[:200]
	}

	return sanitized
}

// GetPDFFiles returns all PDF files in a directory
func GetPDFFiles(dir string) ([]string, error) {
	var pdfFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".pdf" {
			pdfFiles = append(pdfFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return pdfFiles, nil
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
