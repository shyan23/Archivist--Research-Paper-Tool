# Knowledge Graph Feature - Decision Log & User Requirements

**Date**: 2025-11-11
**Feature**: Semantic Search & Knowledge Graph
**Status**: Design Complete, Awaiting Implementation Approval

---

## Your Requirements (From plans.md)

### What You Want
```
Problem: Students can't discover related papers or understand relationships between concepts

Solution: "Archivist Explore Mode"
Features:
├── Build local knowledge graph from processed papers
│   ├── Extract: concepts, methodologies, datasets, metrics
│   ├── Link papers by: citations, shared concepts, similar architectures
│   └── Store in Neo4j or embedded graph DB
├── Semantic search using embeddings
│   ├── Use Gemini Embeddings API for paper chunks
│   ├── Vector search with FAISS/Qdrant
│   └── "Find papers similar to X methodology"
└── Interactive graph visualization
    ├── Terminal-based: using ASCII/Unicode graphs
    ├── Web-based: D3.js visualization server
    └── Show: paper clusters, concept evolution, citation networks

Commands:
archivist explore "attention mechanisms"
archivist graph --show-citations
archivist relate paper1.pdf paper2.pdf
archivist cluster --by-methodology
```

**Estimated Time**: 12-16 hours

---

## Your Answers to My Questions

### Question 1: Integration Strategy
**Your Answer**: "Integrate with existing RAG"

**What This Means**:
- ✅ Reuse existing FAISS vector store (internal/rag/faiss_store.go)
- ✅ Reuse Gemini embeddings client (internal/rag/embeddings.go)
- ✅ Extend existing chunker and indexer
- ✅ No duplication of embedding infrastructure
- ✅ Graph layer sits on top of existing RAG

**Impact on Plan**:
- Saved 3-4 hours (no need to rebuild vector search)
- Can leverage existing 768-dim embeddings
- Hybrid search combines existing retriever with new graph traversal

---

### Question 2: Graph Database Choice
**Your Answer**: "Neo4j (external service)"

**What This Means**:
- ✅ Use Neo4j Community Edition in Docker
- ✅ Powerful Cypher query language for complex graph queries
- ✅ Built-in graph algorithms (shortest path, centrality, clustering)
- ✅ Neo4j Browser for visualization (bonus debugging tool)
- ⚠️ Requires Docker setup (similar to Redis)

**Impact on Plan**:
- Added docker-compose.yml Neo4j service
- Created setup script: scripts/setup_graph.sh
- Can use advanced graph algorithms for recommendations
- Graph schema uses Neo4j best practices (indexes, constraints)

**Configuration Added**:
```yaml
graph:
  neo4j:
    uri: "bolt://localhost:7687"
    username: "neo4j"
    password: "password"
    database: "archivist"
```

---

### Question 3: Vector Search Technology
**Your Answer**: "FAISS (in-process)"

**What This Means**:
- ✅ Use existing FAISS implementation (no changes needed)
- ✅ No additional server to manage
- ✅ Fast local vector search
- ✅ Already integrated with your RAG system

**Impact on Plan**:
- Zero new dependencies for vector search
- Hybrid search uses existing VectorStoreInterface
- Search latency will be very fast (<100ms for vector component)

---

### Question 4: Target Scale
**Your Answer**: "Small (10-50 papers)"

**What This Means**:
- ✅ Simplified storage OK (no need for distributed systems)
- ✅ Can load entire graph in memory for visualization
- ✅ Simple graph layout algorithms sufficient
- ✅ No need for pagination/sharding

**Impact on Plan**:
- Terminal TUI can show all papers at once
- Graph layout uses simple force-directed algorithm
- No need for complex indexing strategies
- Background processing uses 2 workers (not 10+)

**Optimization Choices**:
- Show up to 15 nodes in TUI (with pagination if needed)
- Traversal depth limited to 2 hops
- No distributed graph processing needed

---

### Question 5: Visualization Priority
**Your Answer**: "Terminal first, web later"

**What This Means**:
- ✅ Phase 1: ASCII/Unicode graph in terminal using Bubble Tea
- ⏳ Phase 2 (Future): D3.js web visualization
- ✅ Matches CLI-first architecture
- ✅ Faster to implement

**Impact on Plan**:
- Terminal graph viewer in Phase 4 (2-3 hours)
- Web viewer marked as "Future Enhancement"
- TUI uses box-drawing characters and colors
- Navigation with arrow keys (fits existing TUI patterns)

**Terminal Design**:
```
┌────────────────────────────────────────┐
│     Knowledge Graph: Transformers      │
├────────────────────────────────────────┤
│                                        │
│      [Attention Is All You Need]       │
│                  │                     │
│         ┌────────┼────────┐           │
│         │        │        │           │
│     [BERT]   [GPT]   [ViT]            │
│                                        │
└────────────────────────────────────────┘
```

---

### Question 6: Citation Extraction Strategy
**Your Answer**: "in the reference section, there are list of papers, for best similarities we dont need all of them, but the ones mentioned throughout the papers, connect to those links first. keep an option of the Manual Citation also"

**What This Means**:
- ✅ **Priority 1**: Extract in-text citations (papers mentioned in the main text)
- ✅ **Priority 2**: Parse references section
- ✅ **Smart filtering**: Only link citations that are actually discussed, not just listed
- ✅ **Manual override**: Support YAML files for manual citation mapping

**Impact on Plan**:
- Citation extractor uses two-pass approach:
  1. Extract references section (all papers)
  2. Use Gemini to find in-text mentions with importance rating
- Only high/medium importance citations create graph edges
- Manual YAML overrides win over automatic extraction

**Implementation**:
```go
type InTextCitation struct {
    ReferenceIndex int
    Context        string // Surrounding text
    Importance     string // "high", "medium", "low"
}

// Gemini Prompt:
"Find papers cited in this text. For each citation, rate its
importance based on how much the paper discusses it:
- high: core methodology or major influence
- medium: supporting work or comparison
- low: passing mention in related work"
```

**Manual Override Format**:
```yaml
# papers/attention_is_all_you_need_citations.yaml
citations:
  - target: "neural_machine_translation"
    relation: "extends"
    importance: "high"
    note: "Core inspiration for attention mechanism"
```

---

### Question 7: Graph Building Timing
**Your Answer**: "keep parallel processing, the graph will be updated in the background, it must not bottleneck the paper processing, latex report and all."

**What This Means**:
- ✅ Graph building runs asynchronously (separate goroutines)
- ✅ Paper processing is never blocked
- ✅ If graph building fails, paper processing still succeeds
- ✅ Background worker pool for graph updates

**Impact on Plan**:
- Added GraphUpdateWorker with separate job queue
- Graph jobs submitted after LaTeX compilation succeeds
- Uses worker pool pattern (like paper processing)
- Configuration: `max_graph_workers: 2` (doesn't steal from paper workers)

**Architecture**:
```
Paper Processing Pipeline:
PDF → LaTeX → Compile → Cache → [DONE]
                         ↓
                   [Async Branch]
                         ↓
               Extract Citations
                         ↓
            Queue Graph Update Job
                         ↓
          Background Worker Processes
                         ↓
               Update Neo4j Graph
```

**Error Handling**:
- If Neo4j is down: Log warning, continue processing
- If citation extraction fails: Log warning, continue
- Graph updates are "best effort" - never block main pipeline

---

### Question 8: Search Mechanism
**Your Answer**: "Hybrid ranking" (selected all options: vector, graph, keyword)

**What This Means**:
- ✅ Combine multiple search signals for best results
- ✅ Vector search (semantic similarity via embeddings)
- ✅ Graph traversal (follow citation/concept links)
- ✅ Keyword search (traditional text matching)
- ✅ Weighted ranking algorithm

**Impact on Plan**:
- Implemented hybrid SearchEngine with 3 search modes
- Weighted scoring: Vector (50%), Graph (30%), Keyword (20%)
- Deduplication and reranking of combined results
- Boosts for cited papers and frequently mentioned concepts

**Scoring Formula**:
```go
hybridScore = (vectorScore * 0.5) +
              (graphScore * 0.3) +
              (keywordScore * 0.2)

// Boost citations
if paper.IsCited {
    hybridScore *= 1.2
}

// Boost frequently discussed papers
if paper.InTextMentions > 3 {
    hybridScore *= 1.1
}
```

**User Control**:
```bash
# Use all signals (default)
archivist explore "attention mechanisms"

# Only semantic
archivist explore "attention" --mode semantic

# Only graph traversal
archivist explore "attention" --mode graph --depth 2

# Only keywords
archivist explore "attention" --mode keyword
```

---

## Complete Feature Design Based on Your Answers

### Architecture Summary

```
┌─────────────────────────────────────────────────────────────┐
│                    Archivist Explore Mode                    │
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐ │
│  │ Vector Search│    │ Graph Search │    │   Terminal   │ │
│  │   (FAISS)    │    │   (Neo4j)    │    │  Visualizer  │ │
│  │  [EXISTING]  │    │    [NEW]     │    │    [NEW]     │ │
│  └──────────────┘    └──────────────┘    └──────────────┘ │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │          Hybrid Search Engine [NEW]                  │  │
│  │  Combines: Vector (50%) + Graph (30%) + Keyword (20%)│  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │        Citation Extractor [NEW]                      │  │
│  │  • Parse references section                          │  │
│  │  • Extract in-text mentions (Gemini)                 │  │
│  │  • Rate importance (high/medium/low)                 │  │
│  │  • Manual YAML overrides                             │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │     Background Graph Builder [NEW]                   │  │
│  │  • Non-blocking async processing                     │  │
│  │  • 2 worker pool (separate from paper workers)       │  │
│  │  • Updates Neo4j graph                               │  │
│  │  • Never blocks paper processing                     │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              ↓
    ┌────────────────────────────────────────────────┐
    │      Existing RAG Infrastructure               │
    │  • Chunker • Embeddings • Indexer • FAISS     │
    └────────────────────────────────────────────────┘
```

### New Commands

#### 1. `archivist explore <query>`
**What it does**: Hybrid search across papers

**Your input influence**:
- Uses hybrid ranking (all 3 signals)
- Respects small scale (fast, shows all results)
- Integrates with existing RAG

**Example**:
```bash
$ archivist explore "attention mechanisms"

Found 5 papers matching "attention mechanisms":

1. ★★★★★ Attention Is All You Need (2017)
   Match: semantic (0.92), graph (0.85), keyword (0.78)
   Concepts: attention, transformer, encoder-decoder
   Citations: 8 outgoing, 12 incoming

2. ★★★★☆ BERT (2018)
   Match: semantic (0.89), graph (0.92), keyword (0.45)
   Concepts: attention, pre-training, bidirectional
   Cites: #1 (high importance)
```

#### 2. `archivist graph [options]`
**What it does**: Terminal graph visualization

**Your input influence**:
- Terminal-first approach
- ASCII art (works for 10-50 papers)
- Shows citations prominently

**Example**:
```bash
$ archivist graph --show-citations

┌────────────────────────────────────────────────┐
│      Knowledge Graph: Citation Network        │
├────────────────────────────────────────────────┤
│                                                │
│         [Attention Is All You Need]            │
│                       │                        │
│          ┌────────────┼────────────┐          │
│          │            │            │          │
│      [BERT]        [GPT]     [Vision ViT]     │
│          │                          │          │
│    [RoBERTa]                    [DETR]        │
│                                                │
├────────────────────────────────────────────────┤
│ [↑↓←→] Navigate | [Enter] Details | [Q] Quit │
└────────────────────────────────────────────────┘
```

#### 3. `archivist relate <paper1> <paper2>`
**What it does**: Analyze relationship between papers

**Your input influence**:
- Shows citation links (your priority)
- Shows shared concepts
- Uses graph + vector similarity

**Example**:
```bash
$ archivist relate attention_is_all_you_need.pdf bert.pdf

Relationship Analysis:
├─ Direct Citation: ✓ Yes (BERT cites Attention, high importance)
├─ Semantic Similarity: 0.87 (very high)
├─ Graph Distance: 1 hop
├─ Shared Concepts (5):
│  ├─ attention mechanism (mentioned in both extensively)
│  ├─ transformer architecture
│  ├─ self-attention
│  └─ positional encoding
└─ Citation Context:
   "We build on the Transformer architecture [1] and apply it to..."
```

#### 4. `archivist cluster --by-methodology`
**What it does**: Group papers by attributes

**Example**:
```bash
$ archivist cluster --by-methodology

Clustering 10 papers by methodology...

Cluster 1: Transformer-based (6 papers)
├─ Attention Is All You Need
├─ BERT
├─ GPT-2
├─ Vision Transformers
├─ DETR
└─ T5

Cluster 2: Convolutional (3 papers)
├─ ResNet
├─ VGG
└─ AlexNet
```

---

## Implementation Phases (Your Answers Applied)

### Phase 1: Foundation (4-5 hours)
**Based on**: Neo4j choice, integration with existing RAG

**Tasks**:
1. ✅ Set up Neo4j in Docker (from your answer)
2. ✅ Implement CitationExtractor with in-text priority (from your answer)
3. ✅ Add manual YAML override support (from your answer)
4. ✅ Create GraphBuilder with Neo4j driver
5. ✅ Initialize graph schema

**Deliverable**: Citations extractable, Neo4j connected

---

### Phase 2: Graph Building (3-4 hours)
**Based on**: Background processing, non-blocking

**Tasks**:
1. ✅ Create background worker pool (2 workers, from your answer)
2. ✅ Integrate with paper processing pipeline (non-blocking, from your answer)
3. ✅ Add `archivist index --rebuild-graph` command
4. ✅ Implement concept extraction
5. ✅ Create semantic similarity links

**Deliverable**: Graph builds in background during processing

---

### Phase 3: Search Engine (3-4 hours)
**Based on**: Hybrid ranking (all signals)

**Tasks**:
1. ✅ Implement SearchEngine with hybrid algorithm (from your answer)
2. ✅ Add weighting: Vector (50%), Graph (30%), Keyword (20%) (from your answer)
3. ✅ Create `archivist explore` command
4. ✅ Create `archivist relate` command
5. ✅ Optimize for small scale (10-50 papers, from your answer)

**Deliverable**: Explore command functional, <2s latency

---

### Phase 4: Visualization (2-3 hours)
**Based on**: Terminal-first approach

**Tasks**:
1. ✅ Implement ASCII graph layout (from your answer)
2. ✅ Create TUI graph view with Bubble Tea
3. ✅ Add navigation (arrow keys)
4. ✅ Optimize for 10-50 papers display (from your answer)

**Deliverable**: Interactive terminal graph visualization

---

## Files to Create

Based on all your answers:

```
internal/
├── citation/              [NEW - from your citation priority]
│   ├── extractor.go      # In-text + references extraction
│   ├── parser.go         # LaTeX parsing
│   └── manual.go         # YAML overrides
│
├── graph/                 [NEW - from Neo4j choice]
│   ├── builder.go        # Neo4j graph construction
│   ├── search.go         # Hybrid search (from your ranking choice)
│   ├── query.go          # Cypher queries
│   └── worker.go         # Background processing (from your answer)
│
├── tui/                   [EXTEND - from terminal-first]
│   ├── graph_view.go     # ASCII visualization
│   └── graph_handlers.go
│
└── explorer/              [NEW]
    └── explorer.go       # Orchestration

cmd/main/commands/         [NEW]
├── explore.go            # archivist explore
├── graph.go              # archivist graph
├── relate.go             # archivist relate
└── cluster.go            # archivist cluster

papers/                    [NEW - from manual citation answer]
└── <paper_name>_citations.yaml
```

---

## Configuration Changes

Based on your answers:

```yaml
# config/config.yaml

graph:
  enabled: true

  # From your Neo4j choice
  neo4j:
    uri: "bolt://localhost:7687"
    username: "neo4j"
    password: "password"
    database: "archivist"

  # From your background processing answer
  async_building: true
  max_graph_workers: 2        # Separate from paper workers

  # From your citation strategy answer
  citation_extraction:
    enabled: true
    prioritize_in_text: true  # Your requirement
    confidence_threshold: 0.7
    importance_filter: ["high", "medium"]  # Skip "low"

  # From your hybrid ranking answer
  search:
    default_top_k: 10
    vector_weight: 0.5         # 50%
    graph_weight: 0.3          # 30%
    keyword_weight: 0.2        # 20%
    traversal_depth: 2         # For graph search

  # From your scale answer (10-50 papers)
  optimization:
    max_papers_in_memory: 50
    cache_graph_layout: true
    precompute_similarities: true

# From your terminal-first answer
visualization:
  terminal:
    enabled: true
    max_nodes_displayed: 15
    layout_algorithm: "force_directed"
  web:
    enabled: false  # Future enhancement
```

---

## Docker Compose Changes

From your Neo4j choice:

```yaml
# docker-compose.yml

services:
  neo4j:                    # NEW SERVICE
    image: neo4j:5.13-community
    container_name: archivist-neo4j
    ports:
      - "7474:7474"         # Browser UI
      - "7687:7687"         # Bolt protocol
    environment:
      - NEO4J_AUTH=neo4j/password
    volumes:
      - neo4j_data:/data
    networks:
      - archivist-network

  redis:                    # EXISTING
    # ... your current Redis config ...

volumes:
  neo4j_data:               # NEW VOLUME
  redis_data:               # EXISTING
```

---

## Summary: How Your Answers Shaped the Plan

| Your Answer | Impact on Design |
|-------------|-----------------|
| **Integrate with RAG** | Reused FAISS, embeddings, chunker (saved 3-4 hours) |
| **Neo4j** | Added Docker service, powerful Cypher queries, graph algorithms |
| **FAISS** | No new vector DB dependencies, fast local search |
| **10-50 papers** | Simplified layout, no distributed systems, can show all in TUI |
| **Terminal first** | ASCII graph in Phase 4, web viewer deferred |
| **In-text citations** | Two-pass extraction: references + in-text with importance rating |
| **Manual override** | YAML citation files in papers/ directory |
| **Background processing** | 2-worker pool, non-blocking, never blocks paper pipeline |
| **Hybrid ranking** | Combined vector (50%) + graph (30%) + keyword (20%) |

---

## What You Need to Confirm

Before I start implementing:

### 1. Citation Matching Strategy
When a paper cites "Attention Is All You Need", how do we find it in your library?

**Options**:
- a) Fuzzy title matching (85%+ similarity) ✅ **RECOMMENDED**
- b) Manual mapping only
- c) Fuzzy + manual fallback ✅ **BEST**

**Your decision**: ____________

---

### 2. Manual Citation YAML Format
Is this structure OK?

```yaml
# papers/attention_is_all_you_need_citations.yaml
paper: attention_is_all_you_need
citations:
  - target_title: "Neural Machine Translation by Jointly Learning to Align and Translate"
    target_file: "neural_machine_translation.pdf"  # Optional: direct mapping
    relation: "cites"
    importance: "high"
    context: "Core inspiration for attention mechanism"

  - target_title: "Sequence to Sequence Learning"
    relation: "extends"
    importance: "medium"
```

**Your approval**: ✅ / Changes needed: ____________

---

### 3. Error Handling: Neo4j Down
If Neo4j is unavailable during paper processing:

**Options**:
- a) Fail entire processing
- b) Skip graph building, log warning ✅ **RECOMMENDED**
- c) Queue for later retry

**Your decision**: ____________

---

### 4. Concept Extraction Timing
Extract concepts (methodologies, datasets, etc.):

**Options**:
- a) During initial LaTeX analysis (extend current prompt) ✅ **RECOMMENDED** (fewer API calls)
- b) Separate extraction pass after processing

**Your decision**: ____________

---

### 5. Graph Persistence
Should the graph persist between application restarts?

**Options**:
- a) Yes, Neo4j persists everything ✅ **RECOMMENDED**
- b) Rebuild graph on each startup

**Your decision**: ____________

---

## Ready to Implement?

✅ All design decisions documented
✅ Your answers integrated into architecture
✅ File structure defined
✅ Configuration specified
✅ Docker setup ready

**Total Estimated Time**: 12-16 hours

**Next Step**: Confirm the 5 questions above, then I start Phase 1!

---

## Quick Reference: What Gets Built

### Week 1 (Phase 1-2): 7-9 hours
- Neo4j setup and connection
- Citation extraction (references + in-text)
- Manual YAML override system
- Background graph builder
- Integration with paper processing

**Deliverable**: Papers processed with citations extracted, graph builds in background

---

### Week 2 (Phase 3-4): 5-7 hours
- Hybrid search engine
- CLI commands: explore, relate, cluster, graph
- Terminal graph visualization
- Testing and polish

**Deliverable**: Full explore mode functional with visualization

---

**Status**: ⏸️ Waiting for confirmation on 5 questions before starting implementation.
