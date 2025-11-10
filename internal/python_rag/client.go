package python_rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PythonRAGClient is a client for the Python RAG API
type PythonRAGClient struct {
	baseURL string
	client  *http.Client
}

// NewPythonRAGClient creates a new Python RAG API client
func NewPythonRAGClient(baseURL string) *PythonRAGClient {
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	return &PythonRAGClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 120 * time.Second, // Increased timeout for embedding/chat
		},
	}
}

// IndexPaperRequest represents a paper indexing request
type IndexPaperRequest struct {
	PaperTitle   string `json:"paper_title"`
	LatexContent string `json:"latex_content"`
	PDFPath      string `json:"pdf_path,omitempty"`
	ForceReindex bool   `json:"force_reindex"`
}

// IndexPaperResponse represents a paper indexing response
type IndexPaperResponse struct {
	Success    bool   `json:"success"`
	PaperTitle string `json:"paper_title"`
	NumChunks  int    `json:"num_chunks"`
	Message    string `json:"message"`
}

// CreateSessionRequest represents a chat session creation request
type CreateSessionRequest struct {
	PaperTitles []string `json:"paper_titles"`
}

// CreateSessionResponse represents a chat session creation response
type CreateSessionResponse struct {
	SessionID   string   `json:"session_id"`
	PaperTitles []string `json:"paper_titles"`
	CreatedAt   float64  `json:"created_at"`
}

// ChatRequest represents a chat message request
type ChatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// ChatResponse represents a chat message response
type ChatResponse struct {
	Role      string   `json:"role"`
	Content   string   `json:"content"`
	Timestamp float64  `json:"timestamp"`
	Citations []string `json:"citations"`
}

// RetrieveRequest represents a context retrieval request
type RetrieveRequest struct {
	Query       string   `json:"query"`
	PaperTitles []string `json:"paper_titles,omitempty"`
	TopK        int      `json:"top_k"`
}

// RetrieveResponse represents a context retrieval response
type RetrieveResponse struct {
	Query        string   `json:"query"`
	ContextText  string   `json:"context_text"`
	Sources      []string `json:"sources"`
	Sections     []string `json:"sections"`
	TotalChunks  int      `json:"total_chunks"`
}

// SystemInfoResponse represents system information
type SystemInfoResponse struct {
	Status            string   `json:"status"`
	TotalDocuments    int      `json:"total_documents"`
	IndexedPapers     []string `json:"indexed_papers"`
	EmbeddingModel    string   `json:"embedding_model"`
	EmbeddingDimension int     `json:"embedding_dimension"`
	VectorStore       string   `json:"vector_store"`
}

// HealthCheck checks if the Python API is running
func (c *PythonRAGClient) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Python RAG API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// GetSystemInfo retrieves system information
func (c *PythonRAGClient) GetSystemInfo(ctx context.Context) (*SystemInfoResponse, error) {
	url := fmt.Sprintf("%s/system/info", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get system info: status %d", resp.StatusCode)
	}

	var info SystemInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return &info, nil
}

// IndexPaper indexes a research paper
func (c *PythonRAGClient) IndexPaper(ctx context.Context, req *IndexPaperRequest) (*IndexPaperResponse, error) {
	url := fmt.Sprintf("%s/index/paper", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("indexing failed: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var indexResp IndexPaperResponse
	if err := json.NewDecoder(resp.Body).Decode(&indexResp); err != nil {
		return nil, err
	}

	return &indexResp, nil
}

// GetIndexedPapers retrieves list of indexed papers
func (c *PythonRAGClient) GetIndexedPapers(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/index/papers", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get papers: status %d", resp.StatusCode)
	}

	var result struct {
		Papers []string `json:"papers"`
		Total  int      `json:"total"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Papers, nil
}

// CreateChatSession creates a new chat session
func (c *PythonRAGClient) CreateChatSession(ctx context.Context, paperTitles []string) (*CreateSessionResponse, error) {
	url := fmt.Sprintf("%s/chat/session", c.baseURL)

	req := CreateSessionRequest{
		PaperTitles: paperTitles,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to create session: status %d", resp.StatusCode)
	}

	var sessionResp CreateSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return nil, err
	}

	return &sessionResp, nil
}

// SendChatMessage sends a message in a chat session
func (c *PythonRAGClient) SendChatMessage(ctx context.Context, sessionID, message string) (*ChatResponse, error) {
	url := fmt.Sprintf("%s/chat/message", c.baseURL)

	req := ChatRequest{
		SessionID: sessionID,
		Message:   message,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("chat failed: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, err
	}

	return &chatResp, nil
}

// Retrieve retrieves relevant context for a query
func (c *PythonRAGClient) Retrieve(ctx context.Context, query string, paperTitles []string, topK int) (*RetrieveResponse, error) {
	url := fmt.Sprintf("%s/retrieve", c.baseURL)

	if topK <= 0 {
		topK = 5
	}

	req := RetrieveRequest{
		Query:       query,
		PaperTitles: paperTitles,
		TopK:        topK,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("retrieval failed: status %d", resp.StatusCode)
	}

	var retrieveResp RetrieveResponse
	if err := json.NewDecoder(resp.Body).Decode(&retrieveResp); err != nil {
		return nil, err
	}

	return &retrieveResp, nil
}
