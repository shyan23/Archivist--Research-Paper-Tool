package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// loadGraphMenu initializes the graph explorer menu
func (m *Model) loadGraphMenu() {
	items := []list.Item{
		item{
			title:       "📈 Dashboard & Statistics",
			description: "View graph overview, paper count, citations, authors",
			action:      "graph_dashboard",
		},
		item{
			title:       "🔍 Search Graph",
			description: "Semantic search across the knowledge graph",
			action:      "graph_search",
		},
		item{
			title:       "📚 My Papers in Graph",
			description: "View your processed papers and their relationships",
			action:      "graph_my_papers",
		},
		item{
			title:       "🌐 Open Neo4j Browser",
			description: "Visualize the graph at http://localhost:7474",
			action:      "graph_neo4j",
		},
		item{
			title:       "🔙 Back to Main Menu",
			description: "Return to main menu",
			action:      "main_menu",
		},
	}

	delegate := createStyledDelegate()
	m.graphMenu = list.New(items, delegate, 0, 0)
	m.graphMenu.Title = "Knowledge Graph Explorer"
	m.graphMenu.SetShowStatusBar(false)
	m.graphMenu.SetFilteringEnabled(false)
	m.graphMenu.Styles.Title = titleStyle
	m.graphMenu.Styles.TitleBar = titleStyle
}

// handleGraphMenuAction handles graph menu selections
func (m *Model) handleGraphMenuAction(action string) {
	switch action {
	case "graph_dashboard":
		m.navigateTo(screenGraphDashboard)
		m.fetchGraphStats()
	case "graph_search":
		m.navigateTo(screenGraphSearch)
		m.graphSearchQuery = ""
	case "graph_my_papers":
		m.navigateTo(screenGraphMyPapers)
		m.loadMyPapersInGraph()
	case "graph_neo4j":
		// Display Neo4j URL info
		m.err = fmt.Errorf("Open in browser: http://localhost:7474\nUsername: neo4j\nPassword: password")
	case "main_menu":
		m.screen = screenMain
		m.screenHistory = []screen{}
	}
}

// fetchGraphStats fetches statistics from the graph service
func (m *Model) fetchGraphStats() {
	resp, err := http.Get(m.graphServiceURL + "/api/graph/stats")
	if err != nil {
		m.err = fmt.Errorf("Graph service not available: %v\nStart it with: docker-compose -f docker-compose-graph.yml up -d", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		m.err = fmt.Errorf("Failed to read response: %v", err)
		return
	}

	var stats map[string]interface{}
	if err := json.Unmarshal(body, &stats); err != nil {
		m.err = fmt.Errorf("Failed to parse stats: %v", err)
		return
	}

	m.graphStats = stats
}

// loadMyPapersInGraph loads user's papers from the graph
func (m *Model) loadMyPapersInGraph() {
	// Get list of processed papers (those with .tex files)
	resp, err := http.Get(m.graphServiceURL + "/api/graph/stats")
	if err != nil {
		m.err = fmt.Errorf("Graph service not available: %v", err)
		return
	}
	defer resp.Body.Close()

	// For now, just show a placeholder
	// In a real implementation, we'd query the graph for papers
	items := []list.Item{
		item{
			title:       "Loading papers...",
			description: "Fetching your papers from the knowledge graph",
			action:      "",
		},
	}

	delegate := createStyledDelegate()
	m.graphMyPapers = list.New(items, delegate, 0, 0)
	m.graphMyPapers.Title = "My Papers in Knowledge Graph"
	m.graphMyPapers.SetShowStatusBar(false)
	m.graphMyPapers.SetFilteringEnabled(true)
	m.graphMyPapers.Styles.Title = titleStyle
}

// renderGraphDashboard renders the graph statistics dashboard
func (m Model) renderGraphDashboard() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("📊 KNOWLEDGE GRAPH DASHBOARD") + "\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render(m.err.Error()) + "\n\n")
		b.WriteString(helpStyle.Render("Press 'esc' to go back") + "\n")
		return b.String()
	}

	if m.graphStats == nil {
		b.WriteString(infoStyle.Render("Loading statistics...") + "\n\n")
		return b.String()
	}

	// Display statistics
	b.WriteString(boxStyle.Render(fmt.Sprintf(`
Graph Overview
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📄 Papers:        %v
👥 Authors:       %v
🔗 Citations:     %v
🔬 Methods:       %v
📊 Datasets:      %v
📚 Venues:        %v
🏛️  Institutions:  %v

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,
		m.graphStats["paper_count"],
		m.graphStats["author_count"],
		m.graphStats["citation_count"],
		m.graphStats["method_count"],
		m.graphStats["dataset_count"],
		m.graphStats["venue_count"],
		m.graphStats["institution_count"],
	)))

	b.WriteString("\n\n")
	b.WriteString(infoStyle.Render("🌐 Neo4j Browser: http://localhost:7474") + "\n")
	b.WriteString(subtitleStyle.Render("   Username: neo4j | Password: password") + "\n\n")

	b.WriteString(helpStyle.Render("Press 'esc' to go back | 'q' to quit") + "\n")

	return b.String()
}

// renderGraphSearch renders the semantic search interface
func (m Model) renderGraphSearch() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🔍 SEMANTIC GRAPH SEARCH") + "\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render(m.err.Error()) + "\n\n")
	}

	b.WriteString(subtitleStyle.Render("Search the knowledge graph using natural language") + "\n\n")

	// Search input box
	searchBox := inputBoxStyle.Render(fmt.Sprintf("Query: %s_", m.graphSearchQuery))
	b.WriteString(searchBox + "\n\n")

	b.WriteString(helpStyle.Render(`
Examples:
  • "papers about attention mechanisms"
  • "transformers and language models"
  • "neural machine translation 2017"

Press 'enter' to search | 'esc' to go back
`) + "\n")

	return b.String()
}

// renderGraphMyPapers renders the user's papers view
func (m Model) renderGraphMyPapers() string {
	if m.graphMyPapers.Items() == nil || len(m.graphMyPapers.Items()) == 0 {
		return titleStyle.Render("📚 MY PAPERS IN GRAPH") + "\n\n" +
			infoStyle.Render("Loading your papers...") + "\n\n" +
			helpStyle.Render("Press 'esc' to go back")
	}

	return m.graphMyPapers.View()
}

// handleGraphSearchInput handles text input for graph search
func (m Model) handleGraphSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.navigateBack()
		return m, nil

	case "enter":
		if m.graphSearchQuery == "" {
			m.err = fmt.Errorf("Please enter a search query")
			return m, nil
		}
		// TODO: Implement actual semantic search API call
		// For now, just show an info message
		m.err = fmt.Errorf("Semantic search will be implemented soon")
		return m, nil

	case "backspace":
		if len(m.graphSearchQuery) > 0 {
			m.graphSearchQuery = m.graphSearchQuery[:len(m.graphSearchQuery)-1]
		}
		return m, nil

	default:
		// Add character to query
		if len(msg.Runes) == 1 {
			m.graphSearchQuery += string(msg.Runes[0])
		}
	}

	return m, nil
}
