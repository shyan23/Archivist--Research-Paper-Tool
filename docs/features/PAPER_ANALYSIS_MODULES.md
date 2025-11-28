# Paper Analysis Modules

Two powerful modules for advanced paper analysis and discovery.

## Module 1: Similar Paper Finder

Analyzes a research paper and finds similar papers based on its methodology, key concepts, and techniques.

### Features

- **Essence Extraction**: Uses AI to extract the core methodology, key concepts, problem domain, and techniques from a paper
- **Smart Search**: Automatically constructs search queries based on the extracted essence
- **Multi-Source**: Searches across arXiv, OpenReview, and ACL Anthology
- **Relevance Ranking**: Results are ranked by similarity and relevance

### Usage

```bash
# Find similar papers
./rph similar lib/paper.pdf

# Limit results
./rph similar lib/paper.pdf --max-results 20

# Download similar papers
./rph similar lib/paper.pdf --download

# Use custom search service URL
./rph similar lib/paper.pdf --service-url http://localhost:8000
```

### How It Works

1. **Extract Paper Essence**:
   - Main methodology (e.g., "Transformer-based approach with attention mechanisms")
   - Key concepts (e.g., ["self-attention", "positional encoding", "multi-head attention"])
   - Problem domain (e.g., "Neural machine translation")
   - Techniques (e.g., ["layer normalization", "residual connections"])
   - Related fields (e.g., ["NLP", "sequence-to-sequence models"])

2. **Build Search Query**:
   - Combines methodology, top concepts, and techniques
   - Weights more important terms higher

3. **Search and Rank**:
   - Queries multiple paper databases
   - Returns ranked results with relevance scores

### Example Output

```
🔍 Finding Similar Papers
   Paper: attention_is_all_you_need.pdf
   Max results: 10

📊 Analyzing paper...
✓ Paper Essence Extracted:

   Main Methodology: Transformer architecture using self-attention mechanisms
   Problem Domain: Sequence transduction for neural machine translation
   Key Concepts: self-attention, multi-head attention, positional encoding, scaled dot-product
   Techniques: layer normalization, residual connections, feed-forward networks

✓ Found 10 similar papers:

[1] BERT: Pre-training of Deep Bidirectional Transformers for Language Understanding
    Source: arXiv | Venue: NAACL 2019 | Published: 2018-10-11
    📊 Relevance: 94.2% | Similarity: 91.5%
    Authors: Devlin, Chang, Lee, Toutanova
    Pre-training contextual representations using Transformer architecture...

[2] GPT-2: Language Models are Unsupervised Multitask Learners
    ...
```

---

## Module 2: Citation Extractor

Extracts and analyzes all papers referenced throughout a research paper, not just from the references section.

### Features

- **Comprehensive Extraction**: Finds citations throughout the entire document
- **Context Analysis**: Identifies why and where each paper was cited
- **Foundational Papers**: Identifies papers that directly influenced the work
- **Citation Frequency**: Counts how many times each paper is mentioned
- **Multiple Output Formats**: Text, JSON, and Markdown

### Usage

```bash
# Extract all citations
./rph citations lib/paper.pdf

# Show only foundational papers
./rph citations lib/paper.pdf --foundational

# Show top 10 most cited papers
./rph citations lib/paper.pdf --top 10

# Output as JSON
./rph citations lib/paper.pdf --format json

# Output as Markdown table
./rph citations lib/paper.pdf --format markdown
```

### How It Works

1. **Full Document Scan**:
   - Scans introduction, related work, methodology, and entire document
   - Not limited to references section

2. **Context Extraction**:
   - Identifies where each paper is cited
   - Extracts the context (why it was cited)

3. **Classification**:
   - Marks foundational papers (directly helped this work)
   - Counts citation frequency
   - Ranks by importance

4. **Structured Output**:
   - Title, authors, year, venue
   - Citation context and frequency
   - Foundational status

### Example Output

#### Text Format (Default)

```
📚 Extracting Citations
   Paper: transformer_paper.pdf
   Mode: All citations

✓ Found 45 citations:

[1] Neural Machine Translation by Jointly Learning to Align and Translate
    Authors: Bahdanau, Cho, Bengio (2014)
    Venue: ICLR 2015
    📊 Cited 8x | FOUNDATIONAL
    Context: Introduced attention mechanisms that this paper builds upon

[2] Long Short-Term Memory
    Authors: Hochreiter, Schmidhuber (1997)
    Venue: Neural Computation
    📊 Cited 3x
    Context: Baseline recurrent architecture for comparison

[3] Learning Phrase Representations using RNN Encoder-Decoder
    ...
```

#### Markdown Format

```markdown
# Citations

| # | Title | Authors | Year | Venue | Cited | Foundational | Context |
|---|-------|---------|------|-------|-------|--------------|---------|
| 1 | Neural Machine Translation... | Bahdanau et al. | 2014 | ICLR | 8x | ✓ | Introduced attention mechanisms... |
| 2 | Long Short-Term Memory | Hochreiter et al. | 1997 | Neural Comp | 3x |  | Baseline recurrent architecture... |
```

#### JSON Format

```json
[
  {
    "title": "Neural Machine Translation by Jointly Learning to Align and Translate",
    "authors": ["Bahdanau", "Cho", "Bengio"],
    "year": "2014",
    "venue": "ICLR 2015",
    "citation_count": 8,
    "is_foundational": true,
    "context": "Introduced attention mechanisms that this paper builds upon"
  },
  ...
]
```

---

## Prerequisites

### 1. Gemini API Key

Both modules require a configured Gemini API key:

```bash
export GEMINI_API_KEY=your_key_here
```

Or configure in `config/config.yaml`:

```yaml
gemini:
  api_key: your_key_here
  model: gemini-2.0-flash-exp
```

### 2. Search Service (for Similar Papers)

The similar papers module requires the search microservice:

```bash
cd services/search-engine
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
python run.py
```

The service runs on `http://localhost:8000` by default.

---

## Use Cases

### Research Exploration

```bash
# Start with a paper you like
./rph similar lib/interesting_paper.pdf --max-results 20 --download

# Extract its foundational references
./rph citations lib/interesting_paper.pdf --foundational

# Find papers similar to those foundational works
./rph similar lib/foundational_paper.pdf
```

### Literature Review

```bash
# Extract all citations from a paper
./rph citations lib/survey_paper.pdf --format markdown > citations.md

# Find papers similar to the most cited works
./rph citations lib/survey_paper.pdf --top 5
# Then search for similar papers to each
```

### Understanding Research Lineage

```bash
# Extract foundational papers
./rph citations lib/paper.pdf --foundational

# Find similar papers to understand the research trajectory
./rph similar lib/paper.pdf --max-results 50
```

---

## API Reference

### Similar Paper Finder

```go
import "archivist/internal/analyzer"

// Create finder
finder := analyzer.NewSimilarPaperFinder(analyzerInstance, "http://localhost:8000")

// Extract essence
essence, err := finder.ExtractEssence(ctx, "paper.pdf")

// Find similar papers
results, err := finder.FindSimilarPapers(ctx, essence, 10)

// Or do both in one call
essence, results, err := finder.FindSimilarPapersFromPDF(ctx, "paper.pdf", 10)
```

### Citation Extractor

```go
import "archivist/internal/analyzer"

// Create extractor
extractor := analyzer.NewCitationExtractor(analyzerInstance)

// Extract all citations
citations, err := extractor.ExtractAllCitations(ctx, "paper.pdf")

// Extract only foundational papers
foundational, err := extractor.ExtractFoundationalPapers(ctx, "paper.pdf")

// Get most cited papers
topCited, err := extractor.ExtractMostCitedPapers(ctx, "paper.pdf", 10)
```

---

## Performance Tips

1. **Batch Processing**: Process multiple papers and cache results
2. **Use Haiku Model**: For faster essence extraction, use `gemini-2.0-flash-exp`
3. **Filter Early**: Use `--foundational` or `--top N` to reduce API calls
4. **Cache Search Results**: The search service caches results for faster repeated queries

---

## Troubleshooting

### "Search service is not running"

```bash
# Check if service is running
curl http://localhost:8000/health

# Start the service
cd services/search-engine && python run.py
```

### "Gemini API key not configured"

```bash
# Set environment variable
export GEMINI_API_KEY=your_key_here

# Or check config file
cat config/config.yaml
```

### No citations found

- Ensure the PDF is readable (not image-based)
- Check that the paper has citations
- Try increasing `max_tokens` in config for longer papers

---

## Future Enhancements

- [ ] Automatic download of cited papers
- [ ] Citation graph visualization
- [ ] Semantic similarity scoring
- [ ] Integration with graph database for citation networks
- [ ] Batch processing of multiple papers
- [ ] Export to BibTeX format
