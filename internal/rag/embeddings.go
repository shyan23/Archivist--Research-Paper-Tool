package rag

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const (
	// EmbeddingModel is the Gemini embedding model to use
	EmbeddingModel = "models/text-embedding-004"
	// EmbeddingDimensions is the output dimension size
	EmbeddingDimensions = 768
)

// EmbeddingClient handles text embedding generation using Gemini API
type EmbeddingClient struct {
	client *genai.Client
	model  string
}

// NewEmbeddingClient creates a new embedding client
func NewEmbeddingClient(apiKey string) (*EmbeddingClient, error) {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &EmbeddingClient{
		client: client,
		model:  EmbeddingModel,
	}, nil
}

// Close closes the embedding client
func (ec *EmbeddingClient) Close() error {
	return ec.client.Close()
}

// GenerateEmbedding generates an embedding vector for a single text
func (ec *EmbeddingClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	model := ec.client.EmbeddingModel(ec.model)

	res, err := model.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if res.Embedding == nil || len(res.Embedding.Values) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	return res.Embedding.Values, nil
}

// GenerateBatchEmbeddings generates embeddings for multiple texts
// Note: Processes one at a time as batch API may not be available in all versions
func (ec *EmbeddingClient) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	embeddings := make([][]float32, len(texts))

	// Process each text individually
	// In production, optimize with actual batch API if available
	for i, text := range texts {
		embedding, err := ec.GenerateEmbedding(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for text %d: %w", i, err)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}
