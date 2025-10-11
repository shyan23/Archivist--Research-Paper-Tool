# Archivist - Research Paper Helper

A powerful CLI tool that converts AI/ML research papers into comprehensive, student-friendly LaTeX reports using Google Gemini AI.

## Features

- 🎨 **Interactive TUI**: Beautiful terminal interface for browsing and processing papers
- 🤖 **AI-Powered Analysis**: Uses Google Gemini API with agentic workflows for deep paper analysis
- 📚 **Student-Friendly**: Generates detailed explanations targeted at CS students
- ⚡ **Parallel Processing**: Process multiple papers concurrently with worker pools
- 🔄 **Smart Deduplication**: MD5 hashing prevents reprocessing of papers
- 📝 **LaTeX Output**: Generates professional LaTeX documents and compiles to PDF
- 🎯 **Multi-Stage Analysis**: Optional agentic workflow with self-reflection and refinement
- 📊 **Progress Tracking**: Real-time logging and status monitoring

## Prerequisites

### Linux/macOS

1. **Go 1.20+**
2. **LaTeX Distribution**:
   ```bash
   sudo apt install texlive-latex-extra latexmk
   ```
3. **Google Gemini API Key**: Get one from [Google AI Studio](https://aistudio.google.com/app/apikey)

### Windows

1. **Go 1.20+** - Download from [https://go.dev/dl/](https://go.dev/dl/)
2. **LaTeX Distribution**:
   - **MiKTeX** (Recommended): [https://miktex.org/download](https://miktex.org/download)
   - **TeX Live**: [https://www.tug.org/texlive/](https://www.tug.org/texlive/)
3. **Google Gemini API Key**: Get one from [Google AI Studio](https://aistudio.google.com/app/apikey)

📘 **Windows users**: See [windows/README_WINDOWS.md](windows/README_WINDOWS.md) for complete installation guide.

## Installation

### Linux/macOS

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/archivist.git
   cd archivist
   ```

2. Set up your API key:
   ```bash
   # Create .env file:
   echo "GEMINI_API_KEY=your_api_key_here" > .env
   ```

3. Install Go dependencies:
   ```bash
   go mod tidy
   ```

4. Build the application:
   ```bash
   go build -o archivist ./cmd/main
   ```

### Windows

**Quick Installation (Automated):**

1. Clone the repository:
   ```powershell
   git clone https://github.com/yourusername/archivist.git
   cd archivist
   ```

2. Run the automated installer:
   ```powershell
   .\windows\install.bat
   ```

**Manual Installation:**

1. Clone and navigate to the repository
2. Create `.env` file with your API key:
   ```
   GEMINI_API_KEY=your_api_key_here
   ```
3. Build:
   ```powershell
   .\windows\build.bat
   ```

📘 **For detailed Windows instructions**, see [windows/README_WINDOWS.md](windows/README_WINDOWS.md)

## Usage

### 🎨 Interactive TUI (Recommended)

Launch the beautiful interactive terminal interface:

**Linux/macOS:**
```bash
./archivist run
```

**Windows:**
```powershell
.\rph.exe run
```

The TUI provides:
- 📚 Browse all papers in your library
- ✅ View processed papers with status
- 📄 Select and process single papers
- 🚀 Batch process all papers
- 🎯 Choose between Fast ⚡ and Quality 🎯 modes
- 🎨 Colorful, intuitive navigation with arrow keys

**Quick Start:**
1. `./archivist run` - Launch TUI
2. Navigate with `↑/↓` or `j/k`
3. Press `Enter` to select an option
4. Press `ESC` to go back, `Q` to quit

See [TUI_GUIDE.md](./TUI_GUIDE.md) for detailed documentation.

### Check Dependencies
```bash
./archivist check
```

### Process Papers

Process a single PDF:
```bash
./archivist process lib/paper.pdf
```

Process all PDFs in a directory:
```bash
./archivist process lib/
```

Process with custom parallel workers:
```bash
./archivist process lib/ --parallel 8
```

Force reprocess already processed papers:
```bash
./archivist process lib/ --force
```

### List Processed Papers
```bash
./archivist list
```

Show unprocessed papers:
```bash
./archivist list --unprocessed
```

### Check Processing Status
```bash
./archivist status lib/paper.pdf
```

### Clean Temporary Files
```bash
./archivist clean
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
├── cmd/main/              # CLI entry point
├── internal/
│   ├── analyzer/         # Gemini API client & analysis logic
│   ├── app/              # Configuration & logging
│   ├── compiler/         # LaTeX compilation
│   ├── generator/        # LaTeX file generation
│   ├── parser/           # PDF metadata extraction
│   ├── storage/          # Metadata persistence
│   ├── tui/              # Interactive Terminal UI (Bubble Tea)
│   ├── ui/               # UI utilities & styling
│   └── worker/           # Worker pool for parallel processing
├── pkg/fileutil/         # File utilities (hashing, etc.)
├── windows/              # Windows-specific files
│   ├── README_WINDOWS.md      # Complete Windows installation guide
│   ├── TROUBLESHOOTING.md     # Windows troubleshooting guide
│   ├── build.bat             # Windows build script
│   ├── install.bat           # Automated Windows installer
│   ├── process-all.bat       # Batch processing script
│   └── Makefile.windows      # Windows Makefile
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

## Testing

Archivist includes a comprehensive test suite covering unit tests, integration tests, and end-to-end workflows.

### Quick Start

```bash
# Run all tests
make test

# Run unit tests only (fast)
make test-unit

# Run tests with coverage report
make test-coverage

# Run quick tests during development
make test-quick
```

### Test Commands

#### Using Make

```bash
make test              # Run all tests
make test-unit         # Run unit tests only
make test-integration  # Run integration tests only
make test-verbose      # Run tests with verbose output
make test-coverage     # Generate coverage report (HTML)
make test-quick        # Quick tests for development
make bench             # Run benchmarks
```

#### Using test.sh Script

The `test.sh` script provides more control:

```bash
./test.sh all          # Run all tests
./test.sh unit         # Unit tests only
./test.sh integration  # Integration tests only
./test.sh coverage     # Coverage report with browser preview
./test.sh bench        # Benchmarks
./test.sh quick        # Quick development tests
./test.sh verbose      # Verbose output
./test.sh specific TestName  # Run specific test
./test.sh watch        # Watch mode - rerun on changes
./test.sh clean        # Clean test artifacts
```

#### Using Go directly

```bash
# Run all tests
go test ./...

# Run tests with race detector
go test -race ./...

# Run specific package tests
go test ./pkg/fileutil

# Run specific test
go test -run TestComputeFileHash ./pkg/fileutil

# Verbose output
go test -v ./...

# With coverage
go test -cover ./...
```

### Test Organization

```
archivist/
├── pkg/
│   └── fileutil/
│       └── hash_test.go           # File hashing & discovery tests
├── internal/
│   ├── storage/
│   │   └── metadata_test.go       # Metadata storage & deduplication
│   ├── parser/
│   │   └── pdf_parser_test.go     # PDF parsing tests
│   ├── generator/
│   │   └── latex_generator_test.go # LaTeX generation tests
│   ├── compiler/
│   │   └── latex_compiler_test.go  # PDF compilation tests
│   ├── analyzer/
│   │   └── analyzer_test.go       # LLM analysis tests
│   ├── worker/
│   │   └── pool_test.go           # Parallel processing tests
│   ├── batch_test.go              # Batch processing tests
│   ├── integration_test.go        # End-to-end integration tests
│   ├── cli_test.go                # CLI command tests
│   └── testhelpers/
│       └── helpers.go             # Shared test utilities
└── testdata/                      # Test fixtures & sample data
```

### Test Coverage

View current test coverage:

```bash
make test-coverage
# Opens coverage.html in your browser
```

Coverage reports include:
- ✅ **Per-package coverage** - See which packages need more tests
- ✅ **Line-by-line coverage** - Identify untested code paths
- ✅ **Function coverage** - Track tested vs untested functions

### Running Specific Test Suites

**Unit Tests** (Fast, isolated tests):
```bash
make test-unit
# Tests: fileutil, storage, parser, generator, compiler, analyzer
```

**Integration Tests** (End-to-end workflows):
```bash
make test-integration
# Tests: Complete paper processing workflows
```

**Benchmarks** (Performance testing):
```bash
make bench
# Benchmarks: Hash computation, file discovery, parsing, etc.
```

### Continuous Integration

For CI/CD pipelines:

```bash
# Run tests with race detector and coverage
go test -race -timeout 10m -coverprofile=coverage.out ./...

# Check coverage threshold
go tool cover -func=coverage.out | grep total
```

### Watch Mode (Development)

Auto-run tests when files change:

```bash
./test.sh watch
# Requires: entr (install with: sudo apt install entr)
```

### Test Configuration

Tests use temporary directories and mock configurations. No manual setup required!

Key test helpers (in `internal/testhelpers`):
- `TestConfig()` - Creates isolated test configuration
- `CreateTestPDF()` - Generates test PDF files
- `CreateTestLaTeX()` - Generates test LaTeX files
- `ComputeTestFileHash()` - Hash computation for tests

### Troubleshooting Tests

**Tests fail with "API key" errors:**
- Unit tests use mocks - no API key needed
- Integration tests may require `GEMINI_API_KEY` environment variable

**Tests timeout:**
- Increase timeout: `go test -timeout 10m ./...`
- Use quick tests: `make test-quick`

**Coverage report not opening:**
- Manually open `coverage.html` in your browser
- Or run: `xdg-open coverage.html` (Linux) / `open coverage.html` (Mac)

## CI/CD Pipeline

✅ **Fully automated CI/CD pipeline implemented with GitHub Actions!**

### What's Included

🔄 **Continuous Integration (CI)**
- ✅ Automated testing on every push and pull request
- ✅ Multi-version Go testing (1.20, 1.21, 1.22)
- ✅ Code linting with golangci-lint
- ✅ Race condition detection
- ✅ Test coverage reporting to Codecov
- ✅ Security scanning with Trivy and gosec
- ✅ Docker image building

📦 **Continuous Deployment (CD)**
- ✅ Automatic releases on version tags
- ✅ Multi-platform binary builds (Linux, macOS, Windows)
- ✅ Multi-architecture support (amd64, arm64)
- ✅ Docker images published to Docker Hub and GitHub Container Registry
- ✅ Automated changelog generation
- ✅ GitHub Releases with artifacts

🤖 **Automation**
- ✅ Dependabot for automatic dependency updates
- ✅ Security vulnerability scanning
- ✅ Code quality checks

### Quick Start

**No manual setup required!** The CI/CD pipeline runs automatically when you:

1. **Push to GitHub** - CI pipeline runs on every commit
2. **Create a Pull Request** - Automated testing and validation
3. **Push a version tag** - Full release pipeline with binaries and Docker images

```bash
# Create and push a release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

### GitHub Secrets Setup

For full functionality, configure these secrets in your repository (`Settings` → `Secrets and variables` → `Actions`):

| Secret | Purpose | Required |
|--------|---------|----------|
| `GEMINI_API_KEY` | Run integration tests | Optional |
| `CODECOV_TOKEN` | Upload coverage reports | Optional |
| `DOCKER_USERNAME` | Publish to Docker Hub | Optional |
| `DOCKER_PASSWORD` | Docker Hub authentication | Optional |

**Note:** The pipeline works without secrets, but some features will be skipped.

### Viewing CI/CD Status

Add these badges to show pipeline status:

```markdown
![CI](https://github.com/YOUR_USERNAME/archivist/workflows/CI/badge.svg)
![Release](https://github.com/YOUR_USERNAME/archivist/workflows/Release/badge.svg)
```

### Documentation

- 📖 **[CI/CD Setup Guide](.github/SETUP_GUIDE.md)** - Detailed configuration instructions
- 📊 **[Test Coverage Report](tests/TEST_COVERAGE.md)** - Test sufficiency assessment

### Workflow Files

- `.github/workflows/ci.yml` - Main CI pipeline
- `.github/workflows/release.yml` - Release automation
- `.github/dependabot.yml` - Dependency updates
- `.golangci.yml` - Linting configuration
- `.goreleaser.yml` - Release configuration

## Troubleshooting

### General Issues

**LaTeX compilation fails:**
- Linux/macOS: Ensure `texlive-latex-extra` is installed
- Windows: Ensure MiKTeX or TeX Live is installed and in PATH
- Check `.metadata/processing.log` for details

**Gemini API errors:**
- Verify API key in `.env`
- Check quota limits at Google AI Studio
- Try reducing `max_workers` in config

**Out of memory:**
- Reduce `processing.max_workers` in config
- Process papers in smaller batches

**CI/CD Pipeline Issues:**
- See [CI/CD Setup Guide](.github/SETUP_GUIDE.md) for troubleshooting
- Check Actions tab in GitHub for detailed logs
- Verify secrets are configured correctly

### Windows-Specific Issues

📘 **Windows users**: For comprehensive troubleshooting, see [windows/TROUBLESHOOTING.md](windows/TROUBLESHOOTING.md)

Common Windows issues:
- **"pdflatex is not recognized"**: Add LaTeX to PATH (see troubleshooting guide)
- **"Access denied" errors**: Run as Administrator or check file permissions
- **Slow performance**: Add `rph.exe` to antivirus exclusions
- **Path too long errors**: Place project closer to root (e.g., `C:\archivist`)

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
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
