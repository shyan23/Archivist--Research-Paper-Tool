package tui

import (
	"archivist/internal/search"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// renderSearchScreen renders the search input screen
func (m Model) renderSearchScreen() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("🔍 Search for Research Papers") + "\n\n")
	sb.WriteString("Search across arXiv, OpenReview, and ACL Anthology\n\n")

	// Show loading state if searching
	if m.searchLoading {
		sb.WriteString(successStyle.Render("🔄 Searching...") + "\n\n")
		sb.WriteString(helpStyle.Render("Please wait while we search for papers...") + "\n")
		return sb.String()
	}

	// Show error if there was one
	if m.searchError != "" {
		sb.WriteString(warningStyle.Render("⚠️  " + m.searchError) + "\n\n")
	}

	// Input field
	sb.WriteString(inputBoxStyle.Render("Query: " + m.searchInput + "█") + "\n\n")

	// Instructions
	sb.WriteString(helpStyle.Render("Type your search query and press Enter\n"))
	sb.WriteString(helpStyle.Render("Examples:\n"))
	sb.WriteString(helpStyle.Render("  • \"transformer architecture\"\n"))
	sb.WriteString(helpStyle.Render("  • \"vision transformers\"\n"))
	sb.WriteString(helpStyle.Render("  • \"attention mechanisms\"\n\n"))

	// Check if search service is running
	client := search.NewClient("http://localhost:8000")
	if !client.IsServiceRunning() {
		sb.WriteString(warningStyle.Render("\n⚠️  Search service is not running\n\n"))
		sb.WriteString(helpStyle.Render("To start the search service:\n"))
		sb.WriteString(helpStyle.Render("  cd services/search-engine\n"))
		sb.WriteString(helpStyle.Render("  source venv/bin/activate\n"))
		sb.WriteString(helpStyle.Render("  python run.py\n"))
	} else {
		sb.WriteString(successStyle.Render("✓ Search service is running\n"))
	}

	return sb.String()
}

// handleSearchEnter processes the search query when Enter is pressed
func (m *Model) handleSearchEnter() (tea.Model, tea.Cmd) {
	if m.searchInput == "" {
		return m, nil
	}

	// Clear previous error
	m.searchError = ""

	// Create search client
	client := search.NewClient("http://localhost:8000")

	// Check if service is running
	if !client.IsServiceRunning() {
		// Service not running, stay on search screen
		m.searchError = "Search service is not running"
		return m, nil
	}

	// Perform search
	m.searchLoading = true
	query := &search.SearchQuery{
		Query:      m.searchInput,
		MaxResults: 20,
		Sources:    []string{}, // Search all sources
	}

	// Execute search synchronously (could be made async with tea.Cmd)
	results, err := client.Search(query)
	m.searchLoading = false

	if err != nil {
		// Show error message
		m.searchError = fmt.Sprintf("Search failed: %v", err)
		return m, nil
	}

	if results.Total == 0 {
		// No results, stay on search screen
		m.searchError = "No results found for your query"
		return m, nil
	}

	// Convert results to list items
	items := make([]list.Item, len(results.Results))
	for i, result := range results.Results {
		// Truncate abstract for display
		abstract := result.Abstract
		if len(abstract) > 150 {
			abstract = abstract[:150] + "..."
		}

		items[i] = item{
			title:       result.Title,
			description: fmt.Sprintf("%s | %s | %s", result.Source, result.Venue, abstract),
			action:      result.PDFURL, // Store PDF URL in action field
		}
	}

	// Create results list
	delegate := createStyledDelegate()
	m.searchResultsList = list.New(items, delegate, m.width, m.height)
	m.searchResultsList.Title = fmt.Sprintf("Search Results: \"%s\" (%d papers found)", m.searchInput, results.Total)
	m.searchResultsList.SetShowStatusBar(false)
	m.searchResultsList.SetFilteringEnabled(false)
	m.searchResultsList.Styles.Title = titleStyle

	// Set proper size for the list
	if m.width > 0 && m.height > 0 {
		m.searchResultsList.SetSize(m.width-4, m.height-8)
	}

	// Navigate to results screen
	m.navigateTo(screenSearchResults)

	return m, nil
}

// handleSearchResultSelection handles selection of a search result
func (m Model) handleSearchResultSelection() (tea.Model, tea.Cmd) {
	selectedItem := m.searchResultsList.SelectedItem()
	if selectedItem == nil {
		return m, nil
	}

	// Get the PDF URL from the action field
	pdfURL := selectedItem.(item).action
	paperTitle := selectedItem.(item).title

	// Store download information and exit TUI to download
	m.processingMsg = "download:" + pdfURL + "|" + paperTitle
	return m, tea.Quit
}
