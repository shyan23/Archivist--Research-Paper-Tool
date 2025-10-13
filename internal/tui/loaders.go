package tui

import (
	"archivist/internal/storage"
	"archivist/pkg/fileutil"
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
)

// loadLibraryPapers loads all papers from lib folder
func (m *Model) loadLibraryPapers() {
	files, err := fileutil.GetPDFFiles(m.config.InputDir)
	if err != nil {
		m.err = err
		return
	}

	items := make([]list.Item, len(files))
	for i, file := range files {
		basename := filepath.Base(file)

		// Check if processed
		hash, _ := fileutil.ComputeFileHash(file)
		status := "🔴 Unprocessed"
		if m.metadataStore.IsProcessed(hash) {
			status = "✅ Processed"
		}

		items[i] = item{
			title:       basename,
			description: fmt.Sprintf("%s • %s", status, file),
			action:      file,
		}
	}

	delegate := createStyledDelegate()
	m.libraryList = list.New(items, delegate, 0, 0)
	m.libraryList.Title = fmt.Sprintf("📚 Library Papers (%d total)", len(files))
	m.libraryList.SetShowStatusBar(false)
	m.libraryList.Styles.Title = titleStyle
	if m.width > 0 && m.height > 0 {
		m.libraryList.SetSize(m.width-4, m.height-8)
	}
}

// loadProcessedPapers loads processed papers (excludes failed ones)
func (m *Model) loadProcessedPapers() {
	records := m.metadataStore.GetAllRecords()

	// Filter out failed papers
	items := make([]list.Item, 0)
	for _, record := range records {
		// Skip failed papers - don't show them in TUI
		if record.Status == storage.StatusFailed {
			continue
		}

		statusIcon := "✅"
		if record.Status == storage.StatusProcessing {
			statusIcon = "⏳"
		}

		items = append(items, item{
			title:       record.PaperTitle,
			description: fmt.Sprintf("%s %s • Processed: %s", statusIcon, record.Status, record.ProcessedAt.Format("2006-01-02 15:04")),
			action:      record.FilePath,
		})
	}

	delegate := createStyledDelegate()
	m.processedList = list.New(items, delegate, 0, 0)
	m.processedList.Title = fmt.Sprintf("✅ Processed Papers (%d total)", len(items))
	m.processedList.SetShowStatusBar(false)
	m.processedList.Styles.Title = titleStyle
	if m.width > 0 && m.height > 0 {
		m.processedList.SetSize(m.width-4, m.height-8)
	}
}

// loadPapersForSelection loads papers for single selection
func (m *Model) loadPapersForSelection() {
	files, err := fileutil.GetPDFFiles(m.config.InputDir)
	if err != nil {
		m.err = err
		return
	}

	items := make([]list.Item, 0)
	for _, file := range files {
		basename := filepath.Base(file)

		// Check if processed
		hash, _ := fileutil.ComputeFileHash(file)
		if m.metadataStore.IsProcessed(hash) {
			continue // Skip already processed papers
		}

		items = append(items, item{
			title:       basename,
			description: file,
			action:      file,
		})
	}

	delegate := createStyledDelegate()
	m.singlePaperList = list.New(items, delegate, 0, 0)
	m.singlePaperList.Title = fmt.Sprintf("📄 Select Paper to Process (%d unprocessed)", len(items))
	m.singlePaperList.SetShowStatusBar(false)
	m.singlePaperList.Styles.Title = titleStyle
	if m.width > 0 && m.height > 0 {
		m.singlePaperList.SetSize(m.width-4, m.height-8)
	}
}
