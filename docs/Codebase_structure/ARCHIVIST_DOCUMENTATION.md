# ARCHIVIST COMPREHENSIVE DOCUMENTATION

## Table of Contents
1. [Overview](#overview)
2. [Project Structure](#project-structure)
3. [Command-Line Interface (CLI)](#command-line-interface)
4. [Core Application Components](#core-application-components)
5. [Paper Processing Pipeline](#paper-processing-pipeline)
6. [AI Integration (Gemini)](#ai-integration)
7. [Knowledge Graph System](#knowledge-graph-system)
8. [RAG & Chat System](#rag--chat-system)
9. [Search Engine Microservice](#search-engine-microservice)
10. [Caching System](#caching-system)
11. [Terminal UI](#terminal-ui)
12. [Infrastructure & Deployment](#infrastructure--deployment)
13. [Architecture Diagrams](#architecture-diagrams)
14. [Complete Workflow](#complete-workflow)

## Overview

Archivist is a sophisticated research paper processing system that converts AI/ML research papers into comprehensive, student-friendly LaTeX reports using Gemini AI. The system provides advanced features including knowledge graph creation, semantic search, and interactive Q&A capabilities for academic papers.

### Key Features
- Multi-stage AI paper analysis with Gemini
- LaTeX report generation with academic formatting
- Knowledge graph for paper relationships
- RAG-powered chat with papers
- Multi-source academic paper search
- Redis caching for optimization
- Terminal UI with menu navigation

### Technology Stack
- **Backend**: Go (Golang)
- **AI**: Google Gemini API (Vision, Text, Embeddings)
- **Databases**: Redis (cache), Neo4j (graph), Qdrant/FAISS (vector)
- **Web Framework**: FastAPI (Python search service)
- **UI**: Bubble Tea (terminal UI)
- **Infrastructure**: Docker, Docker Compose

## Project Structure

```
├── cmd/
│   └── main/
│       ├── main.go
│       └── commands/
│           ├── root.go          # Root command and CLI setup
│           ├── process.go       # Paper processing command
│           ├── list.go          # List papers command
│           ├── search.go        # Search papers command
│           ├── cache.go         # Cache management command
│           ├── chat.go          # Chat with papers command
│           ├── models.go        # Gemini models command
│           ├── index.go         # Index papers for chat command
│           └── other commands...
├── config/
│   └── config.yaml             # Configuration file
├── internal/
│   ├── analyzer/              # AI paper analysis
│   │   ├── analyzer.go        # Main analyzer logic
│   │   ├── gemini_client.go   # Gemini API client
│   │   └── prompts.go         # AI prompts
│   ├── app/                   # Application configuration
│   │   ├── config.go          # Configuration parsing
│   │   └── logger.go          # Logging setup
│   ├── cache/                 # Caching system
│   │   └── redis_cache.go     # Redis caching implementation
│   ├── chat/                  # Chat system
│   │   └── chat_engine.go     # Chat engine logic
│   ├── compiler/              # LaTeX compilation
│   │   └── latex_compiler.go  # LaTeX to PDF compilation
│   ├── generator/             # LaTeX generation
│   │   └── latex_generator.go # LaTeX file creation
│   ├── graph/                 # Knowledge graph
│   │   ├── builder.go         # Graph builder base
│   │   ├── enhanced_neo4j_builder.go # Enhanced Neo4j operations
│   │   ├── citation_extractor.go # Citation extraction
│   │   ├── enhanced_builder.go # Enhanced builder logic
│   │   ├── enhanced_models.go # Graph models
│   │   ├── hybrid_search.go   # Hybrid graph/vector search
│   │   ├── models.go          # Graph models
│   │   └── various graph components...
│   ├── parser/                # PDF parsing (uses Gemini vision)
│   ├── profiler/              # Performance profiling
│   │   └── profiler.go        # CPU/Memory profiling
│   ├── python_rag/            # Python RAG components
│   ├── rag/                   # RAG system
│   │   ├── chunker.go         # Text chunking
│   │   ├── embeddings.go      # Embedding client
│   │   ├── faiss_store.go     # FAISS vector store
│   │   ├── indexer.go         # Index management
│   │   ├── retriever.go       # Context retrieval
│   │   ├── vector_store_interface.go # Vector store interface
│   │   ├── vector_store.go    # Vector store logic
│   │   └── direct_indexer.go  # Direct indexing
│   ├── search/                # Search client
│   │   └── client.go          # Python search client
│   ├── tui/                   # Terminal UI
│   │   ├── chat.go            # Chat UI
│   │   ├── chat_handlers.go   # Chat handlers
│   │   ├── chat_indexing.go   # Chat indexing
│   │   ├── command_palette.go # Command palette
│   │   ├── handlers.go        # UI handlers
│   │   ├── loaders.go         # Loading indicators
│   │   ├── model.go           # TUI model
│   │   ├── navigation.go      # Navigation logic
│   │   ├── search.go          # Search UI
│   │   ├── styles.go          # UI styling
│   │   ├── types.go           # TUI types
│   │   └── views.go           # UI views
│   ├── ui/                    # UI utilities
│   │   └── ui.go              # UI helper functions
│   ├── vectorstore/           # Vector store
│   ├── wizard/                # Setup wizard
│   └── worker/                # Processing workers
│       └── pool.go            # Worker pool logic
├── pkg/
│   └── fileutil/              # File utilities
│       ├── hash.go            # File hashing
│       └── hash_test.go       # Hash tests
├── services/                  # External services
│   └── search-engine/         # Python search service
├── scripts/                   # Helper scripts
├── tex_files/                 # Generated LaTeX files
├── reports/                   # Generated PDF reports
├── lib/                       # Input PDF library
└── various configuration and build files
```

## Command-Line Interface

### Root Command (`cmd/main/commands/root.go`)
Handles the main CLI structure and subcommands

```go
func NewRootCommand() *cobra.Command {
    rootCmd := &cobra.Command{
        Use:   "rph",
        Short: "Research Paper Helper - Convert research papers to student-friendly LaTeX reports",
        Long: `Research Paper Helper analyzes AI/ML research papers using Gemini AI
and generates comprehensive, student-friendly LaTeX reports with detailed
explanations of methodologies, breakthroughs, and results.`,
    }

    // Add all subcommands
    rootCmd.AddCommand(
        NewProcessCommand(),
        NewListCommand(),
        NewStatusCommand(),
        NewCleanCommand(),
        NewCheckCommand(),
        NewRunCommand(),
        NewModelsCommand(),
        NewCacheCommand(),
        NewConfigureCommand(),
        NewChatCommand(),
        NewIndexCommand(),
        NewSearchCommand(),
    )
    return rootCmd
}
```

### Process Command (`cmd/main/commands/process.go`)
Handles paper processing with parallel workers

**Functions:**
- `NewProcessCommand()` - Creates the process command
- `runProcess()` - Main processing logic
- `applyModeConfig()` - Applies processing mode configuration

**Key Features:**
- Parallel processing with configurable workers
- Interactive mode selection
- Dependency checking
- Redis caching integration
- RAG indexing option

### List Command (`cmd/main/commands/list.go`)
Displays input PDFs or generated reports

**Functions:**
- `NewListCommand()` - Creates the list command
- `runList()` - Main listing logic

### Search Command (`cmd/main/commands/search.go`)
Searches for academic papers across multiple sources

**Functions:**
- `NewSearchCommand()` - Creates the search command
- `runSearch()` - Main search logic
- `printSearchResult()` - Formats search results
- `handleDownload()` - Handles paper downloads
- `sanitizeFilename()` - Sanitizes downloaded file names
- `copyFile()` - Copies files
- `min()` - Helper function

### Cache Command (`cmd/main/commands/cache.go`)
Manages Redis analysis cache

**Functions:**
- `NewCacheCommand()` - Creates cache command with subcommands
- `newCacheClearCommand()` - Creates clear subcommand
- `newCacheStatsCommand()` - Creates stats subcommand
- `newCacheListCommand()` - Creates list subcommand
- `runCacheClear()` - Clears cache entries
- `runCacheStats()` - Shows cache statistics
- `runCacheList()` - Lists cached papers

### Chat Command (`cmd/main/commands/chat.go`)
Interactive Q&A with papers using RAG

**Functions:**
- `NewChatCommand()` - Creates chat command
- `runChat()` - Main chat logic
- `selectPapersForChat()` - Interactive paper selection
- `extractPaperTitle()` - Extracts paper title from path
- `findPDFFiles()` - Finds PDF files in directory

### Models Command (`cmd/main/commands/models.go`)
Lists available Gemini AI models

**Functions:**
- `NewModelsCommand()` - Creates models command
- `runModels()` - Lists available models

### Index Command (`cmd/main/commands/index.go`)
Indexes processed papers for chat functionality

**Functions:**
- `NewIndexCommand()` - Creates index command
- `runIndex()` - Main indexing logic

## Core Application Components

### App Configuration (`internal/app/config.go`)

**Structs:**
- `Config` - Main configuration structure
- `ProcessingConfig` - Processing settings
- `GeminiConfig` - Gemini AI settings
- `AgenticConfig` - Agentic workflow settings
- `StagesConfig` - Multi-stage analysis settings
- `StageConfig` - Individual stage settings
- `RetryConfig` - Retry logic settings
- `LatexConfig` - LaTeX compilation settings
- `LoggingConfig` - Logging configuration
- `CacheConfig` - Caching configuration
- `RedisConfig` - Redis connection settings
- `FAISSConfig` - FAISS settings
- `GraphConfig` - Graph database settings
- `Neo4jConfig` - Neo4j connection settings
- `CitationExtractionConfig` - Citation settings
- `SearchConfig` - Search settings
- `OptimizationConfig` - Optimization settings
- `VisualizationConfig` - Visualization settings
- `TerminalVisualizationConfig` - Terminal visualization
- `WebVisualizationConfig` - Web visualization

**Functions:**
- `LoadConfig(configPath string) (*Config, error)` - Loads configuration from YAML and .env
- `validateConfig(config *Config) error` - Validates configuration values
- `ensureDirectories(config *Config) error` - Creates required directories

### Logger (`internal/app/logger.go`)
Handles application logging

## Paper Processing Pipeline

### Worker Pool (`internal/worker/pool.go`)

**Structs:**
- `ProcessingJob` - Represents a processing job
- `ProcessingResult` - Represents processing result
- `WorkerPool` - Manages worker pool

**Functions:**
- `NewWorkerPool(numWorkers int, config *app.Config, redisCache *cache.RedisCache) *WorkerPool` - Creates new worker pool
- `SetEnableRAG(enable bool)` - Sets RAG indexing flag
- `Start(ctx context.Context)` - Starts worker pool
- `worker(ctx context.Context, id int)` - Individual worker process
- `processJob(ctx context.Context, job *ProcessingJob) *ProcessingResult` - Processes individual PDF
- `SubmitJob(job *ProcessingJob)` - Submits job to pool
- `Close()` - Closes job channel
- `Wait()` - Waits for workers to finish
- `Results()` - Returns results channel
- `ProcessBatch(ctx context.Context, files []string, config *app.Config, force bool, enableRAG bool) error` - Processes batch of files
- `extractTitleFromLatex(latexContent string) string` - Extracts paper title from LaTeX

### Paper Indexing (`internal/worker/indexing.go`)
Handles RAG indexing for processed papers

**Functions:**
- `IndexPaperAfterProcessing(ctx context.Context, config *app.Config, paperTitle, latexContent, pdfPath string) error` - Indexes paper after processing

## AI Integration

### Analyzer (`internal/analyzer/analyzer.go`)

**Structs:**
- `Analyzer` - Main analyzer structure

**Functions:**
- `NewAnalyzer(config *app.Config) (*Analyzer, error)` - Creates new analyzer
- `Close() error` - Closes analyzer
- `GetClient() *GeminiClient` - Returns Gemini client
- `AnalyzePaper(ctx context.Context, pdfPath string) (string, error)` - Multi-stage paper analysis
- `simplAnalysis(ctx context.Context, pdfPath string) (string, error)` - Simple analysis
- `agenticAnalysis(ctx context.Context, pdfPath string) (string, error)` - Agentic analysis
- `validateLatexSyntax(ctx context.Context, latexContent string) (string, error)` - Syntax validation
- `cleanLatexOutput(content string) string` - Cleans LaTeX output

### Gemini Client (`internal/analyzer/gemini_client.go`)

**Structs:**
- `GeminiClient` - Gemini API client

**Functions:**
- `NewGeminiClient(apiKey, model string, temperature float64, maxTokens int) (*GeminiClient, error)` - Creates new client
- `Close() error` - Closes client connection
- `GenerateText(ctx context.Context, prompt string) (string, error)` - Generates text from prompt
- `AnalyzePDFWithVision(ctx context.Context, pdfPath, prompt string) (string, error)` - Multimodal PDF analysis
- `GenerateWithRetry(ctx context.Context, prompt string, maxAttempts int, backoffMultiplier int, initialDelayMs int) (string, error)` - Retry logic for generation
- `ListAvailableModels(ctx context.Context) ([]string, error)` - Lists available models
- `FindThinkingModel(ctx context.Context) (string, error)` - Finds best thinking model

### Prompts (`internal/analyzer/prompts.go`)

**Constants:**
- `AnalysisPrompt` - Main prompt for paper analysis
- `SyntaxValidationPrompt` - Prompt for LaTeX syntax validation

## Knowledge Graph System

### Graph Builder (`internal/graph/builder.go`)

**Structs:**
- `GraphConfig` - Graph configuration
- `GraphBuilder` - Graph builder base
- `PaperNode` - Paper node structure
- `AuthorNode` - Author node structure
- `InstitutionNode` - Institution node structure
- `ConceptNode` - Concept node structure
- `MethodNode` - Method node structure
- `VenueNode` - Venue node structure
- `DatasetNode` - Dataset node structure
- `CitationRelationship` - Citation relationship
- `AuthorshipRelationship` - Authorship relationship
- `AffiliationRelationship` - Affiliation relationship
- `UsesMethodRelationship` - Method usage relationship
- `PublishedInRelationship` - Publication relationship
- `CoAuthorshipRelationship` - Co-authorship relationship
- `UsesDatasetRelationship` - Dataset usage relationship
- `ExtendsRelationship` - Extension relationship
- `GraphStats` - Graph statistics
- `AuthorImpact` - Author impact metrics
- `CollaborationNetwork` - Collaboration network

**Functions:**
- `NewGraphBuilder(config *GraphConfig) (*GraphBuilder, error)` - Creates new graph builder
- `Close(ctx context.Context)` - Closes graph connection
- `InitializeSchema(ctx context.Context)` - Initializes schema
- `AddPaper(ctx context.Context, paper *PaperNode)` - Adds paper node
- `AddAuthor(ctx context.Context, author *AuthorNode)` - Adds author node
- `AddInstitution(ctx context.Context, inst *InstitutionNode)` - Adds institution node
- `AddConcept(ctx context.Context, concept *ConceptNode)` - Adds concept node
- `AddMethod(ctx context.Context, method *MethodNode)` - Adds method node
- `AddVenue(ctx context.Context, venue *VenueNode)` - Adds venue node
- `AddDataset(ctx context.Context, dataset *DatasetNode)` - Adds dataset node
- `LinkPaperToCitation(ctx context.Context, rel *CitationRelationship)` - Links papers via citations
- `GetStats(ctx context.Context) (*GraphStats, error)` - Gets graph statistics

### Enhanced Neo4j Builder (`internal/graph/enhanced_neo4j_builder.go`)

**Structs:**
- `EnhancedNeo4jBuilder` - Enhanced Neo4j builder

**Functions:**
- `NewEnhancedNeo4jBuilder(config *GraphConfig) (*EnhancedNeo4jBuilder, error)` - Creates enhanced builder
- `InitializeEnhancedSchema(ctx context.Context) error` - Enhanced schema initialization
- `AddAuthor(ctx context.Context, author *AuthorNode) error` - Enhanced author creation
- `AddInstitution(ctx context.Context, inst *InstitutionNode) error` - Enhanced institution creation
- `AddMethod(ctx context.Context, method *MethodNode) error` - Enhanced method creation
- `AddVenue(ctx context.Context, venue *VenueNode) error` - Enhanced venue creation
- `AddDataset(ctx context.Context, dataset *DatasetNode) error` - Enhanced dataset creation
- `LinkPaperToAuthor(ctx context.Context, rel *AuthorshipRelationship) error` - Authorship linking
- `LinkAuthorToInstitution(ctx context.Context, rel *AffiliationRelationship) error` - Institution linking
- `LinkPaperToMethod(ctx context.Context, rel *UsesMethodRelationship) error` - Method linking
- `LinkPaperToVenue(ctx context.Context, rel *PublishedInRelationship) error` - Venue linking
- `LinkCoAuthors(ctx context.Context, rel *CoAuthorshipRelationship) error` - Co-author linking
- `LinkPaperToDataset(ctx context.Context, rel *UsesDatasetRelationship) error` - Dataset linking
- `AddExtensionRelationship(ctx context.Context, rel *ExtendsRelationship) error` - Extension linking
- `GetAuthorImpact(ctx context.Context, authorName string) (*AuthorImpact, error)` - Author impact metrics
- `GetCollaborationNetwork(ctx context.Context, authorName string, depth int) (*CollaborationNetwork, error)` - Collaboration network

### Enhanced Models (`internal/graph/enhanced_models.go`)
Contains the model structures for enhanced graph components.

### Citation Extractor (`internal/graph/citation_extractor.go`)
Extracts citations from papers for graph relationships.

### Hybrid Search (`internal/graph/hybrid_search.go`)
Combines graph and vector search capabilities.

### Graph Models (`internal/graph/models.go`)
Base models for graph components.

## RAG & Chat System

### Embeddings (`internal/rag/embeddings.go`)

**Constants:**
- `EmbeddingModel` - Default embedding model
- `EmbeddingDimensions` - Embedding dimension size

**Structs:**
- `EmbeddingClient` - Embedding client structure

**Functions:**
- `NewEmbeddingClient(apiKey string) (*EmbeddingClient, error)` - Creates embedding client
- `Close() error` - Closes embedding client
- `GenerateEmbedding(ctx context.Context, text string) ([]float32, error)` - Generates single embedding
- `GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error)` - Generates batch embeddings

### FAISS Vector Store (`internal/rag/faiss_store.go`)

**Structs:**
- `FAISSVectorStore` - FAISS vector store
- `VectorDocument` - Vector document structure
- `SearchResult` - Search result structure

**Functions:**
- `NewFAISSVectorStore(indexDir string) (*FAISSVectorStore, error)` - Creates FAISS store
- `AddDocument(ctx context.Context, doc VectorDocument) error` - Adds document
- `AddDocuments(ctx context.Context, docs []VectorDocument) error` - Adds multiple documents
- `Search(ctx context.Context, queryEmbedding []float32, topK int, filter map[string]string) ([]SearchResult, error)` - Vector search
- `SearchBySource(ctx context.Context, queryEmbedding []float32, source string, topK int) ([]SearchResult, error)` - Source-specific search
- `GetDocumentsBySource(ctx context.Context, source string) ([]VectorDocument, error)` - Gets documents by source
- `DeleteBySource(ctx context.Context, source string) (int, error)` - Deletes by source
- `save() error` - Saves index to disk
- `load() error` - Loads index from disk
- `GetStats() map[string]interface{}` - Gets statistics
- `GetIndexedPapers() []string` - Gets indexed paper titles
- `cosineSimilarity(a, b []float32) float32` - Calculates cosine similarity

### Chunker (`internal/rag/chunker.go`)

**Constants:**
- `DefaultChunkSize` - Default chunk size
- `DefaultChunkOverlap` - Default chunk overlap

**Structs:**
- `Chunker` - Text chunker structure

**Functions:**
- `NewChunker(chunkSize, overlap int) *Chunker` - Creates chunker
- `ChunkText(text string) []string` - Chunks text
- `ChunkLatex(latexContent string) []string` - Chunks LaTeX content
- `chunkBySentences(text string) []string` - Chunks by sentences
- `chunkByWords(text string, chunkSize int) []string` - Chunks by words

### Indexer (`internal/rag/indexer.go`)

**Structs:**
- `Indexer` - Indexer structure

**Functions:**
- `NewIndexer(chunker *Chunker, embedClient *EmbeddingClient, vectorStore VectorStoreInterface) *Indexer` - Creates indexer
- `IndexPaper(ctx context.Context, paperTitle, latexContent, pdfPath string) error` - Indexes single paper
- `IndexPapers(ctx context.Context, papers []Paper) error` - Indexes multiple papers
- `ReindexPaper(ctx context.Context, paperTitle, latexContent, pdfPath string) error` - Reindexes paper
- `CheckIfIndexed(ctx context.Context, paperTitle string) (bool, int, error)` - Checks if paper is indexed
- `RemovePaper(ctx context.Context, paperTitle string) error` - Removes paper from index
- `RebuildIndex(ctx context.Context, papers []Paper) error` - Rebuilds entire index

### Retriever (`internal/rag/retriever.go`)

**Structs:**
- `RetrievalConfig` - Retrieval configuration
- `RetrievedContext` - Retrieved context structure
- `Retriever` - Retriever structure

**Functions:**
- `DefaultRetrievalConfig() RetrievalConfig` - Default retrieval config
- `NewRetriever(vectorStore VectorStoreInterface, embedClient *EmbeddingClient, config RetrievalConfig) *Retriever` - Creates retriever
- `Retrieve(ctx context.Context, query string, filter map[string]string) (*RetrievedContext, error)` - Retrieves context
- `RetrieveFromPaper(ctx context.Context, query, paperTitle string) (*RetrievedContext, error)` - Retrieves from specific paper
- `RetrieveMultiPaper(ctx context.Context, query string, paperTitles []string) (*RetrievedContext, error)` - Retrieves from multiple papers
- `RetrieveWithCitations(ctx context.Context, query string, filter map[string]string) (*RetrievedContext, error)` - Retrieves with citations
- `filterByScore(results []SearchResult) []SearchResult` - Filters by score
- `rankAndDeduplicate(results []SearchResult) []SearchResult` - Ranks and deduplicates
- `buildContext(results []SearchResult) *RetrievedContext` - Builds context from results
- `generateCitation(doc VectorDocument) string` - Generates citation
- `min(a, b int) int` - Helper function
- `truncateString(s string, maxLen int) string` - Truncates string

### Vector Store Interface (`internal/rag/vector_store_interface.go`)

**Interfaces:**
- `VectorStoreInterface` - Main vector store interface
- `VectorDocument` - Document structure
- `SearchResult` - Search result structure

### Vector Store (`internal/rag/vector_store.go`)
Implementation of vector store interface.

### Direct Indexer (`internal/rag/direct_indexer.go`)
Direct indexing functionality.

### Chat Engine (`internal/chat/chat_engine.go`)

**Constants:**
- `ChatHistoryPrefix` - Redis key prefix for chat histories
- `ChatHistoryTTL` - TTL for chat histories

**Structs:**
- `Message` - Chat message structure
- `ChatSession` - Chat session structure
- `ChatEngine` - Chat engine structure

**Functions:**
- `NewChatEngine(retriever *rag.Retriever, geminiClient *analyzer.GeminiClient, redisClient *redis.Client) *ChatEngine` - Creates chat engine
- `StartSession(ctx context.Context, paperTitles []string) (*ChatSession, error)` - Starts chat session
- `Chat(ctx context.Context, session *ChatSession, userMessage string) (*Message, error)` - Processes chat message
- `GetSession(ctx context.Context, sessionID string) (*ChatSession, error)` - Gets session
- `ListSessions(ctx context.Context) ([]*ChatSession, error)` - Lists sessions
- `DeleteSession(ctx context.Context, sessionID string) error` - Deletes session
- `ExportSessionToLatex(session *ChatSession) string` - Exports session to LaTeX
- `saveSession(ctx context.Context, session *ChatSession) error` - Saves session
- `buildPrompt(session *ChatSession, userMessage string, context *rag.RetrievedContext) string` - Builds RAG prompt
- `extractCitations(context *rag.RetrievedContext) []string` - Extracts citations
- `truncateString(s string, maxLen int) string` - Truncates string
- `escapeLatex(text string) string` - Escapes LaTeX characters
- `replaceAll(s, old, new string) string` - Replaces all occurrences

## Search Engine Microservice

### Search Client (`internal/search/client.go`)

**Structs:**
- `Client` - Search client structure
- `SearchQuery` - Search query structure
- `SearchResult` - Search result structure
- `SearchResponse` - Search response structure
- `DownloadRequest` - Download request structure
- `DownloadResponse` - Download response structure
- `HealthResponse` - Health check response structure

**Functions:**
- `NewClient(baseURL string) *Client` - Creates search client
- `Search(query *SearchQuery) (*SearchResponse, error)` - Performs search
- `DownloadPaper(pdfURL, filename string) (*DownloadResponse, error)` - Downloads paper
- `HealthCheck() (*HealthResponse, error)` - Health check
- `IsServiceRunning() bool` - Checks if service is running

### Python Search Engine Service (`services/search-engine/`)
Complete Python microservice for academic paper search.

**Main Components:**
- `app/main.py` - FastAPI application
- `app/models.py` - Data models
- `app/providers/` - Search providers (arXiv, OpenReview, ACL)
- `app/hybrid_search.py` - Hybrid search orchestrator
- `app/vector_store.py` - Vector store interface

## Caching System

### Redis Cache (`internal/cache/redis_cache.go`)

**Structs:**
- `CachedAnalysis` - Cached analysis structure
- `RedisCache` - Redis cache structure

**Functions:**
- `NewRedisCache(addr, password string, db int, ttl time.Duration) (*RedisCache, error)` - Creates Redis cache
- `Close() error` - Closes Redis connection
- `Get(ctx context.Context, contentHash string) (*CachedAnalysis, error)` - Gets cached analysis
- `Set(ctx context.Context, contentHash string, analysis *CachedAnalysis) error` - Sets cached analysis
- `Clear(ctx context.Context) (int64, error)` - Clears all cache
- `GetStats(ctx context.Context) (int64, error)` - Gets cache statistics
- `Exists(ctx context.Context, contentHash string) (bool, error)` - Checks if exists
- `Delete(ctx context.Context, contentHash string) error` - Deletes entry
- `ListAll(ctx context.Context) ([]*CachedAnalysis, error)` - Lists all entries

## Terminal UI

### TUI Model (`internal/tui/model.go`)

**Structs:**
- `Model` - Main TUI model
- `screen` - Screen enumeration
- `item` - List item structure
- `command` - Command structure

**Functions:**
- `InitialModel(configPath string) (*Model, error)` - Creates initial model
- `Init() tea.Cmd` - Initialization command
- `Update(msg tea.Msg) (tea.Model, tea.Cmd)` - Updates model
- `executeCommand(action string) (tea.Model, tea.Cmd)` - Executes command
- `Run(configPath string) error` - Runs TUI
- `handleBatchProcessing(config *app.Config) error` - Handles batch processing
- `handleMultiplePapersProcessing(selectedPapers []string, config *app.Config) error` - Handles multiple paper processing
- `handleSinglePaperProcessing(selectedPaper string, config *app.Config) error` - Handles single paper processing
- `handleOpenPDF(pdfPath string) error` - Opens PDF
- `handleProcessAndChat(pdfPath string, config *app.Config) error` - Processes and chats

### TUI Components (various files in `internal/tui/`)

**Files:**
- `chat.go` - Chat interface
- `chat_handlers.go` - Chat event handlers
- `chat_indexing.go` - Chat indexing operations
- `command_palette.go` - Command palette functionality
- `handlers.go` - General event handlers
- `loaders.go` - Loading indicators and progress bars
- `navigation.go` - Navigation logic
- `search.go` - Search interface
- `styles.go` - UI styling
- `types.go` - Type definitions
- `views.go` - UI views and layouts

## LaTeX Processing

### LaTeX Generator (`internal/generator/latex_generator.go`)

**Structs:**
- `LatexGenerator` - LaTeX generator structure

**Functions:**
- `NewLatexGenerator(outputDir string) *LatexGenerator` - Creates generator
- `GenerateLatexFile(paperTitle, latexContent string) (string, error)` - Generates LaTeX file
- `sanitizeFilename(name string) string` - Sanitizes filename

### LaTeX Compiler (`internal/compiler/latex_compiler.go`)

**Structs:**
- `LatexCompiler` - LaTeX compiler structure

**Functions:**
- `NewLatexCompiler(engine string, useLatexmk, cleanAux bool, outputDir string) *LatexCompiler` - Creates compiler
- `Compile(texPath string) (string, error)` - Compiles LaTeX to PDF
- `compileWithLatexmk(workDir, texFile string) error` - Compiles with latexmk
- `compileManual(workDir, texFile string) error` - Manual compilation
- `cleanAuxiliaryFiles(workDir, baseName string)` - Cleans aux files
- `CheckDependencies(useLatexmk bool, engine string) error` - Checks dependencies
- `checkCommand(cmd string) error` - Checks command availability

## File Utilities

### Hash Utilities (`pkg/fileutil/hash.go`)

**Functions:**
- `ComputeFileHash(filePath string) (string, error)` - Computes file hash
- `SanitizeFilename(name string) string` - Sanitizes filename
- `GetPDFFiles(dir string) ([]string, error)` - Gets PDF files
- `FileExists(path string) bool` - Checks if file exists

## Profiling System

### Profiler (`internal/profiler/profiler.go`)

**Structs:**
- `Profiler` - Profiler structure
- `FunctionTiming` - Function timing structure
- `ProfileConfig` - Profile configuration structure

**Functions:**
- `DefaultConfig() *ProfileConfig` - Default profile config
- `NewProfiler(config *ProfileConfig) (*Profiler, error)` - Creates profiler
- `Start() error` - Starts profiling
- `Stop() error` - Stops profiling
- `TimeFunction(name string) func()` - Times function execution
- `TimeFunctionWithContext(ctx context.Context, name string) func()` - Times function with context
- `recordTiming(name string, duration time.Duration)` - Records timing
- `writeTimingReport(path string) error` - Writes timing report
- `GetTimings() map[string]*FunctionTiming` - Gets timings
- `PrintTimings()` - Prints timings
- `MemoryStats() runtime.MemStats` - Gets memory stats
- `PrintMemoryStats()` - Prints memory stats

## UI Utilities (`internal/ui/ui.go`)

**Enums:**
- `ProcessingMode` - Processing mode enumeration

**Structs:**
- `ModeConfig` - Mode configuration

**Functions:**
- `GetModeConfigs() map[ProcessingMode]ModeConfig` - Gets mode configs
- `ShowBanner()` - Shows application banner
- `PromptMode() (ProcessingMode, error)` - Prompts for processing mode
- `PromptEnableRAG() bool` - Prompts to enable RAG
- `ShowModeDetails(mode ProcessingMode)` - Shows mode details
- `ShowModeDetailsWithConfig(mode ProcessingMode, actualConfig *app.Config)` - Shows mode details with config
- `PromptSelectPapers(papers []string) ([]string, error)` - Prompts to select papers
- `ConfirmProcessing(fileCount int) bool` - Confirms processing
- `CreateProgressBar(total int, description string) *progressbar.ProgressBar` - Creates progress bar
- `PrintSuccess(msg string)` - Prints success message
- `PrintError(msg string)` - Prints error message
- `PrintWarning(msg string)` - Prints warning message
- `PrintInfo(msg string)` - Prints info message
- `PrintStage(stage, description string)` - Prints stage header
- `PrintSummary(successful, failed, skipped int, totalTime time.Duration)` - Prints summary
- `formatBool(b bool) string` - Formats boolean
- `formatDuration(d time.Duration) string` - Formats duration

## Infrastructure & Deployment

### Docker Files

**Dockerfile:**
- Multi-stage build for optimized image size
- Ubuntu base with LaTeX tools installed
- Binary compilation and copying
- Configuration setup

**docker-compose.yml:**
- Archivist application service
- Redis cache service
- Neo4j graph database
- Network and volume configuration
- Resource limits and health checks

**docker-compose-graph.yml:**
- Neo4j graph database
- Qdrant vector database
- Redis cache
- Service-specific configurations

## Architecture Diagrams

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     ARCHIVIST SYSTEM ARCHITECTURE                       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐     │
│  │   CLI CLIENT    │    │   PYTHON API    │    │    DATABASES    │     │
│  │                 │    │   SEARCH ENGINE │    │                 │     │
│  │  rph process    │───▶│                 │───▶│  ┌─────────────┐ │     │
│  │  rph chat       │    │  arXiv API      │    │  │    REDIS    │ │     │
│  │  rph search     │    │  OpenReview API │    │  │   (Cache)   │ │     │
│  │  rph index      │    │  ACL API        │    │  └─────────────┘ │     │
│  └─────────────────┘    └─────────────────┘    │  ┌─────────────┐ │     │
│                                                │  │    NEO4J    │ │     │
│                                                │  │  (Graph DB) │ │     │
│  ┌─────────────────┐    ┌─────────────────┐    │  └─────────────┘ │     │
│  │  PROCESSING     │    │   GEMINI API    │    │  ┌─────────────┐ │     │
│  │   WORKERS       │───▶│                 │    │  │   QDRANT    │ │     │
│  │ (Parallel Pool) │    │  Vision API     │    │  │ (Vector DB) │ │     │
│  │                 │    │  Text Embedding │    │  └─────────────┘ │     │
│  └─────────────────┘    └─────────────────┘    └─────────────────┘     │
│                                                                         │
│  ┌─────────────────┐    ┌─────────────────┐                             │
│  │     TUI         │    │   FILE SYSTEM   │                             │
│  │                 │    │                 │                             │
│  │  Interactive    │    │  Input PDFs     │                             │
│  │  BubbleTea UI   │    │  Processed LaTeX│                             │
│  │                 │    │  Output PDFs    │                             │
│  └─────────────────┘    └─────────────────┘                             │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Complete System Workflow

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    PAPER PROCESSING WORKFLOW                                    │
├─────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                 │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                                    USER INITIATION                                        │ │
│  │                                                                                           │ │
│  │  User runs: rph process [pdf_file | directory | --select]                                  │ │
│  │  Example: rph process lib/paper.pdf                                                        │ │
│  │  or: rph process lib/                                                                      │ │
│  │  or: rph process --select                                                                  │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                                 CLI COMMAND HANDLER                                         │ │
│  │                                                                                           │ │
│  │  • Command: process                                                                      │ │
│  │  • Args validation: PDF file, directory, or interactive selection                        │ │
│  │  • Config loading from config/config.yaml                                                │ │
│  │  • Logger initialization                                                                 │ │
│  │  • Mode selection (fast/slow, interactive)                                               │ │
│  │  • Parallel worker count configuration                                                   │ │
│  │  • Dependency checking (LaTeX tools)                                                     │ │
│  │  • File discovery (PDFs to process)                                                      │ │
│  │  • Processing confirmation                                                               │ │
│  │  • RAG indexing option prompt                                                            │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                              WORKER POOL INITIALIZATION                                   │ │
│  │                                                                                           │ │
│  │  • NewWorkerPool(                                                                      │ │
│  │      numWorkers: config.Processing.MaxWorkers,                                           │ │
│  │      config: loaded config,                                                              │ │
│  │      redisCache: RedisCache instance                                                     │ │
│  │  )                                                                                       │ │
│  │  • SetEnableRAG(enableRAG option)                                                        │ │
│  │  • Start worker pool                                                                     │ │
│  │  • Submit jobs to pool                                                                   │ │
│  │  • Close job channel                                                                     │ │
│  │  • Wait for all workers to complete                                                      │ │
│  │  • Collect results from results channel                                                  │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                              INDIVIDUAL JOB PROCESSING                                    │ │
│  │                                                                                           │ │
│  │  For each PDF in job queue:                                                              │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                              HASH COMPUTATION                                       │ │ │
│  │  │  • ComputeFileHash(job.FilePath) → job.FileHash                                    │ │ │
│  │  │  • Used for cache lookup and identification                                          │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                    │                                                   │ │ │
│  │                                    ▼                                                   │ │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                            CACHE CHECK (REDIS)                                      │ │ │
│  │  │  • wp.cache.Get(ctx, fileHash)                                                     │ │ │
│  │  │  • Returns CachedAnalysis object if found                                          │ │ │
│  │  │  • Contains: latexContent, paperTitle, modelUsed, cachedAt                         │ │ │
│  │  │  • CACHE HIT: Use cached result, skip API call                                     │ │ │
│  │  │  • CACHE MISS: Continue to Gemini analysis                                         │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                    │                                                   │ │ │
│  │                                    ▼                                                   │ │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                         GEMINI ANALYSIS (API CALL)                                  │ │ │
│  │  │  • NewAnalyzer(config)                                                             │ │ │
│  │  │  • analyzer.AnalyzePaper(ctx, job.FilePath)                                        │ │ │
│  │  │  • Uses: config.Gemini.Model                                                       │ │ │
│  │  │  • Agentic workflow if enabled:                                                    │ │ │
│  │  │    - Multi-stage analysis with self-reflection                                     │ │ │
│  │  │    - Syntax validation after analysis                                              │ │ │
│  │  │  • Returns: latexContent string                                                    │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                    │                                                   │ │ │
│  │                                    ▼                                                   │ │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                        LATEX FILE GENERATION                                        │ │ │
│  │  │  • latexGen := NewLatexGenerator(config.TexOutputDir)                             │ │ │
│  │  │  • texPath, err := latexGen.GenerateLatexFile(paperTitle, latexContent)           │ │ │
│  │  │  • Creates: /app/tex_files/[sanitized_title].tex                                   │ │ │
│  │  │  • Sanitizes filename for filesystem compatibility                                 │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                    │                                                   │ │ │
│  │                                    ▼                                                   │ │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                         PDF COMPILATION (LaTeX)                                     │ │ │
│  │  │  • compiler := NewLatexCompiler(                                                  │ │ │
│  │  │      engine: config.Latex.Compiler,                                              │ │ │
│  │  │      useLatexmk: config.Latex.Engine == "latexmk",                                 │ │ │
│  │  │      cleanAux: config.Latex.CleanAux,                                              │ │ │
│  │  │      outputDir: config.ReportOutputDir                                             │ │ │
│  │  │    )                                                                               │ │ │
│  │  │  • reportPath, err := compiler.Compile(texPath)                                   │ │ │
│  │  │  • Creates: /app/reports/[sanitized_title].pdf                                     │ │ │
│  │  │  • Uses pdflatex, xelatex, or latexmk                                              │ │ │
│  │  │  • Multiple compilation passes for TOC and references                              │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                    │                                                   │ │ │
│  │                                    ▼                                                   │ │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                           RESULT CACHING (REDIS)                                    │ │ │
│  │  │  • If cache miss: Cache the analysis result                                        │ │ │
│  │  │  • cacheEntry := &cache.CachedAnalysis{                                           │ │ │
│  │  │      ContentHash: fileHash,                                                        │ │ │
│  │  │      PaperTitle: paperTitle,                                                       │ │ │
│  │  │      LatexContent: latexContent,                                                   │ │ │
│  │  │      ModelUsed: config.Gemini.Model                                                │ │ │
│  │  │    }                                                                               │ │ │
│  │  │  • wp.cache.Set(ctx, fileHash, cacheEntry)                                         │ │ │
│  │  │  • TTL configured in config: cache.ttl hours                                       │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                    │                                                   │ │ │
│  │                                    ▼                                                   │ │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                        RAG INDEXING (OPTIONAL)                                      │ │ │
│  │  │  • If wp.enableRAG is true:                                                        │ │ │
│  │  │  • IndexPaperAfterProcessing(ctx, config, paperTitle, latexContent, job.FilePath)  │ │ │
│  │  │  • This triggers embedding generation and vector storage                           │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                                PROCESSING SUMMARY                                         │ │
│  │                                                                                           │ │
│  │  • Progress bar showing: [###....] 60/100 papers processed                               │ │
│  │  • Success/Failure counts                                                                │ │
│  │  • Processing time per paper                                                             │ │
│  │  • Total processing time                                                                 │ │
│  │  • If failures: return error                                                             │ │
│  │  • If successful: launch TUI automatically                                               │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                               TUI AUTOLAUNCH                                                │ │
│  │                                                                                           │ │
│  │  • tui.Run(ConfigPath)                                                                   │ │
│  │  • Shows processing results in interactive terminal interface                            │ │
│  │  • Provides navigation to view processed papers, search, chat, etc.                      │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### Service Communication Flow

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────┐
│                           SERVICE COMMUNICATION MAP                                             │
├─────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                 │
│  ┌─────────────────┐          HTTP          ┌─────────────────────┐                             │
│  │   CLI Client    │ ────────────────▶      │   Python Search     │                             │
│  │    (rph)        │                        │      API            │                             │
│  └─────────────────┘                        └─────────────────────┘                             │
│         │                                             │                                         │
│         │                                    ┌─────────┴─────────┐                               │
│         │                                    │   External APIs   │                               │
│         │                                    │  (arXiv, etc.)    │                               │
│         │                                    └─────────┬─────────┘                               │
│         │                                              │                                         │
│         ▼                                              ▼                                         │
│  ┌─────────────────┐                        ┌─────────────────────┐                             │
│  │  Processing     │                        │      FAISS          │                             │
│  │    Workers      │ ─────────────────────▶ │   Vector Store      │                             │
│  │                 │   gRPC/Embeddings      │  (Semantic Search)  │                             │
│  └─────────────────┘                        └─────────────────────┘                             │
│         │                                              │                                         │
│         │                                              ▼                                         │
│         │                                    ┌─────────────────────┐                             │
│         │                                    │      Gemini         │                             │
│         │                                    │      APIs           │                             │
│         │                                    │ (Vision, Embedding) │                             │
│         ▼                                    └─────────────────────┘                             │
│  ┌─────────────────┐                              │                                             │
│  │     Redis       │ ──────────────────────▶      │                                             │
│  │     Cache       │                        ┌─────▼─────┐                                       │
│  │ (Results, Chat) │                        │   Neo4j   │                                       │
│  └─────────────────┘                        │ Knowledge │                                       │
│         │                                    │   Graph   │                                       │
│         │                                    └───────────┘                                       │
│         │                                              │                                         │
│         │                                              ▼                                         │
│         │                                    ┌─────────────────────┐                             │
│         └───────────────────────────────────▶│     LaTeX Tools     │                             │
│                                              │ (pdflatex, latexmk) │                             │
│                                              └─────────────────────┘                             │
│                                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Complete Workflow

### 1. Paper Processing Flow
User → CLI → Configuration → File Discovery → Worker Pool → Cache Check → Gemini Analysis → LaTeX Generation → PDF Compilation → RAG Indexing → Result Caching

### 2. Chat & RAG Flow
User Query → Embedding Generation → Vector Search → Context Retrieval → RAG Prompt → Gemini Response → Session Management → Response Display

### 3. Search Flow
User Query → CLI Client → Python API → Multi-Source Search → Relevance Ranking → Result Display → Download Option

### 4. Knowledge Graph Flow
Paper Processing → Information Extraction → Node Creation → Relationship Building → Graph Analytics → Querying

## Configuration

### Main Configuration (`config/config.yaml`)
Comprehensive configuration with sections for:
- Processing settings (workers, timeouts)
- Gemini AI settings (models, parameters)
- Agentic workflow configuration
- LaTeX compilation settings
- Caching configuration
- Knowledge graph settings
- Vector database settings
- Logging configuration

## Dependencies

### Go Modules
- github.com/charmbracelet/bubbles - Bubble UI components
- github.com/charmbracelet/bubbletea - Terminal UI framework
- github.com/charmbracelet/lipgloss - UI styling
- github.com/google/generative-ai-go - Gemini API client
- github.com/neo4j/neo4j-go-driver/v5 - Neo4j driver
- github.com/qdrant/go-client - Qdrant vector database client
- github.com/redis/go-redis/v9 - Redis client
- github.com/spf13/cobra - CLI framework
- github.com/spf13/viper - Configuration management
- google.golang.org/api - Google API client

### Python Dependencies (Search Service)
- FastAPI - Web framework
- arXiv - arXiv API client
- requests - HTTP requests
- Various search and data processing libraries

## Build & Deployment

### Build Process
1. Multi-stage Docker build
2. Go binary compilation
3. LaTeX tools installation
4. Configuration setup
5. Final image optimization

### Deployment Options
1. Docker Compose with all services
2. Individual container deployment
3. Cloud deployment with external services
4. Local development setup

## Testing & Development

### Testing Strategy
- Unit tests for core components
- Integration tests for workflows
- CLI command tests
- Performance testing with large document sets

### Development Tools
- Profiling capabilities
- Detailed logging
- Configuration validation
- Error handling and recovery

This comprehensive documentation covers all files, functions, and microservices in the Archivist system along with detailed architectural diagrams and workflow descriptions.