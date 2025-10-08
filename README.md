# Archivist - Research Paper Helper

A powerful CLI tool that converts AI/ML research papers into comprehensive, student-friendly LaTeX reports using Google Gemini AI.

## Features

- 🤖 **AI-Powered Analysis**: Uses Google Gemini API with agentic workflows for deep paper analysis
- 📚 **Student-Friendly**: Generates detailed explanations targeted at CS students
- ⚡ **Parallel Processing**: Process multiple papers concurrently with worker pools
- 🔄 **Smart Deduplication**: MD5 hashing prevents reprocessing of papers
- 📝 **LaTeX Output**: Generates professional LaTeX documents and compiles to PDF
- 🎯 **Multi-Stage Analysis**: Optional agentic workflow with self-reflection and refinement
- 📊 **Progress Tracking**: Real-time logging and status monitoring

## Prerequisites

1. **Go 1.20+**
2. **LaTeX Distribution**:
   ```bash
   sudo apt install texlive-latex-extra latexmk
   ```
3. **Google Gemini API Key**: Get one from [Google AI Studio](https://aistudio.google.com/app/apikey)

## Installation

1. Clone the repository:
   ```bash
   cd /home/shyan/Desktop/Code/Archivist
   ```

2. Set up your API key:
   ```bash
   # .env file already exists with:
   GEMINI_API_KEY=your_api_key_here
   ```

3. Install Go dependencies:
   ```bash
   go mod tidy
   ```

4. Build the application:
   ```bash
   go build -o rph ./cmd/rph
   ```

## Usage

### Check Dependencies
```bash
./rph check
```

### Process Papers

Process a single PDF:
```bash
./rph process lib/paper.pdf
```

Process all PDFs in a directory:
```bash
./rph process lib/
```

Process with custom parallel workers:
```bash
./rph process lib/ --parallel 8
```

Force reprocess already processed papers:
```bash
./rph process lib/ --force
```

### List Processed Papers
```bash
./rph list
```

Show unprocessed papers:
```bash
./rph list --unprocessed
```

### Check Processing Status
```bash
./rph status lib/paper.pdf
```

### Clean Temporary Files
```bash
./rph clean
```

## Configuration

Edit `config/config.yaml` to customize:

```yaml
processing:
  max_workers: 4              # Parallel processing workers

gemini:
  model: "gemini-2.0-flash"   # Gemini model to use

  agentic:
    enabled: true             # Enable multi-stage agentic workflow
    max_iterations: 3         # Self-reflection iterations
    self_reflection: true     # Enable self-critique

    stages:
      methodology_analysis:
        model: "gemini-2.5-pro"  # Use Pro for complex reasoning
        temperature: 1
```

## Project Structure

```
.
├── cmd/rph/              # CLI entry point
├── internal/
│   ├── analyzer/         # Gemini API client & analysis logic
│   ├── app/              # Configuration & logging
│   ├── compiler/         # LaTeX compilation
│   ├── generator/        # LaTeX file generation
│   ├── parser/           # PDF metadata extraction
│   ├── storage/          # Metadata persistence
│   └── worker/           # Worker pool for parallel processing
├── pkg/fileutil/         # File utilities (hashing, etc.)
├── config/               # Configuration files
├── lib/                  # Input PDFs
├── tex_files/            # Generated LaTeX files
├── reports/              # Final PDF reports
└── .metadata/            # Processing metadata & logs
```

## How It Works

1. **PDF Analysis**: Uses Gemini's multimodal capabilities to analyze PDF content
2. **Metadata Extraction**: Extracts title, authors, abstract from the paper
3. **Agentic Workflow** (if enabled):
   - Stage 1: Deep methodology analysis with Gemini Pro
   - Stage 2: Self-reflection and refinement
   - Stage 3: LaTeX syntax validation
4. **LaTeX Generation**: Creates comprehensive student-friendly document
5. **Compilation**: Compiles LaTeX to PDF using pdflatex/latexmk
6. **Metadata Tracking**: Stores processing status for deduplication

## Output Format

Generated LaTeX reports include:

- **Executive Summary**: Quick overview for students
- **Problem Statement**: What problem the paper solves and why it matters
- **Prerequisites**: Specific concepts needed (not vague like "linear algebra")
- **Detailed Methodology**: Step-by-step explanations with math formulations
- **Key Breakthrough**: The "WOW moment" explained clearly
- **Experimental Results**: Quantitative improvements with specific numbers
- **Conclusion**: Takeaways, impact, and future directions

## Example Output

For a paper titled "Attention Is All You Need":

```
lib/attention_is_all_you_need.pdf
  ↓ (Gemini Analysis)
tex_files/Attention_Is_All_You_Need.tex
  ↓ (pdflatex)
reports/Attention_Is_All_You_Need.pdf  ✅
```

## Troubleshooting

**LaTeX compilation fails:**
- Ensure `texlive-latex-extra` is installed
- Check `.metadata/processing.log` for details

**Gemini API errors:**
- Verify API key in `.env`
- Check quota limits at Google AI Studio
- Try reducing `max_workers` in config

**Out of memory:**
- Reduce `processing.max_workers` in config
- Process papers in smaller batches

## Advanced Features

### Agentic Workflow

Enable sophisticated multi-stage analysis:
```yaml
gemini:
  agentic:
    enabled: true
    multi_stage_analysis: true
    stages:
      methodology_analysis:
        model: "gemini-2.5-pro"  # More powerful for complex reasoning
```

### Custom Models Per Stage

Use different models for different analysis stages:
- `gemini-2.0-flash`: Fast metadata extraction
- `gemini-2.5-pro`: Deep methodology analysis
- Configurable thinking budget for chain-of-thought reasoning

## License

MIT

## Contributing

Contributions welcome! Please check existing issues or create new ones.

## Acknowledgments

Built with:
- [Google Gemini API](https://ai.google.dev/)
- [Cobra CLI](https://github.com/spf13/cobra)
- [Viper Config](https://github.com/spf13/viper)
