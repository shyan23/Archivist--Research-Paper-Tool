package search

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a Go client for the Python search microservice
type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient creates a new search client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	return &Client{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SearchQuery represents a search request
type SearchQuery struct {
	Query      string     `json:"query"`
	MaxResults int        `json:"max_results"`
	Sources    []string   `json:"sources,omitempty"`
	StartDate  *time.Time `json:"start_date,omitempty"`
	EndDate    *time.Time `json:"end_date,omitempty"`
}

// SearchResult represents a single paper result
type SearchResult struct {
	Title           string    `json:"title"`
	Authors         []string  `json:"authors"`
	Abstract        string    `json:"abstract"`
	PublishedAt     time.Time `json:"published_at"`
	PDFURL          string    `json:"pdf_url"`
	SourceURL       string    `json:"source_url"`
	Source          string    `json:"source"`
	Venue           string    `json:"venue"`
	ID              string    `json:"id"`
	Categories      []string  `json:"categories"`
	RelevanceScore  *float64  `json:"relevance_score,omitempty"`
	FuzzyScore      *float64  `json:"fuzzy_score,omitempty"`
	SimilarityScore *float64  `json:"similarity_score,omitempty"`
}

// SearchResponse represents the API response
type SearchResponse struct {
	Query           string         `json:"query"`
	Total           int            `json:"total"`
	Results         []SearchResult `json:"results"`
	SourcesSearched []string       `json:"sources_searched"`
}

// DownloadRequest represents a download request
type DownloadRequest struct {
	PDFURL   string `json:"pdf_url"`
	Filename string `json:"filename,omitempty"`
}

// DownloadResponse represents a download response
type DownloadResponse struct {
	Success   bool   `json:"success"`
	Filename  string `json:"filename"`
	SizeBytes int    `json:"size_bytes"`
	Message   string `json:"message"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status    string   `json:"status"`
	Version   string   `json:"version"`
	Providers []string `json:"providers"`
}

// Search performs a search across all or specified sources
func (c *Client) Search(query *SearchQuery) (*SearchResponse, error) {
	if query.Query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	if query.MaxResults <= 0 {
		query.MaxResults = 20
	}

	// Marshal request
	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Make request
	url := fmt.Sprintf("%s/api/search", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DownloadPaper downloads a paper PDF
func (c *Client) DownloadPaper(pdfURL, filename string) (*DownloadResponse, error) {
	if pdfURL == "" {
		return nil, fmt.Errorf("PDF URL cannot be empty")
	}

	req := DownloadRequest{
		PDFURL:   pdfURL,
		Filename: filename,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/download", c.baseURL)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result DownloadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// HealthCheck checks if the search service is running
func (c *Client) HealthCheck() (*HealthResponse, error) {
	url := fmt.Sprintf("%s/health", c.baseURL)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("service unhealthy: status %d", resp.StatusCode)
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &health, nil
}

// IsServiceRunning checks if the search service is accessible
func (c *Client) IsServiceRunning() bool {
	_, err := c.HealthCheck()
	return err == nil
}
