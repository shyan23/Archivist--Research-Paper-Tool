package rag

import (
	"archivist/internal/analyzer"
	"context"
	"fmt"
	"log"
)

// IndexPDFDirectly indexes a PDF directly using Gemini vision without processing
func (i *Indexer) IndexPDFDirectly(ctx context.Context, pdfPath, geminiAPIKey string) error {
	paperTitle := extractTitleFromPath(pdfPath)

	log.Printf("  📇 Indexing PDF directly: %s", paperTitle)

	// Use Gemini to extract text from PDF
	log.Println("  📄 Extracting text from PDF with Gemini...")
	geminiClient, err := analyzer.NewGeminiClient(geminiAPIKey, "models/gemini-2.0-flash-exp", 0.3, 8000)
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer geminiClient.Close()

	// Extract text using Gemini's multimodal capabilities
	prompt := `Extract all text content from this PDF research paper.
Include all sections, paragraphs, equations, and technical content.
Preserve the structure and formatting as much as possible.
Return only the extracted text, no additional commentary.`

	extractedText, err := geminiClient.AnalyzePDFWithVision(ctx, pdfPath, prompt)
	if err != nil {
		return fmt.Errorf("failed to extract PDF text: %w", err)
	}

	if extractedText == "" {
		return fmt.Errorf("no text extracted from PDF")
	}

	log.Printf("  ✓ Extracted %d characters of text", len(extractedText))

	// Chunk the extracted text
	log.Println("  ✂️  Chunking text...")
	chunks, err := i.chunker.ChunkText(extractedText, paperTitle)
	if err != nil {
		return fmt.Errorf("failed to chunk content: %w", err)
	}

	if len(chunks) == 0 {
		return fmt.Errorf("no chunks generated from content")
	}

	log.Printf("  ✓ Created %d chunks", len(chunks))

	// Extract text for embedding
	texts := make([]string, len(chunks))
	for idx, chunk := range chunks {
		texts[idx] = chunk.Text
	}

	// Generate embeddings in batch
	log.Println("  🧮 Generating embeddings...")
	embeddings, err := i.embedClient.GenerateBatchEmbeddings(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	log.Printf("  ✓ Generated %d embeddings", len(embeddings))

	// Create vector documents
	docs := make([]VectorDocument, len(chunks))
	for idx, chunk := range chunks {
		docID := generateDocID(paperTitle, chunk.ChunkIndex)

		docs[idx] = VectorDocument{
			ID:         docID,
			ChunkText:  chunk.Text,
			Embedding:  embeddings[idx],
			Source:     paperTitle,
			Section:    chunk.Section,
			ChunkIndex: chunk.ChunkIndex,
			Metadata: map[string]string{
				"source":      paperTitle,
				"section":     chunk.Section,
				"chunk_index": fmt.Sprintf("%d", chunk.ChunkIndex),
				"pdf_path":    pdfPath,
				"indexed_directly": "true",
			},
		}
	}

	// Store in vector database
	log.Println("  💾 Storing vectors in database...")
	if err := i.vectorStore.AddDocuments(ctx, docs); err != nil {
		return fmt.Errorf("failed to store vectors: %w", err)
	}

	log.Printf("  ✅ Successfully indexed PDF: %s (%d chunks)", paperTitle, len(chunks))

	return nil
}

// QuickIndexForChat does a fast index for immediate chat (lower chunk count)
func (i *Indexer) QuickIndexForChat(ctx context.Context, pdfPath, geminiAPIKey string) error {
	paperTitle := extractTitleFromPath(pdfPath)

	// Check if already indexed
	indexed, _, err := i.CheckIfIndexed(ctx, paperTitle)
	if err == nil && indexed {
		log.Printf("  ✓ Paper already indexed: %s", paperTitle)
		return nil
	}

	log.Printf("  🚀 Quick indexing for chat: %s", paperTitle)

	// Use larger chunks for faster indexing
	fastChunker := NewChunker(4000, 400) // Double the chunk size
	fastIndexer := NewIndexer(fastChunker, i.embedClient, i.vectorStore)

	return fastIndexer.IndexPDFDirectly(ctx, pdfPath, geminiAPIKey)
}
