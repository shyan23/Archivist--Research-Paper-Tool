# Archivist Feature Improvements: Technical Specifications

This document provides detailed technical specifications for improving each major feature area based on the product strategy.

---

## 1. Paper Processing: From 22s to 15s

### Current Performance Analysis

**Bottleneck Breakdown:**
```
Total: 22.4 seconds
├─ PDF Upload to Gemini: 2-3s (depends on file size)
├─ Stage 1 (Methodology): 16-18s (Gemini Pro API latency)
├─ Stage 2 (Reflection): 2-4s (disabled in fast mode)
├─ Stage 3 (Validation): 1-2s
└─ LaTeX Compilation: 2-3s (3 passes of pdflatex)
```

###

 Optimization Strategy

#### A. Parallel Gemini Calls (Save 8-10s)

**Current Sequential:**
```go
// Stage 1: Methodology analysis
analysis := gemini.Analyze(pdfPath, methodologyPrompt)

// Stage 2: Extract metadata
metadata := gemini.Analyze(pdfPath, metadataPrompt)

// Total: 18-20s
```

**New Parallel:**
```go
type ParallelAnalysis struct {
    Methodology AnalysisResult
    Metadata    MetadataResult
    Error       error
}

func AnalyzePaperParallel(ctx context.Context, pdfPath string) (*ParallelAnalysis, error) {
    var wg sync.WaitGroup
    result := &ParallelAnalysis{}

    // Launch both analysis tasks concurrently
    wg.Add(2)

    // Goroutine 1: Deep methodology analysis
    go func() {
        defer wg.Done()
        analysis, err := geminiClient.AnalyzePDFWithVision(ctx, pdfPath, methodologyPrompt)
        if err != nil {
            result.Error = fmt.Errorf("methodology analysis failed: %w", err)
            return
        }
        result.Methodology = analysis
    }()

    // Goroutine 2: Metadata extraction
    go func() {
        defer wg.Done()
        metadata, err := geminiClient.AnalyzePDFWithVision(ctx, pdfPath, metadataPrompt)
        if err != nil {
            result.Error = fmt.Errorf("metadata extraction failed: %w", err)
            return
        }
        result.Metadata = metadata
    }()

    wg.Wait()

    if result.Error != nil {
        return nil, result.Error
    }

    return result, nil
}

// Time saved: 8-10 seconds (both API calls happen simultaneously)
```

#### B. Smart Context Extraction (Save 1-3s)

**Problem:** Sending entire PDF to Gemini (unnecessary tokens, slower upload)

**Solution:** Extract and send only relevant sections

```go
type PaperSections struct {
    Abstract      string
    Introduction  string
    Methodology   string
    Results       string
    Conclusion    string
    References    []string
}

func ExtractRelevantSections(pdfPath string) (*PaperSections, error) {
    // Use pypdf or unidoc to extract text first
    fullText := extractTextFromPDF(pdfPath)

    // Use lightweight Gemini call to identify section boundaries
    sectionBoundaries := identifySections(fullText)

    // Extract only methodology, results, conclusion
    relevantText := extractSections(fullText, sectionBoundaries,
        []string{"methodology", "results", "conclusion"})

    return relevantText, nil
}

// Then send only relevant text instead of full PDF
func AnalyzeWithSmartContext(ctx context.Context, sections *PaperSections) {
    // Construct focused prompt with just the necessary sections
    focusedContent := fmt.Sprintf(`
Abstract: %s

Methodology: %s

Results: %s

Conclusion: %s
`, sections.Abstract, sections.Methodology, sections.Results, sections.Conclusion)

    // Send text instead of PDF (faster upload, fewer tokens)
    return geminiClient.GenerateText(ctx, focusedContent + analysisPrompt)
}

// Savings:
// - Smaller payload: 1-2s faster upload
// - Fewer tokens: Potentially faster processing
// - More focused analysis: Better quality
```

#### C. Pre-compiled LaTeX Templates (Save 0.5-1s)

**Current:** Generate LaTeX from scratch each time
**New:** Use pre-compiled template with variable substitution

```go
// Load template once at startup
var latexTemplate *template.Template

func init() {
    latexTemplate = template.Must(template.ParseFiles("templates/paper_report.tex"))
}

type TemplateData struct {
    Title            string
    Authors          string
    Summary          string
    ProblemStatement string
    Prerequisites    string
    Methodology      string
    WowMoment        string
    Results          string
    Conclusion       string
}

func GenerateLatexFast(data *TemplateData) (string, error) {
    var buf bytes.Buffer
    err := latexTemplate.Execute(&buf, data)
    if err != nil {
        return "", err
    }
    return buf.String(), nil
}

// No more string concatenation, regex replacement, or manual LaTeX generation
// Just fill in the blanks: 0.1s instead of 0.5-1s
```

#### D. Cached Embeddings (Save 2-3s for chat)

**Current:** Generate embeddings during chat indexing (slow)
**New:** Generate during processing, cache for later

```go
func (wp *WorkerPool) processJob(ctx context.Context, job *ProcessingJob) *ProcessingResult {
    // ... existing processing ...

    // Generate embeddings immediately after analysis
    if wp.enableRAG {
        log.Printf("  🔢 Pre-generating embeddings...")
        chunks := chunker.ChunkLatexContent(latexContent)
        embeddings := make([][]float32, len(chunks))

        // Batch embedding generation (faster than one-by-one)
        for i := 0; i < len(chunks); i += 10 {
            end := min(i+10, len(chunks))
            batch := chunks[i:end]
            batchEmbeddings, err := embeddingClient.GenerateBatchEmbeddings(ctx, batch)
            if err != nil {
                log.Printf("  ⚠️  Embedding generation failed: %v", err)
                continue
            }
            copy(embeddings[i:], batchEmbeddings)
        }

        // Store in Qdrant immediately
        for i, chunk := range chunks {
            doc := VectorDocument{
                ID:        generateID(paperTitle, i),
                Content:   chunk,
                Embedding: embeddings[i],
                Metadata:  map[string]string{"paper": paperTitle},
            }
            vectorStore.AddDocument(ctx, doc)
        }

        log.Printf("  ✓ %d chunks indexed", len(chunks))
    }

    // Result: Paper is chat-ready immediately after processing
    // No separate indexing step needed
}
```

#### E. Streaming Compilation (Perceived Speed)

**Psychological improvement:** Show progress while compiling

```go
func (lc *LatexCompiler) CompileWithProgress(texPath string) error {
    steps := []string{
        "Running pdflatex pass 1/3...",
        "Running pdflatex pass 2/3...",
        "Running pdflatex pass 3/3...",
        "Cleaning auxiliary files...",
    }

    for i, step := range steps {
        fmt.Printf("\r  📄 %s", step)

        cmd := exec.Command("pdflatex", "-interaction=nonstopmode", texPath)
        err := cmd.Run()
        if err != nil {
            return fmt.Errorf("compilation failed at pass %d: %w", i+1, err)
        }

        // Show progress bar
        progress := float64(i+1) / float64(len(steps)) * 100
        fmt.Printf(" (%.0f%%)", progress)
    }

    fmt.Println("\r  ✓ PDF compiled successfully!                    ")
    return nil
}

// Doesn't actually save time, but feels faster to user
```

**Target Performance After Optimizations:**
```
New Total: ~15 seconds
├─ PDF Upload: 0s (text extraction instead)
├─ Parallel Analysis: 8-10s (both calls at once)
├─ LaTeX Generation: 0.1s (template substitution)
└─ Compilation: 2-3s (with progress indicator)

Saved: 7-8 seconds
```

---

## 2. Prompt Engineering: Blueprint Alignment

### Current Prompt Issues

1. **Missing "WOW MOMENT" emphasis**
2. **Excludes experimental results** (blueprint wants them)
3. **Vague prerequisites** ("linear algebra" instead of specifics)
4. **No architecture diagrams**
5. **Inconsistent structure**

### New Blueprint-Aligned Prompt

```go
const NewAnalysisPrompt = `You are creating a comprehensive student guide for an AI/ML research paper.

CONTEXT:
Your audience is CS graduate students studying AI/ML. They have:
- Solid programming background (Python, PyTorch)
- Basic ML knowledge (supervised learning, gradient descent)
- Familiarity with neural networks (CNNs, RNNs basics)

But they may NOT know:
- Latest architectures (Transformers, attention mechanisms)
- Advanced optimization techniques
- Domain-specific tricks and innovations

Your job is to bridge that gap with crystal-clear explanations.

---

MANDATORY STRUCTURE (FOLLOW EXACTLY):

\section{Executive Summary}
Write 3-4 sentences answering:
1. What problem does this paper solve?
2. What is the proposed solution?
3. Why does this matter to the field?

Example:
"This paper addresses the quadratic complexity bottleneck of transformers when processing
long sequences. The authors propose Linformer, which approximates self-attention using
low-rank matrix factorization, reducing complexity from O(n²) to O(n). This enables
transformers to process sequences 100x longer while maintaining competitive accuracy,
unlocking applications in long-document understanding and genomics."

---

\section{Problem Statement}

\subsection{The Challenge}
Explain the specific technical problem or limitation being addressed.
Be concrete with examples.

\subsection{Why Existing Solutions Fail}
Don't just say "prior work is slow" - explain WHY:
- What is the fundamental bottleneck?
- What tradeoff do existing methods make?
- Why can't we just scale up existing approaches?

\subsection{The Gap This Paper Fills}
What insight or technique does this paper contribute?

---

\section{Prerequisites}

\begin{prerequisite}
\textbf{Before reading this paper, you should understand:}

\textbf{Concepts:}
\begin{itemize}
    \item Matrix multiplication and computational complexity (O(n²) vs O(n))
    \item Softmax function and attention mechanisms
    \item Low-rank matrix approximation (SVD basics)
    \item Backpropagation and gradient descent
\end{itemize}

\textbf{Papers to read first:}
\begin{itemize}
    \item "Attention Is All You Need" (Vaswani et al., 2017) - Foundation
    \item "BERT" (Devlin et al., 2018) - Application of transformers
\end{itemize}

\textbf{Math background needed:}
\begin{itemize}
    \item Linear algebra: eigenvalues, matrix rank, matrix factorization
    \item Calculus: partial derivatives, chain rule
    \item Probability: expected value, variance
\end{itemize}
\end{prerequisite}

DO NOT write vague prerequisites like "familiarity with machine learning".
BE SPECIFIC about exact concepts, formulas, and prior papers needed.

---

\section{Methodology}

\subsection{High-Level Architecture}

[Provide a text-based architecture diagram using LaTeX formatting]

Example:
\begin{verbatim}
Input Sequence (x₁, x₂, ..., xₙ)
         ↓
   Embedding Layer
         ↓
   ┌─────────────────────────────┐
   │  Linformer Block            │
   │  ┌──────────────────────┐   │
   │  │ Low-Rank Projection  │   │
   │  │    K' = W_K · K      │   │
   │  │    V' = W_V · V      │   │
   │  └──────────────────────┘   │
   │  ┌──────────────────────┐   │
   │  │ Attention            │   │
   │  │  A = softmax(QK'^T)  │   │
   │  │  Output = A · V'     │   │
   │  └──────────────────────┘   │
   └─────────────────────────────┘
         ↓
   Feed-Forward Network
         ↓
   Output Predictions
\end{verbatim}

\subsection{Step-by-Step Explanation}

Walk through the methodology like teaching a class:

\textbf{Step 1: Problem Setup}
- What are the inputs and outputs?
- What are we trying to optimize?

\textbf{Step 2: Key Innovation}
- What's the core algorithmic contribution?
- Why does this work?

\textbf{Step 3: Implementation Details}
- Specific hyperparameters used
- Training procedures
- Architectural choices and their rationale

\subsection{Mathematical Formulation}

For each key equation, provide:
1. The equation itself
2. Explanation of each variable
3. Intuition for why this formulation works

Example:
\begin{equation}
\text{Attention}(Q, K', V') = \text{softmax}\left(\frac{QK'^T}{\sqrt{d_k}}\right) V'
\end{equation}

Where:
\begin{itemize}
    \item $Q \in \mathbb{R}^{n \times d}$: Query matrix from input sequence (length n)
    \item $K' \in \mathbb{R}^{k \times d}$: Projected key matrix (compressed from n to k)
    \item $V' \in \mathbb{R}^{k \times d}$: Projected value matrix
    \item $d_k$: Dimension of key vectors (for scaling)
\end{itemize}

\textbf{Intuition:} By projecting K and V from length n to k (where k << n),
we reduce the attention matrix from $n \times n$ to $n \times k$, cutting
complexity from O(n²) to O(nk).

---

\section{The Breakthrough} \label{sec:wow}

\begin{keyinsight}
\textbf{The "WOW" Moment:}

[Explain the key insight that makes this paper revolutionary]

Why this matters:
- [Impact on the field]
- [What becomes possible now that wasn't before]
- [How this changes our understanding]
\end{keyinsight}

This section should make the reader think "Oh! That's brilliant!"

Example:
\begin{keyinsight}
The breakthrough is realizing that the self-attention matrix has low-rank structure.
Instead of computing the full $n \times n$ attention matrix, we can project keys
and values to a much smaller dimension $k$ (typically k=256 even when n=100,000).

This is brilliant because:
\begin{itemize}
    \item It's theoretically grounded (Johnson-Lindenstrauss lemma guarantees preservation)
    \item It's practically effective (minimal accuracy loss in experiments)
    \item It's embarrassingly simple (just two learned projection matrices)
\end{itemize}

What becomes possible: Transformers can now process DNA sequences (100K+ base pairs),
entire books (50K+ words), and hour-long audio (1M+ time steps) - all previously impossible.
\end{keyinsight}

---

\section{Experimental Setup}

\subsection{Datasets}
List all benchmarks used with brief descriptions:
- WikiText-103: Language modeling on Wikipedia articles
- ImageNet: Image classification (1000 classes)
- etc.

\subsection{Baselines}
What are they comparing against?
- Vanilla Transformer (Vaswani et al., 2017)
- Transformer-XL (Dai et al., 2019)
- etc.

\subsection{Evaluation Metrics}
- Perplexity (lower is better)
- Accuracy
- Training time
- Memory usage

---

\section{Results and Quantitative Improvements}

\textbf{DO NOT} skip the numbers. Students need to see concrete evidence.

\subsection{Performance Benchmarks}

\begin{table}[h]
\centering
\begin{tabular}{lcccc}
\hline
\textbf{Model} & \textbf{Perplexity} & \textbf{Speed} & \textbf{Memory} \\
\hline
Transformer (baseline) & 24.3 & 1.0x & 16 GB \\
Linformer (this work) & 24.8 & 3.2x & 4 GB \\
\hline
\end{tabular}
\caption{Results on WikiText-103 with sequence length 8192}
\end{table}

\textbf{Key Findings:}
\begin{itemize}
    \item \textbf{Speed:} 3.2x faster training (due to O(n) vs O(n²) complexity)
    \item \textbf{Memory:} 4x less GPU memory (smaller attention matrices)
    \item \textbf{Accuracy:} Minimal degradation (+0.5 perplexity = 2\% worse)
\end{itemize}

\subsection{Where It Excels}
- Long sequences (8K+ tokens): 5x speedup
- Limited GPU memory scenarios: Enables training on single GPU

\subsection{Limitations}
BE HONEST about what doesn't work well:
- Short sequences (<512 tokens): No advantage over vanilla Transformer
- Tasks requiring precise long-range dependencies: Slight accuracy drop
- Dynamic sequence lengths: Projection dimension k must be fixed

---

\section{Impact and Conclusion}

\subsection{Contributions to the Field}
- Theoretical: Proved self-attention has low-rank structure
- Practical: Enabled transformers on long sequences
- Impact: Cited by 400+ papers, used in production systems

\subsection{Follow-Up Work}
Papers that built on this:
- Performer (Choromanski et al., 2020): Kernel-based approximation
- Nyströmformer (Xiong et al., 2021): Nyström method for attention

\subsection{Why Students Should Care}
- Demonstrates how theoretical insights (low-rank approximation) solve practical problems
- Shows that O(n²) complexity isn't fundamental to transformers
- Opens research directions in efficient architectures

---

FORMATTING REQUIREMENTS:

1. Use LaTeX environments:
   - \begin{keyinsight}...\end{keyinsight} for breakthroughs
   - \begin{prerequisite}...\end{prerequisite} for prerequisites
   - \begin{table}...\end{table} for quantitative results

2. Use \textbf{} for emphasis, \textit{} for technical terms

3. Include equations with \begin{equation}...\end{equation}

4. Use itemize and enumerate for lists

5. Add \label{} for cross-references

6. Keep paragraphs concise (3-5 sentences max)

7. Use examples liberally to illustrate abstract concepts

---

TONE AND STYLE:

✅ DO:
- Write like explaining to a smart friend over coffee
- Use concrete examples and numbers
- Admit when something is complex ("This is subtle...")
- Provide intuition before formalism
- Celebrate clever ideas ("This is brilliant because...")

❌ DON'T:
- Dumb down the content
- Skip the math
- Use vague language ("performs well", "significantly better")
- Copy-paste from the paper without explanation
- Assume prerequisite knowledge without stating it

---

Now analyze the paper and generate the LaTeX content following this structure exactly.
`
```

**Changes from Current Prompt:**

| Aspect | Old | New |
|--------|-----|-----|
| Structure | Loose guidelines | Mandatory sections with examples |
| Prerequisites | Vague ("linear algebra") | Specific (eigenvalues, SVD, chain rule) |
| WOW Moment | Optional mention | Dedicated section with emphasis |
| Results | "Skip experiments" | "DO NOT skip numbers" |
| Architecture | Text description | ASCII diagram in LaTeX |
| Tone | Generic | Specific voice ("explain over coffee") |

---

## 3. Knowledge Graph: Complete Implementation

### Critical TODOs to Fix

**File: `internal/graph/hybrid_search.go`**

#### TODO #1: Graph Traversal (Line 315)

```go
// Current stub:
func (hs *HybridSearch) graphTraversal(ctx context.Context, paperID string, depth int) ([]string, error) {
    // TODO: Implement actual graph traversal using Neo4j Cypher
    return []string{}, nil
}

// Complete implementation:
func (hs *HybridSearch) graphTraversal(ctx context.Context, paperID string, depth int) ([]string, error) {
    // Cypher query for citation traversal
    query := `
    MATCH path = (start:Paper {id: $paperID})-[:CITES*1..$depth]-(related:Paper)
    WHERE start <> related
    RETURN DISTINCT related.id AS paperID,
           related.title AS title,
           length(path) AS distance,
           related.citationCount AS citations
    ORDER BY distance ASC, citations DESC
    LIMIT 50
    `

    params := map[string]interface{}{
        "paperID": paperID,
        "depth":   depth,
    }

    result, err := hs.neo4jSession.Run(ctx, query, params)
    if err != nil {
        return nil, fmt.Errorf("graph traversal query failed: %w", err)
    }

    var relatedPapers []string
    for result.Next(ctx) {
        record := result.Record()
        paperID, _ := record.Get("paperID")
        relatedPapers = append(relatedPapers, paperID.(string))
    }

    if err = result.Err(); err != nil {
        return nil, fmt.Errorf("error iterating results: %w", err)
    }

    return relatedPapers, nil
}
```

#### TODO #2: Similarity Traversal (Line 325)

```go
// Current stub:
func (hs *HybridSearch) similarityTraversal(ctx context.Context, paperID string) ([]string, error) {
    // TODO: Implement actual similarity traversal using Neo4j Cypher
    return []string{}, nil
}

// Complete implementation:
func (hs *HybridSearch) similarityTraversal(ctx context.Context, paperID string) ([]string, error) {
    // Get embedding of source paper from Qdrant
    sourceEmbedding, err := hs.getEmbedding(ctx, paperID)
    if err != nil {
        return nil, fmt.Errorf("failed to get source embedding: %w", err)
    }

    // Vector similarity search in Qdrant
    similarDocs, err := hs.vectorStore.Search(ctx, sourceEmbedding, 20, map[string]string{})
    if err != nil {
        return nil, fmt.Errorf("vector search failed: %w", err)
    }

    // Get papers that share authors or concepts with similar papers
    var paperIDs []string
    for _, doc := range similarDocs {
        paperIDs = append(paperIDs, doc.Metadata["paper_id"])
    }

    // Cypher query to find papers sharing authors/concepts
    query := `
    MATCH (similar:Paper)
    WHERE similar.id IN $paperIDs
    MATCH (similar)-[:AUTHORED_BY]->(author:Author)<-[:AUTHORED_BY]-(related:Paper)
    WHERE related.id <> $sourcePaperID
    WITH related, count(DISTINCT author) AS sharedAuthors
    MATCH (similar)-[:DISCUSSES]->(concept:Concept)<-[:DISCUSSES]-(related)
    WITH related, sharedAuthors, count(DISTINCT concept) AS sharedConcepts
    RETURN DISTINCT related.id AS paperID,
           sharedAuthors,
           sharedConcepts,
           (sharedAuthors * 2 + sharedConcepts) AS relevanceScore
    ORDER BY relevanceScore DESC
    LIMIT 30
    `

    params := map[string]interface{}{
        "paperIDs":       paperIDs,
        "sourcePaperID":  paperID,
    }

    result, err := hs.neo4jSession.Run(ctx, query, params)
    if err != nil {
        return nil, fmt.Errorf("similarity query failed: %w", err)
    }

    var relatedPapers []string
    for result.Next(ctx) {
        record := result.Record()
        relPaperID, _ := record.Get("paperID")
        relatedPapers = append(relatedPapers, relPaperID.(string))
    }

    return relatedPapers, nil
}
```

### New Features to Implement

#### A. Citation Path Finding

```go
// File: internal/graph/citation_path.go

type CitationPath struct {
    Start    string   // Paper ID
    End      string   // Paper ID
    Path     []string // Paper IDs in order
    Length   int
    Papers   []*PaperNode
}

func (gb *GraphBuilder) FindCitationPath(ctx context.Context, startPaperID, endPaperID string) (*CitationPath, error) {
    // Cypher query using shortest path algorithm
    query := `
    MATCH (start:Paper {id: $startID}), (end:Paper {id: $endID})
    MATCH path = shortestPath((start)-[:CITES*]-(end))
    WHERE length(path) <= 10
    RETURN [node IN nodes(path) | node.id] AS paperIDs,
           [node IN nodes(path) | node.title] AS titles,
           length(path) AS pathLength
    `

    params := map[string]interface{}{
        "startID": startPaperID,
        "endID":   endPaperID,
    }

    result, err := gb.session.Run(ctx, query, params)
    if err != nil {
        return nil, fmt.Errorf("path finding query failed: %w", err)
    }

    if !result.Next(ctx) {
        return nil, fmt.Errorf("no citation path found between papers")
    }

    record := result.Record()
    paperIDs, _ := record.Get("paperIDs")
    pathLength, _ := record.Get("pathLength")

    path := &CitationPath{
        Start:  startPaperID,
        End:    endPaperID,
        Path:   convertToStringSlice(paperIDs),
        Length: int(pathLength.(int64)),
    }

    // Fetch full paper details
    for _, paperID := range path.Path {
        paper, err := gb.GetPaper(ctx, paperID)
        if err != nil {
            log.Printf("Warning: couldn't fetch paper %s: %v", paperID, err)
            continue
        }
        path.Papers = append(path.Papers, paper)
    }

    return path, nil
}
```

#### B. Author Impact Calculation

```go
// File: internal/graph/author_impact.go

type AuthorImpact struct {
    Name             string
    PaperCount       int
    TotalCitations   int
    HIndex           int
    Collaborators    int
    TopPapers        []*PaperNode
    CollaborationMap map[string]int // Author name → co-authored papers count
}

func (gb *GraphBuilder) CalculateAuthorImpact(ctx context.Context, authorName string) (*AuthorImpact, error) {
    // Comprehensive Cypher query for author metrics
    query := `
    MATCH (author:Author {name: $authorName})-[:AUTHORED]->(paper:Paper)
    WITH author, collect(paper) AS papers, count(paper) AS paperCount

    // Calculate total citations
    UNWIND papers AS p
    WITH author, papers, paperCount, sum(p.citationCount) AS totalCitations

    // Find collaborators
    MATCH (author)-[:AUTHORED]->(:Paper)<-[:AUTHORED]-(collaborator:Author)
    WHERE collaborator <> author
    WITH author, papers, paperCount, totalCitations,
         count(DISTINCT collaborator) AS collaboratorCount,
         collect(DISTINCT collaborator.name) AS collaboratorNames

    // Get top papers
    UNWIND papers AS topPaper
    WITH author, paperCount, totalCitations, collaboratorCount, collaboratorNames,
         topPaper
    ORDER BY topPaper.citationCount DESC
    LIMIT 10

    RETURN paperCount,
           totalCitations,
           collaboratorCount,
           collect({id: topPaper.id, title: topPaper.title,
                    citations: topPaper.citationCount}) AS topPapers,
           collaboratorNames
    `

    params := map[string]interface{}{
        "authorName": authorName,
    }

    result, err := gb.session.Run(ctx, query, params)
    if err != nil {
        return nil, fmt.Errorf("author impact query failed: %w", err)
    }

    if !result.Next(ctx) {
        return nil, fmt.Errorf("author not found: %s", authorName)
    }

    record := result.Record()
    paperCount, _ := record.Get("paperCount")
    totalCitations, _ := record.Get("totalCitations")
    collaboratorCount, _ := record.Get("collaboratorCount")

    impact := &AuthorImpact{
        Name:           authorName,
        PaperCount:     int(paperCount.(int64)),
        TotalCitations: int(totalCitations.(int64)),
        Collaborators:  int(collaboratorCount.(int64)),
    }

    // Calculate H-index
    impact.HIndex = gb.calculateHIndex(ctx, authorName)

    // Get collaboration map
    impact.CollaborationMap = gb.getCollaborationMap(ctx, authorName)

    return impact, nil
}

func (gb *GraphBuilder) calculateHIndex(ctx context.Context, authorName string) int {
    // H-index: largest number h such that author has h papers with ≥h citations each
    query := `
    MATCH (author:Author {name: $authorName})-[:AUTHORED]->(paper:Paper)
    RETURN paper.citationCount AS citations
    ORDER BY citations DESC
    `

    result, err := gb.session.Run(ctx, query, map[string]interface{}{"authorName": authorName})
    if err != nil {
        return 0
    }

    var citationCounts []int
    for result.Next(ctx) {
        record := result.Record()
        citations, _ := record.Get("citations")
        citationCounts = append(citationCounts, int(citations.(int64)))
    }

    // Calculate h-index
    hIndex := 0
    for i, citations := range citationCounts {
        if citations >= (i + 1) {
            hIndex = i + 1
        } else {
            break
        }
    }

    return hIndex
}
```

#### C. Smart Paper Recommendations

```go
// File: internal/graph/recommendations.go

type Recommendation struct {
    Paper           *PaperNode
    RelevanceScore  float64
    Reasons         []string // Why this paper is recommended
}

func (gb *GraphBuilder) RecommendPapers(ctx context.Context, basedOnPaperID string, topK int) ([]*Recommendation, error) {
    // Multi-factor recommendation algorithm

    // Factor 1: Citation network (papers cited by papers you read)
    citationRecs := gb.getCitationBasedRecommendations(ctx, basedOnPaperID, topK*2)

    // Factor 2: Author-based (other papers by same authors)
    authorRecs := gb.getAuthorBasedRecommendations(ctx, basedOnPaperID, topK*2)

    // Factor 3: Concept-based (papers on similar topics)
    conceptRecs := gb.getConceptBasedRecommendations(ctx, basedOnPaperID, topK*2)

    // Factor 4: Vector similarity (semantic similarity)
    vectorRecs := gb.getVectorBasedRecommendations(ctx, basedOnPaperID, topK*2)

    // Combine and rank
    scoredRecs := gb.combineRecommendations(citationRecs, authorRecs, conceptRecs, vectorRecs)

    // Sort by relevance and take top K
    sort.Slice(scoredRecs, func(i, j int) bool {
        return scoredRecs[i].RelevanceScore > scoredRecs[j].RelevanceScore
    })

    if len(scoredRecs) > topK {
        scoredRecs = scoredRecs[:topK]
    }

    return scoredRecs, nil
}

func (gb *GraphBuilder) getCitationBasedRecommendations(ctx context.Context, paperID string, topK int) map[string]float64 {
    // Papers frequently cited together with the base paper
    query := `
    MATCH (base:Paper {id: $paperID})<-[:CITES]-(citing:Paper)-[:CITES]->(related:Paper)
    WHERE related.id <> $paperID
    WITH related, count(citing) AS cocitations
    RETURN related.id AS paperID, cocitations
    ORDER BY cocitations DESC
    LIMIT $topK
    `

    result, err := gb.session.Run(ctx, query, map[string]interface{}{
        "paperID": paperID,
        "topK":    topK,
    })
    if err != nil {
        return make(map[string]float64)
    }

    recommendations := make(map[string]float64)
    maxCocitations := 0.0

    for result.Next(ctx) {
        record := result.Record()
        relatedID, _ := record.Get("paperID")
        cocitations, _ := record.Get("cocitations")

        cocitCount := float64(cocitations.(int64))
        if cocitCount > maxCocitations {
            maxCocitations = cocitCount
        }
        recommendations[relatedID.(string)] = cocitCount
    }

    // Normalize scores to 0-1 range
    for id := range recommendations {
        recommendations[id] /= maxCocitations
    }

    return recommendations
}

func (gb *GraphBuilder) combineRecommendations(citation, author, concept, vector map[string]float64) []*Recommendation {
    // Weighted fusion of different recommendation signals
    weights := map[string]float64{
        "citation": 0.35,
        "author":   0.20,
        "concept":  0.25,
        "vector":   0.20,
    }

    // Aggregate scores
    combinedScores := make(map[string]float64)
    reasons := make(map[string][]string)

    for paperID, score := range citation {
        combinedScores[paperID] += score * weights["citation"]
        reasons[paperID] = append(reasons[paperID], "Frequently cited together")
    }

    for paperID, score := range author {
        combinedScores[paperID] += score * weights["author"]
        reasons[paperID] = append(reasons[paperID], "Same author(s)")
    }

    for paperID, score := range concept {
        combinedScores[paperID] += score * weights["concept"]
        reasons[paperID] = append(reasons[paperID], "Similar research topics")
    }

    for paperID, score := range vector {
        combinedScores[paperID] += score * weights["vector"]
        reasons[paperID] = append(reasons[paperID], "Semantically similar")
    }

    // Convert to Recommendation structs
    var recommendations []*Recommendation
    for paperID, score := range combinedScores {
        paper, _ := gb.GetPaper(context.Background(), paperID)
        if paper != nil {
            recommendations = append(recommendations, &Recommendation{
                Paper:          paper,
                RelevanceScore: score,
                Reasons:        reasons[paperID],
            })
        }
    }

    return recommendations
}
```

#### D. TUI Graph Explorer Integration

```go
// File: internal/tui/graph_explorer.go

type GraphExplorerModel struct {
    currentPaper    *graph.PaperNode
    citationTree    []*graph.PaperNode
    recommendations []*graph.Recommendation
    graphStats      *graph.GraphStats
    selectedIndex   int
    viewport        viewport.Model
}

func (m *GraphExplorerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "c":
            // View citations
            return m, m.loadCitations()

        case "p":
            // Find citation path
            return m, m.showPathFinder()

        case "r":
            // Show recommendations
            return m, m.loadRecommendations()

        case "a":
            // Author impact
            return m, m.showAuthorImpact()
        }

    case CitationsLoadedMsg:
        m.citationTree = msg.Citations
        return m, nil

    case RecommendationsLoadedMsg:
        m.recommendations = msg.Recommendations
        return m, nil
    }

    return m, nil
}

func (m *GraphExplorerModel) View() string {
    if m.currentPaper == nil {
        return "No paper selected"
    }

    var b strings.Builder

    // Header
    b.WriteString(titleStyle.Render("Knowledge Graph Explorer"))
    b.WriteString("\n\n")

    // Current paper info
    b.WriteString(fmt.Sprintf("📄 %s (%d)\n", m.currentPaper.Title, m.currentPaper.Year))
    b.WriteString(fmt.Sprintf("├─ Cited by %d papers\n", m.currentPaper.CitedByCount))
    b.WriteString(fmt.Sprintf("├─ Cites %d papers\n", len(m.currentPaper.References)))
    b.WriteString(fmt.Sprintf("└─ Authors: %s\n", strings.Join(m.currentPaper.Authors, ", ")))
    b.WriteString("\n")

    // Most influential citations
    if len(m.citationTree) > 0 {
        b.WriteString("🔗 Most Influential Citations:\n")
        for i, cited := range m.citationTree[:min(5, len(m.citationTree))] {
            b.WriteString(fmt.Sprintf("%d. %s (%d citations)\n",
                i+1, cited.Title, cited.CitedByCount))
        }
        b.WriteString("\n")
    }

    // Recommendations
    if len(m.recommendations) > 0 {
        b.WriteString("💡 Recommended Next:\n")
        for i, rec := range m.recommendations[:min(3, len(m.recommendations))] {
            b.WriteString(fmt.Sprintf("%d. %s (%.2f match)\n   Reasons: %s\n",
                i+1, rec.Paper.Title, rec.RelevanceScore,
                strings.Join(rec.Reasons, ", ")))
        }
        b.WriteString("\n")
    }

    // Controls
    b.WriteString("[c] Citations  [p] Citation Path  [r] Recommend  [a] Author Impact\n")

    return b.String()
}
```

---

## 4. Chat System Enhancements

### A. Adaptive Context Retrieval

```go
// File: internal/chat/adaptive_retrieval.go

type QueryComplexity int

const (
    SimpleQuery QueryComplexity = iota  // "What is BERT?"
    ComparisonQuery                     // "Compare BERT and GPT"
    SynthesisQuery                      // "Synthesize attention mechanisms"
)

func (ce *ChatEngine) analyzeQueryComplexity(userMessage string) QueryComplexity {
    userMessageLower := strings.ToLower(userMessage)

    // Keywords indicating comparison
    comparisonWords := []string{"compare", "difference", "versus", "vs", "contrast"}
    for _, word := range comparisonWords {
        if strings.Contains(userMessageLower, word) {
            return ComparisonQuery
        }
    }

    // Keywords indicating synthesis
    synthesisWords := []string{"synthesize", "summarize", "overview", "survey", "all papers"}
    for _, word := range synthesisWords {
        if strings.Contains(userMessageLower, word) {
            return SynthesisQuery
        }
    }

    // Check question complexity by counting entities
    entities := ce.extractEntities(userMessage)
    if len(entities) > 2 {
        return SynthesisQuery
    } else if len(entities) == 2 {
        return ComparisonQuery
    }

    return SimpleQuery
}

func (ce *ChatEngine) retrieveAdaptiveContext(ctx context.Context, query string, paperTitles []string) (*rag.RetrievedContext, error) {
    complexity := ce.analyzeQueryComplexity(query)

    // Adaptive parameters based on complexity
    var topK int
    var minScore float32

    switch complexity {
    case SimpleQuery:
        topK = 3
        minScore = 0.4
    case ComparisonQuery:
        topK = 10
        minScore = 0.3
    case SynthesisQuery:
        topK = 20
        minScore = 0.25
    }

    log.Printf("Query complexity: %v, retrieving top %d chunks", complexity, topK)

    return ce.retriever.Retrieve(ctx, query, paperTitles, topK, minScore)
}
```

### B. Semantic Conversation Memory

```go
// File: internal/chat/semantic_memory.go

type ConversationMemory struct {
    ShortTerm  []Message          // Current session
    LongTerm   []PastConversation // Previous sessions on related topics
    UserProfile *UserProfile
}

type PastConversation struct {
    SessionID string
    Topic     string
    Messages  []Message
    Relevance float64
}

type UserProfile struct {
    ResearchInterests []string
    PreferredDepth    string // "concise", "detailed", "comprehensive"
    PaperHistory      []string
}

func (ce *ChatEngine) buildContextWithMemory(ctx context.Context, session *ChatSession, userMessage string) string {
    // 1. Get short-term memory (current conversation)
    recentContext := ce.getRecentConversation(session, 5)

    // 2. Get long-term memory (past relevant conversations)
    relevantPast := ce.findRelevantPastConversations(ctx, userMessage, 2)

    // 3. Get user profile
    profile := ce.getUserProfile(ctx, session.UserID)

    // 4. Build comprehensive context
    var contextParts []string

    // Long-term memory context
    if len(relevantPast) > 0 {
        contextParts = append(contextParts,
            "RELEVANT PAST DISCUSSIONS:")
        for _, past := range relevantPast {
            summary := ce.summarizeConversation(past.Messages)
            contextParts = append(contextParts,
                fmt.Sprintf("- Topic: %s\n  Summary: %s", past.Topic, summary))
        }
        contextParts = append(contextParts, "")
    }

    // User profile context
    if profile != nil && len(profile.ResearchInterests) > 0 {
        contextParts = append(contextParts,
            fmt.Sprintf("USER RESEARCH INTERESTS: %s",
                strings.Join(profile.ResearchInterests, ", ")))
        contextParts = append(contextParts, "")
    }

    // Short-term memory (current conversation)
    if len(recentContext) > 0 {
        contextParts = append(contextParts, "CURRENT CONVERSATION:")
        for _, msg := range recentContext {
            contextParts = append(contextParts,
                fmt.Sprintf("%s: %s", msg.Role, msg.Content))
        }
        contextParts = append(contextParts, "")
    }

    return strings.Join(contextParts, "\n")
}

func (ce *ChatEngine) findRelevantPastConversations(ctx context.Context, query string, topK int) []PastConversation {
    // Get embedding of current query
    queryEmbedding, err := ce.embeddingClient.GenerateEmbedding(ctx, query)
    if err != nil {
        return nil
    }

    // Search through past conversation summaries in Redis
    pastSessionKeys, err := ce.redisClient.Keys(ctx, ChatHistoryPrefix+"*").Result()
    if err != nil {
        return nil
    }

    var relevantConversations []PastConversation

    for _, key := range pastSessionKeys {
        sessionJSON, err := ce.redisClient.Get(ctx, key).Result()
        if err != nil {
            continue
        }

        var pastSession ChatSession
        json.Unmarshal([]byte(sessionJSON), &pastSession)

        // Calculate semantic similarity to current query
        sessionSummary := ce.summarizeConversation(pastSession.Messages)
        summaryEmbedding, _ := ce.embeddingClient.GenerateEmbedding(ctx, sessionSummary)

        similarity := cosineSimilarity(queryEmbedding, summaryEmbedding)

        if similarity > 0.6 { // Relevance threshold
            relevantConversations = append(relevantConversations, PastConversation{
                SessionID: pastSession.ID,
                Topic:     pastSession.PaperTitles[0], // First paper as topic
                Messages:  pastSession.Messages,
                Relevance: float64(similarity),
            })
        }
    }

    // Sort by relevance and return top K
    sort.Slice(relevantConversations, func(i, j int) bool {
        return relevantConversations[i].Relevance > relevantConversations[j].Relevance
    })

    if len(relevantConversations) > topK {
        relevantConversations = relevantConversations[:topK]
    }

    return relevantConversations
}
```

### C. Proactive Insights

```go
// File: internal/chat/proactive_insights.go

func (ce *ChatEngine) generateProactiveInsights(ctx context.Context, session *ChatSession, response string) string {
    // After generating response, add proactive insights

    insights := []string{}

    // 1. Cross-paper connections
    if len(session.PaperTitles) == 1 {
        relatedPapers := ce.findRelatedPapersInLibrary(ctx, session.PaperTitles[0])
        if len(relatedPapers) > 0 {
            insights = append(insights,
                fmt.Sprintf("\n\n💡 I notice you've also read %s, which relates to this topic. Would you like me to compare them?",
                    relatedPapers[0]))
        }
    }

    // 2. Deeper dive suggestions
    technicalTerms := ce.extractTechnicalTerms(response)
    if len(technicalTerms) > 0 {
        insights = append(insights,
            fmt.Sprintf("\n\n🔍 This mentions %s. Would you like me to explain this concept in more detail?",
                technicalTerms[0]))
    }

    // 3. Citation recommendations
    citations := ce.extractCitations(response)
    if len(citations) > 0 {
        unreadCitations := ce.filterUnreadPapers(ctx, session.UserID, citations)
        if len(unreadCitations) > 0 {
            insights = append(insights,
                fmt.Sprintf("\n\n📚 This references %s, which you haven't read yet. I can help you find and process it.",
                    unreadCitations[0]))
        }
    }

    // 4. Methodology connections
    if strings.Contains(strings.ToLower(response), "attention") ||
       strings.Contains(strings.ToLower(response), "transformer") {
        papersUsingAttention := ce.findPapersWithMethod(ctx, "attention mechanism")
        if len(papersUsingAttention) > 1 {
            insights = append(insights,
                fmt.Sprintf("\n\n🔬 You have %d papers in your library that use attention mechanisms. Would you like an overview of how they differ?",
                    len(papersUsingAttention)))
        }
    }

    // Add one most relevant insight (don't overwhelm user)
    if len(insights) > 0 {
        return response + insights[0]
    }

    return response
}

func (ce *ChatEngine) findRelatedPapersInLibrary(ctx context.Context, paperTitle string) []string {
    // Query knowledge graph for related papers
    // (Citation relationships, shared authors, shared concepts)

    // Use graph builder's recommendation system
    paperID := ce.getPaperIDByTitle(ctx, paperTitle)
    recommendations, _ := ce.graphBuilder.RecommendPapers(ctx, paperID, 3)

    var titles []string
    for _, rec := range recommendations {
        titles = append(titles, rec.Paper.Title)
    }

    return titles
}
```

---

## 5. Architecture Simplification

See `ARCHITECTURE_SIMPLIFICATION.md` for detailed migration plan.

**Summary of Changes:**

1. **Remove Kafka**: Direct Neo4j writes from Go worker
2. **Remove Python Graph Service**: Merge into Go using go-neo4j-driver
3. **Remove Python RAG**: Already implemented in Go
4. **Consolidate to Qdrant**: Delete FAISS code
5. **Keep Python Search Service**: Makes sense as separate microservice

**Benefits:**
- Single Go binary deployment
- Simpler debugging and testing
- Fewer failure modes
- Faster processing (no Kafka latency)

---

*Continue to ARCHITECTURE_SIMPLIFICATION.md for migration steps*
*Continue to UX_BLUEPRINT.md for user experience details*
*Continue to IMPLEMENTATION_ROADMAP.md for 90-day execution plan*
