package tui

import (
	"archivist/internal/search"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// cleanTextForDisplay removes LaTeX markup and fixes escaped characters
func cleanTextForDisplay(text string) string {
	// Replace escaped newlines with spaces
	text = strings.ReplaceAll(text, "\\n", " ")
	text = strings.ReplaceAll(text, "\n", " ")

	// Remove LaTeX commands like \tilde{...}, \hat{...}, etc.
	latexCommands := regexp.MustCompile(`\\[a-zA-Z]+\{([^}]*)\}`)
	text = latexCommands.ReplaceAllString(text, "$1")

	// Remove standalone LaTeX commands like \alpha, \beta, etc.
	standaloneLatex := regexp.MustCompile(`\\[a-zA-Z]+`)
	text = standaloneLatex.ReplaceAllString(text, "")

	// Remove extra spaces
	multipleSpaces := regexp.MustCompile(`\s+`)
	text = multipleSpaces.ReplaceAllString(text, " ")

	// Trim
	text = strings.TrimSpace(text)

	return text
}

// renderSearchScreen renders the search input screen
func (m Model) renderSearchScreen() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("🔍 Search for Research Papers") + "\n\n")
	sb.WriteString("Search across arXiv, OpenReview, and ACL Anthology\n\n")

	// Show loading animation if searching
	if m.searchLoading {
		return renderLoadingAnimation(m.searchLoadingFrame, m.searchInput)
	}

	// Show error if there was one
	if m.searchError != "" {
		sb.WriteString(warningStyle.Render("⚠️  " + m.searchError) + "\n\n")
	}

	// Input fields based on mode
	if m.searchInputMode == "count" {
		// Show query (already entered) and ask for count
		sb.WriteString(successStyle.Render("Query: ") + m.searchInput + "\n\n")
		sb.WriteString(inputBoxStyle.Render("How many papers? (5-100): " + m.searchMaxResults + "█") + "\n\n")
		sb.WriteString(helpStyle.Render("Enter a number between 5 and 100, then press Enter\n"))
		sb.WriteString(helpStyle.Render("Default: 20 papers\n\n"))
	} else {
		// Query input mode (default)
		sb.WriteString(inputBoxStyle.Render("Query: " + m.searchInput + "█") + "\n\n")

		// Instructions
		sb.WriteString(helpStyle.Render("Type your search query and press Enter\n"))
		sb.WriteString(helpStyle.Render("Examples:\n"))
		sb.WriteString(helpStyle.Render("  • \"transformer architecture\"\n"))
		sb.WriteString(helpStyle.Render("  • \"vision transformers\"\n"))
		sb.WriteString(helpStyle.Render("  • \"attention mechanisms\"\n\n"))
	}

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
	// If in query mode, move to count mode
	if m.searchInputMode == "" || m.searchInputMode == "query" {
		if m.searchInput == "" {
			return m, nil
		}
		// Switch to count input mode
		m.searchInputMode = "count"
		m.searchMaxResults = "20" // Default
		return m, nil
	}

	// If in count mode, perform search
	if m.searchInputMode == "count" {
		// Parse max results
		maxResults := 20 // Default
		if m.searchMaxResults != "" {
			fmt.Sscanf(m.searchMaxResults, "%d", &maxResults)
			if maxResults < 5 {
				maxResults = 5
			}
			if maxResults > 100 {
				maxResults = 100
			}
		}

		// Clear previous error
		m.searchError = ""

		// Create search client
		client := search.NewClient("http://localhost:8000")

		// Check if service is running
		if !client.IsServiceRunning() {
			m.searchError = "Search service is not running"
			return m, nil
		}

		// Start loading animation
		m.searchLoading = true
		m.searchLoadingFrame = 0

		// Perform search asynchronously and start ticker for animation
		return m, tea.Batch(
			m.performSearch(client, m.searchInput, maxResults),
			tickEvery(100 * time.Millisecond),
		)
	}

	return m, nil
}

// performSearch executes the search and returns results
func (m *Model) performSearch(client *search.Client, query string, maxResults int) tea.Cmd {
	return func() tea.Msg {
		searchQuery := &search.SearchQuery{
			Query:      query,
			MaxResults: maxResults,
			Sources:    []string{}, // Search all sources
		}

		results, err := client.Search(searchQuery)
		return searchResultMsg{
			results: results,
			err:     err,
		}
	}
}

// searchResultMsg contains search results
type searchResultMsg struct {
	results *search.SearchResponse
	err     error
}

// handleSearchResult processes search results
func (m *Model) handleSearchResult(msg searchResultMsg) (tea.Model, tea.Cmd) {
	m.searchLoading = false

	if msg.err != nil {
		m.searchError = fmt.Sprintf("Search failed: %v", msg.err)
		return m, nil
	}

	if msg.results.Total == 0 {
		m.searchError = "No results found for your query"
		return m, nil
	}

	results := msg.results

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
