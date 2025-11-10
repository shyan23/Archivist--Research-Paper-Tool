package tui

import (
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

		items[i] = item{
			title:       basename,
			description: file,
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

// loadProcessedPapers loads papers from reports folder
func (m *Model) loadProcessedPapers() {
	files, err := fileutil.GetPDFFiles(m.config.ReportOutputDir)
	if err != nil {
		m.err = err
		return
	}

	items := make([]list.Item, 0)
	for _, file := range files {
		basename := filepath.Base(file)

		items = append(items, item{
			title:       basename,
			description: file,
			action:      file,
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

		items = append(items, item{
			title:       basename,
			description: file,
			action:      file,
		})
	}

	delegate := createStyledDelegate()
	m.singlePaperList = list.New(items, delegate, 0, 0)
	m.singlePaperList.Title = fmt.Sprintf("📄 Select Paper to Process (%d papers)", len(items))
	m.singlePaperList.SetShowStatusBar(false)
	m.singlePaperList.Styles.Title = titleStyle
	if m.width > 0 && m.height > 0 {
		m.singlePaperList.SetSize(m.width-4, m.height-8)
	}
}

// loadPapersForMultiSelection loads papers for multi-selection
func (m *Model) loadPapersForMultiSelection() {
	files, err := fileutil.GetPDFFiles(m.config.InputDir)
	if err != nil {
		m.err = err
		return
	}

	// Store all files for later reference
	m.allPapersForSelect = files
	m.multiSelectIndexes = make(map[int]bool)

	items := make([]list.Item, 0)
	for _, file := range files {
		basename := filepath.Base(file)

		items = append(items, item{
			title:       basename,
			description: file,
			action:      file,
		})
	}

	delegate := createStyledDelegate()
	m.multiPaperList = list.New(items, delegate, 0, 0)
	m.multiPaperList.Title = fmt.Sprintf("📋 Select Papers (Space to toggle, Enter to confirm) - %d available", len(items))
	m.multiPaperList.SetShowStatusBar(false)
	m.multiPaperList.Styles.Title = titleStyle
	if m.width > 0 && m.height > 0 {
		m.multiPaperList.SetSize(m.width-4, m.height-8)
	}
}
