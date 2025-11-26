package tui

import (
	"archivist/internal/analyzer"
	"archivist/internal/search"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// getPapersInDirectory returns all PDF files in a directory
func getPapersInDirectory(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	papers := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".pdf") {
			papers = append(papers, filepath.Join(dir, entry.Name()))
		}
	}

	return papers, nil
}

// renderSearchModeScreen renders the search mode selection screen
func (m Model) renderSearchModeScreen() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("🔍 Search Mode Selection") + "\n\n")
	sb.WriteString("Choose how you want to search for papers:\n\n")

	sb.WriteString(m.searchModeMenu.View())

	return sb.String()
}

// renderSimilarPaperSelectScreen renders the paper selection screen for similar search
func (m Model) renderSimilarPaperSelectScreen() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("📄 Select Paper for Similar Search") + "\n\n")
	sb.WriteString("Choose a paper to find similar papers:\n\n")

	if m.similarExtractingEssence {
		sb.WriteString(successStyle.Render("🔄 Extracting paper essence...") + "\n\n")
		sb.WriteString(helpStyle.Render("Analyzing paper to identify key concepts, methodology, and techniques...\n"))
		return sb.String()
	}

	if m.similarEssenceError != "" {
		sb.WriteString(warningStyle.Render("⚠️  " + m.similarEssenceError) + "\n\n")
	}

	sb.WriteString(m.similarPaperList.View())

	return sb.String()
}

// renderSimilarFactorsEditScreen renders the factor editing screen
func (m Model) renderSimilarFactorsEditScreen() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("✏️  Edit Search Factors") + "\n\n")
	sb.WriteString(fmt.Sprintf("Paper: %s\n\n", successStyle.Render(filepath.Base(m.selectedSimilarPaper))))

	if len(m.similarFactors) == 0 {
		sb.WriteString(warningStyle.Render("No factors extracted. Press Enter to search anyway or Esc to go back.\n\n"))
		return sb.String()
	}

	sb.WriteString("These factors will be used to find similar papers:\n\n")

	// Display factors as a numbered list with highlighting
	for i, factor := range m.similarFactors {
		// Get the selected index from the list
		selected := m.similarFactorsList.Index() == i

		if selected {
			sb.WriteString(highlightStyle.Render(fmt.Sprintf("  [%d] %s  ← (Press 'd' to delete)", i+1, factor)) + "\n")
		} else {
			sb.WriteString(fmt.Sprintf("  [%d] %s\n", i+1, factor))
		}
	}

	sb.WriteString("\n")

	// Add new factor input
	sb.WriteString(inputBoxStyle.Render("Add factor: " + m.similarFactorInput + "█") + "\n\n")

	// Instructions
	sb.WriteString(helpStyle.Render("Instructions:\n"))
	sb.WriteString(helpStyle.Render("  • ↑/↓: Navigate factors\n"))
	sb.WriteString(helpStyle.Render("  • d: Delete selected factor\n"))
	sb.WriteString(helpStyle.Render("  • Type to add new factor, press Enter to confirm\n"))
	sb.WriteString(helpStyle.Render("  • Tab: Start search with current factors\n"))
	sb.WriteString(helpStyle.Render("  • Esc: Go back\n\n"))

	sb.WriteString(successStyle.Render(fmt.Sprintf("Total factors: %d", len(m.similarFactors))))

	return sb.String()
}

// handleSearchModeSelection handles selection from the search mode menu
func (m *Model) handleSearchModeSelection() (tea.Model, tea.Cmd) {
	selectedItem := m.searchModeMenu.SelectedItem()
	if selectedItem == nil {
		return m, nil
	}

	action := selectedItem.(item).action

	switch action {
	case "manual":
		// Go to manual search screen and initialize state
		m.searchInput = ""
		m.searchMaxResults = ""
		m.searchInputMode = "query"
		m.searchLoading = false
		m.searchError = ""
		m.navigateTo(screenSearch)
		return m, nil

	case "similar":
		// Go to paper selection for similar search
		// Load papers from library
		papers, err := getPapersInDirectory(m.config.InputDir)
		if err != nil || len(papers) == 0 {
			m.similarEssenceError = "No papers found in library"
			return m, nil
		}

		// Create list of papers
		items := make([]list.Item, len(papers))
		for i, paper := range papers {
			items[i] = item{
				title:       filepath.Base(paper),
				description: filepath.Dir(paper),  // Show only directory, not full path with filename
				action:      paper,
			}
		}

		delegate := createStyledDelegate()
		m.similarPaperList = list.New(items, delegate, m.width, m.height)
		m.similarPaperList.Title = "Select a paper from your library"
		m.similarPaperList.SetShowStatusBar(false)
		m.similarPaperList.SetFilteringEnabled(true)
		m.similarPaperList.Styles.Title = titleStyle

		if m.width > 0 && m.height > 0 {
			m.similarPaperList.SetSize(m.width-4, m.height-8)
		}

		m.navigateTo(screenSimilarPaperSelect)
		return m, nil
	}

	return m, nil
}

// handleSimilarPaperSelection handles selection of a paper for similar search
func (m *Model) handleSimilarPaperSelection() (tea.Model, tea.Cmd) {
	selectedItem := m.similarPaperList.SelectedItem()
	if selectedItem == nil {
		return m, nil
	}

	m.selectedSimilarPaper = selectedItem.(item).action
	m.similarExtractingEssence = true
	m.similarEssenceError = ""

	// Extract essence in a goroutine (simulated async)
	return m, m.extractPaperEssence()
}

// extractPaperEssence extracts the essence from the selected paper
func (m *Model) extractPaperEssence() tea.Cmd {
	return func() tea.Msg {
		// Create analyzer
		analyzerInstance, err := analyzer.NewAnalyzer(m.config)
		if err != nil {
			return essenceExtractedMsg{err: fmt.Errorf("failed to create analyzer: %w", err)}
		}
		defer analyzerInstance.Close()

		// Create similar paper finder
		finder := analyzer.NewSimilarPaperFinder(analyzerInstance, "http://localhost:8000")

		// Extract essence
		ctx := context.Background()
		essence, err := finder.ExtractEssence(ctx, m.selectedSimilarPaper)
		if err != nil {
			return essenceExtractedMsg{err: fmt.Errorf("failed to extract essence: %w", err)}
		}

		// Convert essence to factors list
		factors := make([]string, 0)

		// Add methodology
		if essence.MainMethodology != "" {
			factors = append(factors, essence.MainMethodology)
		}

		// Add key concepts
		factors = append(factors, essence.KeyConcepts...)

		// Add techniques
		factors = append(factors, essence.Techniques...)

		return essenceExtractedMsg{
			factors: factors,
			essence: essence,
		}
	}
}

// essenceExtractedMsg is sent when essence extraction completes
type essenceExtractedMsg struct {
	factors []string
	essence *analyzer.PaperEssence
	err     error
}

// handleEssenceExtracted processes the essence extraction result
func (m *Model) handleEssenceExtracted(msg essenceExtractedMsg) (tea.Model, tea.Cmd) {
	m.similarExtractingEssence = false

	if msg.err != nil {
		m.similarEssenceError = msg.err.Error()
		return m, nil
	}

	// Set factors
	m.similarFactors = msg.factors

	// Create factors list for editing
	items := make([]list.Item, len(m.similarFactors))
	for i, factor := range m.similarFactors {
		items[i] = item{
			title:       factor,
			description: "",
			action:      "",
		}
	}

	delegate := createStyledDelegate()
	m.similarFactorsList = list.New(items, delegate, m.width, m.height)
	m.similarFactorsList.Title = "Edit Factors"
	m.similarFactorsList.SetShowStatusBar(false)
	m.similarFactorsList.SetFilteringEnabled(false)
	m.similarFactorsList.Styles.Title = titleStyle

	if m.width > 0 && m.height > 0 {
		m.similarFactorsList.SetSize(m.width-4, m.height-12)
	}

	// Navigate to factor editing screen
	m.navigateTo(screenSimilarFactorsEdit)

	return m, nil
}

// handleSimilarFactorsEdit handles editing of search factors
func (m *Model) handleSimilarFactorsEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "d":
		// Delete selected factor
		if len(m.similarFactors) > 0 {
			idx := m.similarFactorsList.Index()
			if idx >= 0 && idx < len(m.similarFactors) {
				// Remove factor
				m.similarFactors = append(m.similarFactors[:idx], m.similarFactors[idx+1:]...)

				// Update list
				items := make([]list.Item, len(m.similarFactors))
				for i, factor := range m.similarFactors {
					items[i] = item{
						title:       factor,
						description: "",
						action:      "",
					}
				}

				m.similarFactorsList.SetItems(items)

				// Adjust cursor if needed
				if idx >= len(m.similarFactors) && len(m.similarFactors) > 0 {
					m.similarFactorsList.Select(len(m.similarFactors) - 1)
				}
			}
		}
		return m, nil

	case "enter":
		// Add new factor if input is not empty
		if strings.TrimSpace(m.similarFactorInput) != "" {
			m.similarFactors = append(m.similarFactors, strings.TrimSpace(m.similarFactorInput))
			m.similarFactorInput = ""

			// Update list
			items := make([]list.Item, len(m.similarFactors))
			for i, factor := range m.similarFactors {
				items[i] = item{
					title:       factor,
					description: "",
					action:      "",
				}
			}
			m.similarFactorsList.SetItems(items)
		}
		return m, nil

	case "tab":
		// Start search with current factors
		return m.executeSimilarSearch()

	case "backspace":
		// Delete last character from input
		if len(m.similarFactorInput) > 0 {
			m.similarFactorInput = m.similarFactorInput[:len(m.similarFactorInput)-1]
		}
		return m, nil

	case "up", "k":
		// Navigate list
		var cmd tea.Cmd
		m.similarFactorsList, cmd = m.similarFactorsList.Update(msg)
		return m, cmd

	case "down", "j":
		// Navigate list
		var cmd tea.Cmd
		m.similarFactorsList, cmd = m.similarFactorsList.Update(msg)
		return m, cmd

	default:
		// Add to input if it's a regular character
		if len(msg.String()) == 1 {
			m.similarFactorInput += msg.String()
		}
		return m, nil
	}
}

// executeSimilarSearch performs the search with the edited factors
func (m *Model) executeSimilarSearch() (tea.Model, tea.Cmd) {
	if len(m.similarFactors) == 0 {
		m.similarEssenceError = "No factors to search with"
		return m, nil
	}

	// Build search query from factors
	query := strings.Join(m.similarFactors, " ")

	// Create search client
	client := search.NewClient("http://localhost:8000")

	// Check if service is running
	if !client.IsServiceRunning() {
		m.searchError = "Search service is not running"
		return m, nil
	}

	// Perform search
	m.searchLoading = true
	searchQuery := &search.SearchQuery{
		Query:      query,
		MaxResults: 20,
		Sources:    []string{}, // Search all sources
	}

	// Execute search synchronously
	results, err := client.Search(searchQuery)
	m.searchLoading = false

	if err != nil {
		m.searchError = fmt.Sprintf("Search failed: %v", err)
		m.navigateTo(screenSimilarFactorsEdit)
		return m, nil
	}

	if results.Total == 0 {
		m.searchError = "No similar papers found"
		m.navigateTo(screenSimilarFactorsEdit)
		return m, nil
	}

	// Convert results to list items
	items := make([]list.Item, len(results.Results))
	for i, result := range results.Results {
		// Clean title and abstract from LaTeX and escaped characters
		cleanTitle := cleanTextForDisplay(result.Title)
		cleanAbstract := cleanTextForDisplay(result.Abstract)

		// Truncate abstract for display
		if len(cleanAbstract) > 150 {
			cleanAbstract = cleanAbstract[:150] + "..."
		}

		items[i] = item{
			title:       cleanTitle,
			description: fmt.Sprintf("%s | %s | %s", result.Source, result.Venue, cleanAbstract),
			action:      result.PDFURL,
		}
	}

	// Create results list
	delegate := createStyledDelegate()
	m.searchResultsList = list.New(items, delegate, m.width, m.height)
	m.searchResultsList.Title = fmt.Sprintf("Similar Papers (%d found)", results.Total)
	m.searchResultsList.SetShowStatusBar(false)
	m.searchResultsList.SetFilteringEnabled(false)
	m.searchResultsList.Styles.Title = titleStyle

	if m.width > 0 && m.height > 0 {
		m.searchResultsList.SetSize(m.width-4, m.height-8)
	}

	// Navigate to results screen
	m.navigateTo(screenSearchResults)

	return m, nil
}
