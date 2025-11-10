package rag

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	// DefaultChunkSize is the target number of characters per chunk
	DefaultChunkSize = 2000
	// DefaultChunkOverlap is the number of characters to overlap between chunks
	DefaultChunkOverlap = 200
	// MinChunkSize is the minimum chunk size to avoid tiny fragments
	MinChunkSize = 100
)

// Chunk represents a text chunk with metadata
type Chunk struct {
	Text         string            `json:"text"`
	ChunkIndex   int               `json:"chunk_index"`
	Source       string            `json:"source"`        // Paper title or file path
	Section      string            `json:"section"`       // Section name if available
	StartOffset  int               `json:"start_offset"`  // Character offset in original text
	EndOffset    int               `json:"end_offset"`    // Character offset in original text
	Metadata     map[string]string `json:"metadata"`      // Additional metadata
}

// Chunker handles intelligent text chunking
type Chunker struct {
	chunkSize    int
	chunkOverlap int
}

// NewChunker creates a new text chunker
func NewChunker(chunkSize, chunkOverlap int) *Chunker {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	if chunkOverlap < 0 {
		chunkOverlap = DefaultChunkOverlap
	}
	if chunkOverlap >= chunkSize {
		chunkOverlap = chunkSize / 4 // Ensure overlap is less than chunk size
	}

	return &Chunker{
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
	}
}

// ChunkText splits text into overlapping chunks with smart boundaries
func (c *Chunker) ChunkText(text, source string) ([]Chunk, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text provided")
	}

	// Clean the text
	text = c.cleanText(text)

	// Split into sentences for smart boundaries
	sentences := c.splitIntoSentences(text)
	if len(sentences) == 0 {
		return nil, fmt.Errorf("no sentences found in text")
	}

	var chunks []Chunk
	var currentChunk strings.Builder
	var currentSentences []string
	currentOffset := 0
	chunkIndex := 0

	for i, sentence := range sentences {
		testChunk := currentChunk.String() + sentence

		// If adding this sentence exceeds chunk size, create a chunk
		if utf8.RuneCountInString(testChunk) > c.chunkSize && currentChunk.Len() > 0 {
			// Create chunk from accumulated sentences
			chunkText := strings.TrimSpace(currentChunk.String())
			if utf8.RuneCountInString(chunkText) >= MinChunkSize {
				chunks = append(chunks, Chunk{
					Text:        chunkText,
					ChunkIndex:  chunkIndex,
					Source:      source,
					StartOffset: currentOffset,
					EndOffset:   currentOffset + len(chunkText),
					Metadata:    make(map[string]string),
				})
				chunkIndex++
			}

			// Calculate overlap: keep last N sentences
			overlapSentences := c.calculateOverlapSentences(currentSentences, c.chunkOverlap)

			// Reset for next chunk with overlap
			currentChunk.Reset()
			currentSentences = overlapSentences
			for _, s := range overlapSentences {
				currentChunk.WriteString(s)
			}

			// Update offset
			if len(chunks) > 0 {
				currentOffset = chunks[len(chunks)-1].EndOffset - c.chunkOverlap
				if currentOffset < 0 {
					currentOffset = 0
				}
			}
		}

		// Add current sentence
		currentChunk.WriteString(sentence)
		currentSentences = append(currentSentences, sentence)

		// Handle last sentence
		if i == len(sentences)-1 {
			chunkText := strings.TrimSpace(currentChunk.String())
			if utf8.RuneCountInString(chunkText) >= MinChunkSize {
				chunks = append(chunks, Chunk{
					Text:        chunkText,
					ChunkIndex:  chunkIndex,
					Source:      source,
					StartOffset: currentOffset,
					EndOffset:   currentOffset + len(chunkText),
					Metadata:    make(map[string]string),
				})
			}
		}
	}

	if len(chunks) == 0 {
		return nil, fmt.Errorf("no valid chunks created")
	}

	return chunks, nil
}

// ChunkLaTeXContent chunks LaTeX content with section awareness
func (c *Chunker) ChunkLaTeXContent(latexContent, source string) ([]Chunk, error) {
	// Extract sections from LaTeX
	sections := c.extractLaTeXSections(latexContent)

	if len(sections) == 0 {
		// Fallback to regular chunking
		return c.ChunkText(latexContent, source)
	}

	var allChunks []Chunk
	globalChunkIndex := 0

	for sectionName, sectionText := range sections {
		// Chunk each section separately
		chunks, err := c.ChunkText(sectionText, source)
		if err != nil {
			continue // Skip problematic sections
		}

		// Add section metadata and reindex
		for i := range chunks {
			chunks[i].Section = sectionName
			chunks[i].ChunkIndex = globalChunkIndex
			chunks[i].Metadata["section"] = sectionName
			globalChunkIndex++
		}

		allChunks = append(allChunks, chunks...)
	}

	if len(allChunks) == 0 {
		return c.ChunkText(latexContent, source)
	}

	return allChunks, nil
}

// cleanText removes excessive whitespace and special characters
func (c *Chunker) cleanText(text string) string {
	// Remove excessive newlines
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")

	// Remove excessive spaces
	text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")

	// Remove LaTeX commands that don't add meaning
	text = regexp.MustCompile(`\\(textbf|textit|emph)\{([^}]+)\}`).ReplaceAllString(text, "$2")

	return strings.TrimSpace(text)
}

// splitIntoSentences splits text into sentences
func (c *Chunker) splitIntoSentences(text string) []string {
	// Simple sentence boundary detection
	// Splits on: . ! ? followed by space or newline
	sentenceRegex := regexp.MustCompile(`([.!?])\s+`)

	parts := sentenceRegex.Split(text, -1)
	delimiters := sentenceRegex.FindAllString(text, -1)

	var sentences []string
	for i, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}

		sentence := strings.TrimSpace(part)
		if i < len(delimiters) {
			sentence += strings.TrimSpace(delimiters[i])
		}

		sentences = append(sentences, sentence+" ")
	}

	return sentences
}

// calculateOverlapSentences returns the last N sentences that fit within overlap size
func (c *Chunker) calculateOverlapSentences(sentences []string, overlapSize int) []string {
	if len(sentences) == 0 {
		return nil
	}

	var overlapSentences []string
	currentSize := 0

	// Work backwards from the end
	for i := len(sentences) - 1; i >= 0; i-- {
		sentenceLen := len(sentences[i])
		if currentSize+sentenceLen > overlapSize {
			break
		}
		overlapSentences = append([]string{sentences[i]}, overlapSentences...)
		currentSize += sentenceLen
	}

	return overlapSentences
}

// extractLaTeXSections extracts sections from LaTeX content
func (c *Chunker) extractLaTeXSections(latex string) map[string]string {
	sections := make(map[string]string)

	// Match \section{Title} and content until next section
	sectionRegex := regexp.MustCompile(`\\section\{([^}]+)\}`)
	matches := sectionRegex.FindAllStringSubmatchIndex(latex, -1)

	if len(matches) == 0 {
		return sections
	}

	for i, match := range matches {
		sectionName := latex[match[2]:match[3]]

		// Find content from end of \section{} to next \section or end
		contentStart := match[1]
		contentEnd := len(latex)

		if i < len(matches)-1 {
			contentEnd = matches[i+1][0]
		}

		sectionContent := latex[contentStart:contentEnd]
		sections[sectionName] = strings.TrimSpace(sectionContent)
	}

	return sections
}
