# Archivist System Workflows

## Complete System Workflow

This document describes the complete workflows of the Archivist system, from user interaction to final output, including all microservice interactions.

### 1. Paper Processing Workflow

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

### 2. Chat & RAG Workflow

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    CHAT & RAG WORKFLOW                                          │
├─────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                 │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                               CHAT INITIALIZATION                                         │ │
│  │                                                                                           │ │
│  │  User runs: rph chat [pdf_file] or rph chat --papers [files...]                          │ │
│  │  or selects "Chat with Papers" from TUI                                                  │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                             COMPONENT INITIALIZATION                                      │ │
│  │                                                                                           │ │
│  │  • Redis client for cache: github.com/redis/go-redis/v9                                  │ │
│  │  • Embedding client: rag.NewEmbeddingClient(config.Gemini.APIKey)                       │ │
│  │  • Vector store: rag.NewFAISSVectorStore(config.FAISS.IndexDir)                         │ │
│  │  • Retriever: rag.NewRetriever(vectorStore, embedClient, retrievalConfig)               │ │
│  │  • Gemini client for responses: analyzer.NewGeminiClient(...)                           │ │
│  │  • Chat engine: chat.NewChatEngine(retriever, geminiClient, redisClient)                │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                           PAPER INDEX VERIFICATION                                        │ │
│  │                                                                                           │ │
│  │  For each paperTitle in paperPaths:                                                      │ │
│  │  • indexer.CheckIfIndexed(ctx, paperTitle)                                               │ │
│  │  • If not indexed:                                                                       │ │
│  │    - Show error: "Run 'archivist process' first to index this paper"                     │ │
│  │    - Return error                                                                        │ │
│  │  • If indexed:                                                                           │ │
│  │    - Show: "✓ paperTitle (X chunks indexed)"                                             │ │
│  │  • Continue if all papers are indexed                                                    │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                              CHAT SESSION START                                           │ │
│  │                                                                                           │ │
│  │  • chatEngine.StartSession(ctx, paperTitles)                                             │ │
│  │  • Returns session object with ID, paper titles, messages list, timestamps               │ │
│  │  • Session stored in Redis with TTL: 24 hours                                            │ │
│  │  • Session key: "archivist:chat:history:" + session.ID                                   │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                           INTERACTIVE CHAT LOOP                                           │ │
│  │                                                                                           │ │
│  │  For each user message:                                                                  │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                             USER INPUT                                              │ │ │
│  │  │  • promptui.Prompt for user message                                                │ │ │
│  │  │  • Accepts: "exit", "quit", "export", or regular question                          │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                    │                                                   │ │ │
│  │                                    ▼                                                   │ │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                        CONTEXT RETRIEVAL (RAG)                                      │ │ │
│  │  │  • retriever.RetrieveMultiPaper(ctx, userInput, paperTitles)                      │ │ │
│  │  │  • Generate embedding for userInput: embedClient.GenerateEmbedding()              │ │ │
│  │  • Search vector store for relevant chunks: vectorStore.Search()                       │ │ │
│  │  │  • Filter by paper sources if specified                                            │ │ │
│  │  │  • Return top-K chunks with similarity scores                                      │ │ │
│  │  │  • Apply min score threshold: config.retrieval.MinScore                            │ │ │
│  │  │  • Build context string from retrieved chunks                                      │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                    │                                                   │ │ │
│  │                                    ▼                                                   │ │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                           GEMINI RESPONSE                                           │ │ │
│  │  │  • Build RAG prompt with:                                                          │ │ │
│  │  │    - Retrieved context from papers                                                 │ │ │
│  │  │    - Conversation history (last 3 exchanges)                                       │ │ │
│  │  │    - User's current question                                                       │ │ │
│  │  │    - Instruction guidelines for response format                                    │ │ │
│  │  │  • geminiClient.GenerateText(ctx, ragPrompt)                                      │ │ │
│  │  │  • Returns response with technical explanations and citations                      │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                    │                                                   │ │ │
│  │                                    ▼                                                   │ │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                        SESSION UPDATE & DISPLAY                                     │ │ │
│  │  │  • Add user message to session.Messages                                            │ │ │
│  │  │  • Add assistant message to session.Messages                                       │ │ │
│  │  │  • Update session.LastUpdated timestamp                                            │ │ │
│  │  │  • Save updated session to Redis                                                   │ │ │
│  │  │  • Display response with citations to user                                         │ │ │
│  │  │  • Loop back to user input                                                         │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                              EXPORT & CLEANUP                                             │ │
│  │                                                                                           │ │
│  │  On exit:                                                                                │ │
│  │  • Export to LaTeX if requested: chatEngine.ExportSessionToLatex(session)              │ │
│  │  • Save session to Redis with TTL                                                      │ │
│  │  • Close all clients and connections                                                   │ │
│  │  • Return to main menu or exit                                                         │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 3. Search Service Workflow

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                SEARCH SERVICE WORKFLOW                                          │
├─────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                 │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                             CLI CLIENT REQUEST                                            │ │
│  │                                                                                           │ │
│  │  User runs: rph search "query terms" [--sources arxiv,openreview,acl] [--download]       │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                            SEARCH CLIENT INIT                                             │ │
│  │                                                                                           │ │
│  │  • client := search.NewClient(config.searchServiceURL)                                  │ │
│  │  • Check service health: client.IsServiceRunning()                                      │ │
│  │  • If service not running: show error with setup instructions                           │ │
│  │  • Create search query object: &SearchQuery{...}                                        │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                          HTTP REQUEST TO PYTHON API                                       │ │
│  │                                                                                           │ │
│  │  • POST /api/search to Python service                                                    │ │
│  │  • Body: JSON with query, max_results, sources, dates                                    │ │
│  │  • Timeout: 30 seconds                                                                   │ │
│  │  • Content-Type: application/json                                                        │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                        PYTHON FASTAPI HANDLER                                             │ │
│  │                                                                                           │ │
│  │  @app.post("/api/search")                                                                │ │
│  │  async def search_papers(query: SearchQuery):                                            │ │
│  │  • Create search orchestrator: get_search_orchestrator()                                 │ │
│  │  • Perform search across all specified sources                                           │ │
│  │  • Apply fuzzy matching and abbreviation expansion                                       │ │
│  │  • Rank results by relevance                                                             │ │
│  │  • Return SearchResponse with results                                                    │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                         MULTI-SOURCE SEARCH                                               │ │
│  │                                                                                           │ │
│  │  For each source (arXiv, OpenReview, ACL):                                               │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                            ARXIV SEARCH                                             │ │ │
│  │  │  • Use arXiv API with query terms                                                  │ │ │
│  │  │  • Apply date filters and categories                                               │ │ │
│  │  │  • Parse metadata and create SearchResult objects                                  │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                         OPENREVIEW SEARCH                                           │ │ │
│  │  │  • Query OpenReview API for conference papers                                      │ │ │
│  │  │  • Handle conference-specific endpoints                                            │ │ │
│  │  │  • Extract paper details and create SearchResult objects                           │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                           ACL SEARCH                                                │ │ │
│  │  │  • Query ACL Anthology API for NLP papers                                        │ │ │
│  │  │  • Handle venue and topic filtering                                              │ │ │
│  │  │  • Create standardized SearchResult objects                                      │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                         HYBRID SEARCH & RANKING                                         │ │
│  │                                                                                           │ │
│  │  • Combine results from all sources                                                      │ │
│  │  • Apply fuzzy string matching for typo tolerance                                        │ │
│  │  • Calculate relevance scores using multiple factors                                     │ │
│  │  • Expand common abbreviations (CNN, BERT, GAN, etc.)                                    │ │
│  │  • Rank results by relevance score                                                       │ │
│  │  • Return top N results as SearchResponse                                                │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                          RESULTS FORMATTING & DISPLAY                                     │ │
│  │                                                                                           │ │
│  │  • Format results with:                                                                  │ │
│  │    - Title, authors, abstract, publication date                                          │ │
│  │    - Relevance, fuzzy, and similarity scores                                             │ │
│  │    - PDF and source URLs                                                                 │ │
│  │    - Categories and venue information                                                    │ │
│  │  • Color-coded output using github.com/fatih/color                                       │ │
│  │  • Pagination for large result sets                                                      │ │
│  │  • Interactive download option if --download flag specified                              │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                            DOWNLOAD WORKFLOW                                              │ │
│  │                                                                                           │ │
│  │  If --download flag:                                                                     │ │
│  │  • Interactive paper selection using promptui                                            │ │
│  │  • For selected papers:                                                                  │ │
│  │    - Determine source from URL                                                           │ │
│  │    - Call appropriate provider's download method                                         │ │
│  │    - Save to /tmp/archivist_downloads temporarily                                        │ │
│  │    - Move to config.InputDir (lib/) directory                                            │ │
│  │  • Show download statistics and file sizes                                               │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 4. Knowledge Graph Workflow

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────┐
│                              KNOWLEDGE GRAPH WORKFLOW                                           │
├─────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                 │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                           GRAPH INITIALIZATION                                            │ │
│  │                                                                                           │ │
│  │  During: rph process with graph.enabled: true or rph graph-init                          │ │
│  │  • Load graph config from config.Graph section                                           │ │
│  │  • Connect to Neo4j: bolt://localhost:7687                                               │ │
│  │  • Initialize EnhancedNeo4jBuilder with credentials                                      │ │
│  │  • Create indexes and constraints: paper_title_unique, author_name_unique, etc.          │ │
│  │  • Set up async building with max_graph_workers                                          │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                        PAPER ANALYSIS & EXTRACTION                                        │ │
│  │                                                                                           │ │
│  │  During paper processing, extract:                                                       │ │
│  │  • Paper metadata: title, authors, venue, year, DOI                                      │ │
│  │  • Authors: names, affiliations, ORCID, h-index                                          │ │
│  │  • Institutions: names, countries, research domains                                      │ │
│  │  • Methods: techniques, architectures, algorithms mentioned                              │ │
│  │  • Datasets: benchmarks, corpora used in experiments                                     │ │
│  │  • Citations: in-text and reference list citations                                       │ │
│  │  • Concepts: technical terms, domain concepts                                            │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                      NEO4J NODE & RELATIONSHIP CREATION                                   │ │
│  │                                                                                           │ │
│  │  Create nodes:                                                                           │ │
│  │  • (:Paper {title, authors, year, venue, doi, abstract})                                 │ │
│  │  • (:Author {name, orcid, affiliation, field, h_index})                                  │ │
│  │  • (:Institution {name, country, type, research_domain})                                 │ │
│  │  • (:Method {name, type, description, complexity})                                       │ │
│  │  • (:Dataset {name, type, size, description, url})                                       │ │
│  │  • (:Venue {name, type, rank, impact_factor})                                           │ │
│  │  │                                                                                       │ │
│  │  Create relationships:                                                                   │ │
│  │  • (:Paper)-[:WRITTEN_BY {position, is_corresponding}]->(:Author)                       │ │
│  │  • (:Author)-[:AFFILIATED_WITH {role, start_year, end_year}]->(:Institution)           │ │
│  │  • (:Paper)-[:USES_METHOD {is_main_method, description}]->(:Method)                     │ │
│  │  • (:Paper)-[:USES_DATASET {purpose, results, metric, score}]->(:Dataset)              │ │
│  │  • (:Paper)-[:PUBLISHED_IN {year, pages, best_paper_award}]->(:Venue)                  │ │
│  │  • (:Author)-[:CO_AUTHORED_WITH {joint_papers, first_colab, last_colab, weight}]-(:Author)│ │
│  │  • (:Paper)-[:CITES {context, importance}]->(:Paper)                                    │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                           GRAPH ANALYTICS & QUERIES                                       │ │
│  │                                                                                           │ │
│  │  Available analytics:                                                                    │ │
│  │  • Author impact metrics (GetAuthorImpact): paper count, citations, h-index             │ │
│  │  • Collaboration networks (GetCollaborationNetwork): co-authorship relationships        │ │
│  │  • Citation analysis: paper influence, knowledge flow                                     │ │
│  │  • Research trends: method adoption, venue popularity                                     │ │
│  │  • Similarity search: find related papers, methods, authors                               │ │
│  │  • Pathfinding: connections between researchers, institutions                           │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                                           │
│                                    ▼                                                           │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                         HYBRID SEARCH INTEGRATION                                         │ │
│  │                                                                                           │ │
│  │  During search operations:                                                               │ │
│  │  • Combine vector similarity (FAISS) with graph relationships (Neo4j)                    │ │
│  │  • Default weights: 50% vector, 30% graph, 20% keyword                                   │ │
│  │  • Traversal depth: 2 hops maximum                                                       │ │
│  │  • Graph-enhanced results: papers connected through citations, co-authors, methods       │ │
│  │  • Knowledge graph provides context for RAG responses                                    │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Inter-Microservice Communication

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

## Complete Data Flow

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                 COMPLETE DATA FLOW                                              │
├─────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                 │
│  ┌─────────────────┐  PDF   ┌─────────────────┐  LaTeX  ┌─────────────────┐        ┌─────────┐ │
│  │    INPUT PDF    │───────▶│   PROCESSING    │───────▶│    LATEX ->     │───────▶│         │ │
│  │   [lib/]        │        │   PIPELINE      │        │   COMPILATION   │        │  PDF    │ │
│  │                 │        │                 │        │                 │        │ OUTPUT  │ │
│  │  (Source)       │        │  (Gemini API)   │        │  (LaTeX Tools)  │        │[reports]│ │
│  └─────────────────┘        └─────────────────┘        └─────────────────┘        └─────────┘ │
│         │                             │                           │                      │     │
│         │                             │                           │                      │     │
│         ▼                             ▼                           ▼                      ▼     │
│  ┌─────────────────┐        ┌─────────────────┐         ┌─────────────────┐      ┌─────────────┐ │
│  │   FILE HASH     │        │  GEMINI RESULT  │         │   PDF RESULT    │      │    CACHE    │ │
│  │   COMPUTATION   │        │  (Latex Text)   │         │   GENERATION    │      │   STORAGE   │ │
│  │  (SHA256/MD5)   │        │                 │         │                 │      │  (Redis)    │ │
│  │    UNIQUE       │        │  CACHE CHECK    │         │ STATUS TRACKING │      │             │ │
│  └─────────────────┘        │   RESULT        │         │  PROGRESS BAR   │      │  PAPER ID   │ │
│         │                    │  (IF EXISTS)    │         │  SUCCESS/FAIL   │      │  CONTENT    │ │
│         │                    └─────────────────┘         └─────────────────┘      │   HASH    │ │
│         │                             │                           │                 └─────────────┘ │
│         │                             │                           │                      │         │
│         ▼                             ▼                           ▼                      ▼         │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────────────┤ │
│  │                    EMBEDDING & INDEXING PIPELINE (FOR RAG CHAT)                               │ │
│  │  ┌─────────────────┐  TEXT   ┌─────────────────┐  VECTORS  ┌─────────────────┐                │ │
│  │  │   LATEX TEXT    │───────▶│  TEXT CHUNKING  │──────────▶│   FAISS/QDRANT  │───────────────▶│ │
│  │  │   EXTRACTION    │    │    │   (512 tokens)  │           │  VECTOR STORE   │                │ │
│  │  │ (from processed │    │    │                 │           │   (Similarity   │                │ │
│  │  │   papers)       │    │    │  SECTION-TAGGED │           │   SEARCH)       │                │ │
│  │  └─────────────────┘    │    │   CHUNKS)       │           └─────────────────┘                │ │
│  │                         │    └─────────────────┘                 │                            │ │
│  │                         │            │                           │                            │ │
│  │                         │            ▼                           ▼                            │ │
│  │                         │  ┌─────────────────┐        ┌─────────────────────────────────┐    │ │
│  │                         └─▶│ EMBEDDING GEN   │───────▶│    CHAT CONTEXT RETRIEVAL       │    │ │
│  │                            │  (768-dim)      │        │    (RAG: Retrieval + Generation)│    │ │
│  │                            │ (Gemini API)    │        │                                 │    │ │
│  │                            └─────────────────┘        └─────────────────────────────────┘    │ │
│  └─────────────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                                 │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                             CHAT CONVERSATION FLOW                                        │ │
│  │  ┌─────────────────┐  QUERY   ┌─────────────────┐  RESPONSE  ┌─────────────────────────┐ │ │
│  │  │   USER QUERY    │────────▶ │  RAG PROCESSING │──────────▶ │   AI-ENHANCED         │ │ │
│  │  │   (Natural      │          │   (Similarity   │            │   RESPONSE WITH         │ │ │
│  │  │   Language)     │          │   Search +      │            │   CITATIONS &           │ │ │
│  │  │                 │          │   Context       │            │   EXPLANATIONS          │ │ │
│  │  └─────────────────┘          │   Retrieval)    │            │                       │ │ │
│  │         │                     └─────────────────┘            └─────────────────────────┘ │ │
│  │         │                             │                                │                 │ │
│  │         │                             │                                │                 │ │
│  │         ▼                             ▼                                ▼                 │ │
│  │  ┌─────────────────┐        ┌─────────────────────────┐    ┌─────────────────────────────┐ │ │
│  │  │   EMBEDDING     │        │   RETRIEVED CONTEXT     │    │   CITATION METADATA &       │ │ │
│  │  │  GENERATION     │        │   FROM PAPER CHUNKS     │    │   SOURCE ATTRIBUTION        │ │ │
│  │  │ (Query Text)    │        │   (Based on Similarity) │    │                           │ │ │
│  │  └─────────────────┘        └─────────────────────────┘    └─────────────────────────────┘ │ │
│  │         │                             │                                │                 │ │
│  │         │                             │                                │                 │ │
│  │         └─────────────────────────────┼────────────────────────────────┘                 │ │
│  │                                       │                                                  │ │
│  │                                       ▼                                                  │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                            CONVERSATION HISTORY                                     │ │ │
│  │  │  ┌─────────────────┐  STORE   ┌─────────────────┐  LOAD   ┌─────────────────────┐  │ │ │
│  │  │  │   CURRENT       │────────▶ │   REDIS CACHE   │────────▶│   PAST EXCHANGES    │  │ │ │
│  │  │  │  CONVERSATION   │          │ (SESSION DATA)  │         │   FOR CONTEXT       │  │ │ │
│  │  │  │                 │          │                 │         │                     │  │ │ │
│  │  │  │                 │          │  TTL: 24 Hours  │         │  LATEST 3 EXCHANGES │  │ │ │
│  │  │  └─────────────────┘          └─────────────────┘         └─────────────────────┘  │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────────────────┘ │ │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Summary

This documentation outlines the comprehensive workflows of the Archivist system:

1. **Paper Processing Workflow**: Complete pipeline from PDF input to processed LaTeX/PDF output with caching and optional RAG indexing
2. **Chat & RAG Workflow**: Interactive Q&A system with semantic search using vector embeddings and context retrieval
3. **Search Service Workflow**: Multi-source academic paper search with download capabilities
4. **Knowledge Graph Workflow**: Neo4j-based graph database for paper relationships and analytics

The system integrates multiple microservices including Python FastAPI for search, Redis for caching, Neo4j for knowledge graph, FAISS/Qdrant for vector storage, and Gemini APIs for AI processing, all orchestrated through the main Go CLI application.