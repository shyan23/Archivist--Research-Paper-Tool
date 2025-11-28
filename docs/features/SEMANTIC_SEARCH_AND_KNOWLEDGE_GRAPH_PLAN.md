# Semantic Search & Knowledge Graph Implementation Plan

**Feature**: Archivist Explore Mode
**Target**: 10-50 papers (small-scale research library)
**Integration**: Extends existing RAG infrastructure
**Timeline**: 12-16 hours implementation
**Status**: Detailed Design Phase

---

## Table of Contents
1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Component Breakdown](#component-breakdown)
4. [Data Models](#data-models)
5. [Implementation Phases](#implementation-phases)
6. [API Design](#api-design)
7. [Integration Points](#integration-points)
8. [Dependencies & Setup](#dependencies--setup)
9. [Testing Strategy](#testing-strategy)
10. [Open Questions](#open-questions)

---

## Executive Summary

### What We're Building
A **hybrid semantic search and knowledge graph system** that allows students to:
- Discover related papers through semantic similarity
- Navigate concept relationships via graph traversal
- Visualize paper connections in terminal (TUI)
- Track citations and concept evolution

### Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Graph DB** | Neo4j (Docker) | Powerful Cypher queries, graph algorithms, visualization-ready |
| **Vector Search** | FAISS (existing) | Already integrated, in-process, no new dependencies |
| **Embeddings** | Gemini API (existing) | Already used for RAG, 768-dim vectors |
| **Integration** | Extend RAG system | Reuse embeddings, chunker, indexer |
| **Visualization** | Terminal TUI first | MVP approach, matches CLI architecture |
| **Citations** | Extract + Manual | Smart extraction from references + in-text, manual override |
| **Graph Building** | Background async | Non-blocking parallel processing |
| **Search** | Hybrid ranking | Vector similarity + graph traversal + keywords |

### Success Metrics
- ✅ Find related papers in <2 seconds
- ✅ Graph builds without blocking paper processing
- ✅ TUI graph visualization for 10-50 papers
- ✅ Extract 70%+ of in-text citation relationships
- ✅ Manual citation overrides work smoothly

---

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                     Archivist Explore Mode                        │
└──────────────────────────────────────────────────────────────────┘
                               │
       ┌───────────────────────┼───────────────────────┐
       │                       │                       │
┌──────▼────────┐    ┌────────▼────────┐    ┌────────▼────────┐
│  Vector Search│    │  Knowledge Graph│    │   TUI Viewer    │
│    (FAISS)    │    │     (Neo4j)     │    │   (BubbleTea)   │
└───────────────┘    └─────────────────┘    └─────────────────┘
       │                       │                       │
       │              ┌────────▼────────┐              │
       │              │  Graph Builder  │              │
       │              │   (Background)  │              │
       │              └─────────────────┘              │
       │                       │                       │
┌──────▼───────────────────────▼───────────────────────▼──────┐
│              Existing RAG Infrastructure                     │
│  ┌─────────┐  ┌──────────┐  ┌─────────┐  ┌──────────┐     │
│  │ Chunker │  │Embeddings│  │ Indexer │  │Retriever │     │
│  └─────────┘  └──────────┘  └─────────┘  └──────────┘     │
└──────────────────────────────────────────────────────────────┘
```

### Data Flow

#### 1. Paper Processing (Enhanced)
```
PDF → LaTeX Analysis → [EXISTING] → Chunks → Embeddings → FAISS
                    ↓
               [NEW] Citation Extraction
                    ↓
          Background Graph Builder
                    ↓
               Neo4j Graph
```

#### 2. Explore Query
```
User Query → Hybrid Search Engine
                    ├── Vector Search (FAISS) → Semantic Matches
                    ├── Graph Traversal (Neo4j) → Related Papers
                    └── Keyword Search → Text Matches
                              ↓
                    Ranking & Deduplication
                              ↓
                       TUI Display
```

---

## Component Breakdown

### 1. Citation Extractor (NEW)

**File**: `internal/citation/extractor.go`

**Purpose**: Extract citation relationships from processed papers

**Key Functions**:
```go
type CitationExtractor struct {
    geminiClient *analyzer.GeminiClient
}

// ExtractCitations analyzes paper content to find citations
func (ce *CitationExtractor) ExtractCitations(ctx context.Context, latexContent string) (*CitationData, error)

// ParseReferencesSection extracts formal reference list
func (ce *CitationExtractor) ParseReferencesSection(latexContent string) ([]Reference, error)

// ExtractInTextCitations finds cited papers in main text
func (ce *CitationExtractor) ExtractInTextCitations(ctx context.Context, latexContent string, references []Reference) ([]InTextCitation, error)

// LoadManualCitations reads user-provided citation metadata (YAML)
func (ce *CitationExtractor) LoadManualCitations(paperTitle string) (*ManualCitationData, error)
```

**Data Structures**:
```go
type Reference struct {
    Index       int      // [1], [2], etc.
    Authors     []string
    Title       string
    Year        int
    Venue       string
    RawText     string   // Full reference string
}

type InTextCitation struct {
    ReferenceIndex int
    Context        string // Surrounding text
    Importance     string // "high", "medium", "low" (based on context)
}

type CitationData struct {
    References      []Reference
    InTextCitations []InTextCitation
    ManualOverrides map[string]string // User-provided mappings
}
```

**Implementation Strategy**:
1. **Phase 1**: Parse references section using LaTeX regex patterns
   - Look for `\begin{thebibliography}` or `\section{References}`
   - Extract numbered/labeled references

2. **Phase 2**: Use Gemini to identify in-text citations
   - Prompt: "Find all paper citations in this text and rate their importance based on context"
   - Focus on: "as shown in [X]", "following [Y]", "inspired by [Z]"

3. **Phase 3**: Manual overrides via YAML
   ```yaml
   # papers/<paper_name>_citations.yaml
   citations:
     - source: "attention_is_all_you_need"
       target: "neural_machine_translation"
       relation: "extends"
       importance: "high"
   ```

**Estimation**: 3-4 hours

---

### 2. Knowledge Graph Builder (NEW)

**File**: `internal/graph/builder.go`

**Purpose**: Build and maintain Neo4j knowledge graph

**Key Functions**:
```go
type GraphBuilder struct {
    neo4jDriver neo4j.Driver
    config      *GraphConfig
}

// InitializeSchema creates graph schema (indexes, constraints)
func (gb *GraphBuilder) InitializeSchema(ctx context.Context) error

// AddPaper creates paper node with metadata
func (gb *GraphBuilder) AddPaper(ctx context.Context, paper *PaperNode) error

// AddCitations creates citation relationships
func (gb *GraphBuilder) AddCitations(ctx context.Context, paperTitle string, citations *CitationData) error

// AddConceptLinks creates concept-based relationships
func (gb *GraphBuilder) AddConceptLinks(ctx context.Context, paper1, paper2 string, concepts []string) error

// UpdateGraphAsync processes graph updates in background
func (gb *GraphBuilder) UpdateGraphAsync(ctx context.Context, job *GraphUpdateJob) error
```

**Neo4j Graph Schema**:
```cypher
// Node Types
(:Paper {
  title: string,
  pdf_path: string,
  processed_at: datetime,
  embedding_id: string,  // Link to FAISS
  methodologies: [string],
  datasets: [string],
  metrics: [string]
})

(:Concept {
  name: string,
  category: string,  // "methodology", "architecture", "dataset"
  frequency: int     // How many papers mention it
})

// Relationship Types
(:Paper)-[:CITES {importance: string, context: string}]->(:Paper)
(:Paper)-[:USES_CONCEPT {section: string}]->(:Concept)
(:Paper)-[:SIMILAR_TO {score: float, basis: string}]->(:Paper)
(:Concept)-[:RELATED_TO {strength: float}]->(:Concept)
```

**Background Processing**:
```go
// GraphUpdateJob represents async graph update task
type GraphUpdateJob struct {
    PaperTitle   string
    LatexContent string
    Citations    *CitationData
    Priority     int
}

// Worker pool pattern (similar to paper processing)
type GraphUpdateWorker struct {
    builder *GraphBuilder
    jobs    chan *GraphUpdateJob
    wg      sync.WaitGroup
}
```

**Estimation**: 4-5 hours

---

### 3. Hybrid Search Engine (NEW)

**File**: `internal/graph/search.go`

**Purpose**: Unified search interface combining vector, graph, and keyword search

**Key Functions**:
```go
type SearchEngine struct {
    vectorStore VectorStoreInterface  // Existing FAISS
    graphClient *GraphBuilder
    embedClient *EmbeddingClient      // Existing
}

// Search performs hybrid search with ranking
func (se *SearchEngine) Search(ctx context.Context, query string, opts *SearchOptions) ([]SearchResult, error)

// VectorSearch finds semantically similar papers
func (se *SearchEngine) VectorSearch(ctx context.Context, queryEmbedding []float32, topK int) ([]SearchResult, error)

// GraphSearch finds papers via graph traversal
func (se *SearchEngine) GraphSearch(ctx context.Context, query string, traversalDepth int) ([]SearchResult, error)

// KeywordSearch performs traditional text search
func (se *SearchEngine) KeywordSearch(ctx context.Context, query string) ([]SearchResult, error)

// RankResults combines and ranks results from multiple sources
func (se *SearchEngine) RankResults(vectorResults, graphResults, keywordResults []SearchResult) ([]SearchResult, error)
```

**Hybrid Ranking Algorithm**:
```go
// Score calculation
func calculateHybridScore(result SearchResult) float64 {
    score := 0.0

    // Vector similarity (0.5 weight)
    if result.VectorScore > 0 {
        score += result.VectorScore * 0.5
    }

    // Graph relevance (0.3 weight)
    if result.GraphScore > 0 {
        score += result.GraphScore * 0.3
    }

    // Keyword match (0.2 weight)
    if result.KeywordScore > 0 {
        score += result.KeywordScore * 0.2
    }

    // Boost for citations
    if result.IsCited {
        score *= 1.2
    }

    return score
}
```

**Search Options**:
```go
type SearchOptions struct {
    Query           string
    TopK            int
    IncludeVector   bool
    IncludeGraph    bool
    IncludeKeyword  bool
    TraversalDepth  int    // For graph search
    Filters         map[string]string
}
```

**Estimation**: 3-4 hours

---

### 4. Terminal Graph Visualizer (NEW)

**File**: `internal/tui/graph_view.go`

**Purpose**: ASCII/Unicode graph visualization in terminal

**Key Functions**:
```go
type GraphView struct {
    model       *GraphViewModel
    graphData   *GraphData
    selectedNode int
}

// RenderGraph displays graph using ASCII art
func (gv *GraphView) RenderGraph(width, height int) string

// RenderNodeDetails shows detailed info for selected node
func (gv *GraphView) RenderNodeDetails(node *PaperNode) string

// HandleNavigation processes keyboard input for graph navigation
func (gv *GraphView) HandleNavigation(key string) tea.Cmd
```

**Visualization Strategy**:
```
Terminal Graph Display (80x24 example)

┌────────────────────────────────────────────────────────────────┐
│                 Knowledge Graph: Transformers                  │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│                   [Attention Is All You Need]                  │
│                             │                                  │
│             ┌───────────────┼───────────────┐                 │
│             │               │               │                 │
│      [BERT]          [GPT]          [Vision Transformers]     │
│         │               │               │                     │
│   ┌─────┴─────┐   ┌────┴────┐      [DETR]                   │
│   │           │   │         │                                 │
│ [RoBERTa] [ALBERT] [GPT-2] [GPT-3]                           │
│                                                                │
├────────────────────────────────────────────────────────────────┤
│ Selected: Attention Is All You Need                            │
│ └─ Year: 2017 | Citations: 3 outgoing, 5 incoming            │
│ └─ Concepts: attention mechanism, encoder-decoder             │
│                                                                │
│ [↑↓←→] Navigate | [Enter] Details | [S] Search | [Q] Quit    │
└────────────────────────────────────────────────────────────────┘
```

**Implementation Approach**:
1. Use force-directed layout algorithm (simplified)
2. ASCII box drawing characters: `─│┌┐└┘├┤┬┴┼`
3. Color coding: citations (blue), concepts (green), similar (yellow)
4. Pagination for large graphs (10 nodes per page)

**Libraries**:
- `github.com/gizak/termui` - Terminal UI widgets
- Custom layout engine

**Estimation**: 3-4 hours

---

## Data Models

### File Structure
```
internal/
├── citation/
│   ├── extractor.go       # Citation extraction logic
│   ├── parser.go          # Reference section parsing
│   └── manual.go          # Manual citation loading
├── graph/
│   ├── builder.go         # Neo4j graph construction
│   ├── search.go          # Hybrid search engine
│   ├── query.go           # Cypher query builders
│   ├── models.go          # Graph data structures
│   └── worker.go          # Background graph updates
├── tui/
│   ├── graph_view.go      # Graph visualization
│   ├── graph_handlers.go  # Graph interaction handlers
│   └── graph_layout.go    # Layout algorithms
└── explorer/
    └── explorer.go        # Explore mode orchestrator

cmd/main/commands/
├── explore.go             # archivist explore command
├── graph.go               # archivist graph command
├── relate.go              # archivist relate command
└── cluster.go             # archivist cluster command

papers/
└── <paper_name>_citations.yaml  # Manual citation overrides
```

### Configuration Extensions

**config/config.yaml**:
```yaml
# Existing config...

# Knowledge Graph settings
graph:
  enabled: true
  neo4j:
    uri: "bolt://localhost:7687"
    username: "neo4j"
    password: "password"
    database: "archivist"

  # Background processing
  async_building: true
  max_graph_workers: 2

  # Citation extraction
  citation_extraction:
    enabled: true
    confidence_threshold: 0.7
    max_citations_per_paper: 50

  # Search settings
  search:
    default_top_k: 10
    vector_weight: 0.5
    graph_weight: 0.3
    keyword_weight: 0.2
    traversal_depth: 2

# Visualization settings
visualization:
  terminal:
    enabled: true
    max_nodes_displayed: 15
    layout_algorithm: "force_directed"
  web:
    enabled: false  # Future: D3.js web view
    port: 8080
```

---

## Implementation Phases

### Phase 1: Foundation (4-5 hours)
**Goal**: Set up Neo4j integration and citation extraction

**Tasks**:
1. Add Neo4j Go driver dependency
2. Implement `GraphBuilder` with basic schema
3. Implement `CitationExtractor` (regex + Gemini)
4. Add manual citation YAML support
5. Write unit tests for citation extraction

**Deliverables**:
- Neo4j connected and schema initialized
- Citations extractable from LaTeX
- Manual override system working

**Validation**:
```bash
# Test citation extraction
archivist extract-citations lib/attention_is_all_you_need.pdf

# Output: Found 8 references, 3 high-importance in-text citations
```

---

### Phase 2: Graph Building (3-4 hours)
**Goal**: Build graph from processed papers

**Tasks**:
1. Implement background graph update worker pool
2. Integrate graph building into paper processing pipeline
3. Add `archivist index --rebuild-graph` command
4. Implement concept extraction (reuse LaTeX analysis)
5. Create semantic similarity links (using existing embeddings)

**Deliverables**:
- Graph builds in background during paper processing
- Rebuild command for existing papers
- Concepts and methodologies extracted

**Validation**:
```bash
# Process papers with graph building
archivist process lib/ --enable-graph

# Check graph status
archivist graph status
# Output: 10 papers, 25 citations, 45 concept links
```

---

### Phase 3: Search Engine (3-4 hours)
**Goal**: Implement hybrid search

**Tasks**:
1. Implement `SearchEngine` with vector/graph/keyword search
2. Implement ranking algorithm
3. Add `archivist explore` command
4. Add `archivist relate` command
5. Test search quality on 10+ papers

**Deliverables**:
- Explore command functional
- Hybrid ranking working
- Sub-2-second search response

**Validation**:
```bash
# Search for papers
archivist explore "attention mechanisms"
# Output: 5 results ranked by relevance

# Find related papers
archivist relate paper1.pdf paper2.pdf
# Output: Similarity: 0.85, Shared concepts: attention, transformer
```

---

### Phase 4: Visualization (2-3 hours)
**Goal**: Terminal graph viewer

**Tasks**:
1. Implement ASCII graph layout algorithm
2. Create TUI graph view with Bubble Tea
3. Add navigation and interaction
4. Integrate into `archivist graph` command

**Deliverables**:
- Interactive graph visualization
- Navigation with arrow keys
- Node details on selection

**Validation**:
```bash
# View graph
archivist graph --show-citations
# Opens TUI with visual graph

# View specific cluster
archivist cluster --by-methodology
# Shows grouped papers by methodology
```

---

## API Design

### Command-Line Interface

#### 1. `archivist explore <query>`
Search papers using hybrid approach

```bash
# Basic search
archivist explore "attention mechanisms"

# Options
archivist explore "transformers" --top-k 10 --mode semantic
archivist explore "CNNs" --mode graph --depth 2
archivist explore "BERT" --mode keyword
archivist explore "vision" --filter year:2020-2023

# Output format
Found 5 papers matching "attention mechanisms":

1. Attention Is All You Need (2017) ★★★★★
   └─ Match: semantic (0.92), graph (0.85), keyword (0.78)
   └─ Concepts: attention, transformer, encoder-decoder
   └─ Citations: 8 outgoing, 12 incoming

2. BERT: Pre-training of Deep Bidirectional Transformers (2018) ★★★★☆
   └─ Match: semantic (0.89), graph (0.92), keyword (0.45)
   └─ Concepts: attention, pre-training, bidirectional
   └─ Citations: 2 outgoing (cites #1), 5 incoming

...

Use 'archivist graph' to visualize relationships
```

#### 2. `archivist graph [options]`
Visualize knowledge graph

```bash
# Show full graph
archivist graph

# Filter options
archivist graph --show-citations
archivist graph --show-concepts
archivist graph --paper "Attention Is All You Need"
archivist graph --depth 2  # Show 2 hops from root

# Export
archivist graph --export graph.json
archivist graph --export graph.png  # Future: requires web mode
```

#### 3. `archivist relate <paper1> <paper2>`
Analyze relationship between two papers

```bash
archivist relate attention_is_all_you_need.pdf bert.pdf

# Output
Relationship Analysis:
├─ Direct Citation: Yes (BERT cites Attention Is All You Need)
├─ Semantic Similarity: 0.87 (very high)
├─ Shared Concepts (5):
│  ├─ attention mechanism
│  ├─ transformer architecture
│  ├─ self-attention
│  ├─ encoder-decoder
│  └─ positional encoding
├─ Common Citations (3):
│  ├─ Neural Machine Translation...
│  ├─ Sequence to Sequence...
│  └─ Convolutional Sequence...
└─ Graph Distance: 1 hop

Conclusion: BERT is a direct extension of Attention Is All You Need,
applying the transformer architecture to pre-training tasks.
```

#### 4. `archivist cluster [--by-methodology|--by-dataset|--by-year]`
Cluster papers by attributes

```bash
archivist cluster --by-methodology

# Output
Clustering papers by methodology...

Cluster 1: Transformer-based (6 papers)
├─ Attention Is All You Need
├─ BERT
├─ GPT
├─ GPT-2
├─ Vision Transformers
└─ DETR

Cluster 2: Convolutional (3 papers)
├─ ResNet
├─ AlexNet
└─ VGG

Cluster 3: Recurrent (2 papers)
├─ LSTM
└─ GRU

Use 'archivist graph --cluster 1' to visualize cluster
```

#### 5. `archivist index --rebuild-graph`
Rebuild knowledge graph from existing papers

```bash
archivist index --rebuild-graph

# Output
Rebuilding knowledge graph...
├─ Extracting citations from 10 papers...
├─ Building paper nodes...
├─ Creating citation relationships...
├─ Extracting concepts...
├─ Computing semantic similarities...
└─ Graph rebuilt: 10 nodes, 45 edges (8.2s)
```

---

## Integration Points

### 1. Worker Pool Integration

**File**: `internal/worker/pool.go`

```go
// Modify processJob to include graph building

func (wp *WorkerPool) processJob(ctx context.Context, job *ProcessingJob) *ProcessingResult {
    // ... existing code ...

    // After Step 5 (caching), add graph building
    if wp.graphBuilder != nil && wp.config.Graph.Enabled {
        // Extract citations
        citations, err := wp.citationExtractor.ExtractCitations(ctx, latexContent)
        if err != nil {
            log.Printf("  ⚠️  Citation extraction failed: %v", err)
        } else {
            // Submit graph update job (non-blocking)
            graphJob := &graph.UpdateJob{
                PaperTitle:   paperTitle,
                LatexContent: latexContent,
                Citations:    citations,
                PDFPath:      job.FilePath,
            }

            select {
            case wp.graphUpdateQueue <- graphJob:
                log.Printf("  📊 Graph update queued")
            default:
                log.Printf("  ⚠️  Graph queue full, skipping")
            }
        }
    }

    // ... rest of code ...
}
```

### 2. RAG System Reuse

**Existing Components to Leverage**:
```go
// Reuse embeddings
embedClient := rag.NewEmbeddingClient(apiKey)  // Already exists

// Reuse vector store
vectorStore := rag.NewFAISSVectorStore(indexPath)  // Already exists

// Reuse chunker
chunker := rag.NewChunker(chunkSize, overlap)  // Already exists
```

**New Graph-Enhanced Retriever**:
```go
// internal/graph/retriever.go
type GraphEnhancedRetriever struct {
    vectorRetriever *rag.Retriever  // Existing
    graphSearch     *SearchEngine   // New
}

func (ger *GraphEnhancedRetriever) RetrieveWithContext(
    ctx context.Context,
    query string,
    topK int,
) ([]rag.SearchResult, error) {
    // Use hybrid search instead of just vector search
    return ger.graphSearch.Search(ctx, query, &SearchOptions{
        TopK:           topK,
        IncludeVector:  true,
        IncludeGraph:   true,
        TraversalDepth: 2,
    })
}
```

### 3. Chat Integration

**File**: `internal/chat/chat_engine.go`

```go
// Enhance chat with graph context

func (ce *ChatEngine) Chat(ctx context.Context, query string) (string, error) {
    // Step 1: Hybrid search (NEW)
    results, err := ce.graphRetriever.RetrieveWithContext(ctx, query, 5)

    // Step 2: Add graph context to prompt
    graphContext := ce.buildGraphContext(results)

    // Step 3: Generate response with enhanced context
    prompt := fmt.Sprintf(`
You are a research assistant with access to a knowledge graph of papers.

User Query: %s

Retrieved Papers:
%s

Graph Context:
%s

Please provide a comprehensive answer using both semantic similarity
and citation relationships.
`, query, formatResults(results), graphContext)

    return ce.generateResponse(ctx, prompt)
}

func (ce *ChatEngine) buildGraphContext(results []SearchResult) string {
    // Include citation chains, related papers, shared concepts
    // from knowledge graph
}
```

---

## Dependencies & Setup

### Go Dependencies

Add to `go.mod`:
```go
require (
    github.com/neo4j/neo4j-go-driver/v5 v5.14.0  // Neo4j driver
    github.com/gizak/termui/v3 v3.1.0            // Terminal UI (optional)
    gopkg.in/yaml.v3 v3.0.1                      // Manual citations (already have)
)
```

### Docker Setup

**docker-compose.yml** (add Neo4j service):
```yaml
version: '3.8'

services:
  # Existing Redis service...

  neo4j:
    image: neo4j:5.13-community
    container_name: archivist-neo4j
    ports:
      - "7474:7474"  # HTTP
      - "7687:7687"  # Bolt
    environment:
      - NEO4J_AUTH=neo4j/password
      - NEO4J_PLUGINS=["graph-data-science"]
    volumes:
      - neo4j_data:/data
      - neo4j_logs:/logs
    networks:
      - archivist-network

volumes:
  redis_data:
  neo4j_data:
  neo4j_logs:

networks:
  archivist-network:
    driver: bridge
```

### Setup Script

**scripts/setup_graph.sh**:
```bash
#!/bin/bash

echo "🚀 Setting up Archivist Knowledge Graph..."

# 1. Start Neo4j
echo "Starting Neo4j..."
docker-compose up -d neo4j

# 2. Wait for Neo4j to be ready
echo "Waiting for Neo4j..."
until curl -s http://localhost:7474 > /dev/null; do
    sleep 2
done

# 3. Initialize schema
echo "Initializing graph schema..."
go run cmd/graph-init/main.go

# 4. Create manual citations directory
mkdir -p papers

echo "✅ Knowledge graph setup complete!"
echo ""
echo "Access Neo4j Browser: http://localhost:7474"
echo "Username: neo4j, Password: password"
```

---

## Testing Strategy

### Unit Tests

**Citation Extraction**:
```go
// internal/citation/extractor_test.go
func TestExtractReferences(t *testing.T) {
    extractor := NewCitationExtractor(mockGeminiClient)

    latexContent := `
\section{References}
[1] Vaswani et al. Attention Is All You Need. NeurIPS 2017.
[2] Devlin et al. BERT. arXiv 2018.
    `

    refs, err := extractor.ParseReferencesSection(latexContent)
    assert.NoError(t, err)
    assert.Len(t, refs, 2)
    assert.Equal(t, "Attention Is All You Need", refs[0].Title)
}
```

**Graph Building**:
```go
// internal/graph/builder_test.go
func TestAddPaperNode(t *testing.T) {
    builder := NewGraphBuilder(mockNeo4jDriver, config)

    paper := &PaperNode{
        Title: "Test Paper",
        Methodologies: []string{"transformer", "attention"},
    }

    err := builder.AddPaper(context.Background(), paper)
    assert.NoError(t, err)

    // Verify in Neo4j
    exists := builder.PaperExists(context.Background(), "Test Paper")
    assert.True(t, exists)
}
```

**Hybrid Search**:
```go
// internal/graph/search_test.go
func TestHybridSearch(t *testing.T) {
    engine := NewSearchEngine(mockVectorStore, mockGraphClient, mockEmbedClient)

    results, err := engine.Search(context.Background(), "attention", &SearchOptions{
        TopK: 5,
        IncludeVector: true,
        IncludeGraph: true,
    })

    assert.NoError(t, err)
    assert.Len(t, results, 5)

    // Verify ranking
    assert.Greater(t, results[0].HybridScore, results[1].HybridScore)
}
```

### Integration Tests

**End-to-End Workflow**:
```go
func TestEndToEndGraphBuilding(t *testing.T) {
    // 1. Process paper
    ProcessPaper("test_paper.pdf")

    // 2. Verify graph updated
    time.Sleep(2 * time.Second)  // Wait for async processing

    graph := GetGraphClient()
    paper, err := graph.GetPaper("test_paper")
    assert.NoError(t, err)
    assert.NotNil(t, paper)

    // 3. Search for paper
    results, err := SearchEngine.Explore("test_paper")
    assert.NoError(t, err)
    assert.Contains(t, results[0].Title, "test_paper")
}
```

### Manual Testing

**Test Cases**:
1. Process 10 papers and verify graph builds
2. Search for "transformers" and verify results
3. View graph in TUI and navigate
4. Relate two papers and verify relationship
5. Cluster papers and verify groupings
6. Add manual citation and verify override

**Performance Tests**:
- Search latency < 2 seconds
- Graph build doesn't block paper processing
- Memory usage reasonable for 50 papers

---

## Open Questions

### Questions for You (Need Answers Before Implementation)

1. **Citation Matching**:
   - Q: How should we match extracted citations to processed papers?
   - Options:
     a) Fuzzy title matching (85%+ similarity)
     b) Manual mapping via YAML
     c) Both (try fuzzy, fall back to manual)
   - **My Recommendation**: Option c

2. **Graph Persistence**:
   - Q: Should we persist graph state between runs?
   - If yes, need to handle: schema migrations, incremental updates
   - **My Recommendation**: Yes, use Neo4j persistence + versioned schema

3. **Concept Extraction**:
   - Q: Should we extract concepts during initial analysis or separately?
   - Options:
     a) Extend analysis prompt to extract concepts
     b) Separate extraction pass
   - **My Recommendation**: Option a (less API calls)

4. **Error Handling**:
   - Q: If Neo4j is down, should we:
     a) Fail paper processing?
     b) Skip graph building, log warning?
     c) Queue for later retry?
   - **My Recommendation**: Option b or c

5. **Manual Citation Format**:
   - Q: Preferred YAML structure for manual citations?
   - Proposed:
     ```yaml
     paper: attention_is_all_you_need
     citations:
       - title: "Neural Machine Translation by Jointly Learning to Align and Translate"
         relation: "cites"
         importance: "high"
     ```
   - **Need Confirmation**: OK?

### Technical Uncertainties

1. **Neo4j Performance**:
   - Unknown: Query performance with 50+ papers
   - Mitigation: Will optimize after initial implementation

2. **Terminal Limitations**:
   - Unknown: Best layout algorithm for terminal constraints
   - Mitigation: Start with simple hierarchical layout, iterate

3. **Citation Accuracy**:
   - Unknown: Gemini's accuracy for citation extraction
   - Mitigation: Extensive testing + manual override system

---

## Timeline & Milestones

### Week 1: Foundation
- **Day 1-2**: Neo4j setup, citation extraction
- **Day 3**: Graph builder implementation
- **Milestone**: Citations extracted, basic graph builds

### Week 2: Search & Visualization
- **Day 4-5**: Hybrid search engine
- **Day 6**: Terminal graph viewer
- **Day 7**: Integration testing, polish
- **Milestone**: Full explore mode functional

### Estimated Total: 12-16 hours

---

## Next Steps

1. **Get Your Feedback** on open questions
2. **Review** this plan, adjust priorities
3. **Approve** dependencies and architecture
4. **Start Phase 1**: Citation extraction + Neo4j integration

Once you confirm the answers to the open questions, I'll start implementing Phase 1!

---

## Appendix: Example Queries

### Neo4j Cypher Queries

**Find papers citing a given paper**:
```cypher
MATCH (p1:Paper {title: $title})<-[:CITES]-(p2:Paper)
RETURN p2.title, p2.year
ORDER BY p2.year DESC
```

**Find papers using specific concept**:
```cypher
MATCH (c:Concept {name: $concept})<-[:USES_CONCEPT]-(p:Paper)
RETURN p.title, p.methodologies
```

**Find similar papers (2-hop traversal)**:
```cypher
MATCH (p1:Paper {title: $title})-[:SIMILAR_TO|CITES*1..2]-(p2:Paper)
WHERE p1 <> p2
RETURN DISTINCT p2.title, p2.year
LIMIT 10
```

**Cluster by methodology**:
```cypher
MATCH (p:Paper)
UNWIND p.methodologies AS method
WITH method, collect(p.title) AS papers
WHERE size(papers) > 1
RETURN method, papers
ORDER BY size(papers) DESC
```

---

**Status**: Awaiting feedback on open questions before implementation starts.
