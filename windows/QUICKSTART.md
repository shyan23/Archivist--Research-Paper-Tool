# Windows Quick Start Guide

Get Archivist running on Windows in 5 minutes!

## Prerequisites Checklist

Before starting, ensure you have:

- [ ] **Go 1.20+** - [Download](https://go.dev/dl/)
- [ ] **Git** - [Download](https://git-scm.com/download/win)
- [ ] **LaTeX** (MiKTeX or TeX Live) - [MiKTeX Download](https://miktex.org/download)
- [ ] **Gemini API Key** - [Get one here](https://aistudio.google.com/app/apikey)

## 5-Minute Installation

### Step 1: Open PowerShell or Command Prompt

Press `Win + X` and select "Windows PowerShell" or "Command Prompt"

### Step 2: Clone the Repository

```powershell
cd C:\Users\YourUsername\Documents
git clone https://github.com/yourusername/archivist.git
cd archivist
```

### Step 3: Run Automated Installer

```powershell
.\windows\install.bat
```

This will:
- ✅ Check Go installation
- ✅ Check LaTeX installation
- ✅ Install dependencies
- ✅ Build the executable
- ✅ Set up configuration

### Step 4: Add Your API Key

When prompted, enter your Gemini API key, or add it manually:

```powershell
notepad .env
```

Add:
```
GEMINI_API_KEY=your_api_key_here
```

Save and close.

### Step 5: Verify Installation

```powershell
.\rph.exe check
```

You should see:
```
✅ All dependencies installed
  📦 LaTeX Compiler:  pdflatex
  🔧 Workers:         4
  🤖 AI Model:        gemini-2.0-flash
```

## First Run

### Add Papers

Place your PDF research papers in the `lib\` folder:

```powershell
# Create lib folder if it doesn't exist
mkdir lib

# Copy your papers
copy C:\Downloads\paper.pdf lib\
```

### Launch Interactive Mode

```powershell
.\rph.exe run
```

This opens a beautiful terminal UI where you can:
- Browse papers
- Select processing mode (Fast ⚡ or Quality 🎯)
- Process papers interactively

### Or Use Command Line

Process all papers:
```powershell
.\rph.exe process lib\
```

Process a single paper:
```powershell
.\rph.exe process lib\paper.pdf
```

## What to Expect

### Processing Time

For a typical research paper:
- **Fast Mode**: 2-4 minutes
- **Quality Mode**: 4-8 minutes

### Output Location

After processing, you'll find:

```
archivist\
├── tex_files\          # LaTeX source files
│   └── Paper_Title.tex
└── reports\            # 🎉 Your final PDF reports!
    └── Paper_Title.pdf
```

### View Your Report

```powershell
# Open the report in your default PDF viewer
start reports\Paper_Title.pdf
```

## Common Commands

| Command | Description |
|---------|-------------|
| `.\rph.exe run` | Launch interactive TUI |
| `.\rph.exe process lib\` | Process all papers |
| `.\rph.exe list` | Show processed papers |
| `.\rph.exe status lib\paper.pdf` | Check paper status |
| `.\rph.exe clean` | Clean temporary files |
| `.\rph.exe check` | Verify dependencies |

## Quick Troubleshooting

### "Go is not recognized"
- **Fix**: Install Go from [https://go.dev/dl/](https://go.dev/dl/)
- Restart PowerShell after installation

### "pdflatex is not recognized"
- **Fix**: Install MiKTeX or TeX Live
- Add to PATH: `C:\Program Files\MiKTeX\miktex\bin\x64`
- Restart PowerShell

### "Failed to load API key"
- **Fix**: Check `.env` file exists and contains valid API key
- Format: `GEMINI_API_KEY=your_key` (no quotes, no spaces)

### "API quota exceeded"
- **Fix**: Reduce workers in `config\config.yaml`:
  ```yaml
  processing:
    max_workers: 2
  ```

### Papers processing slowly?
- Use **Fast Mode** (40% faster)
- Reduce parallel workers
- Add `rph.exe` to antivirus exclusions

## Tips for Best Results

### 1. Choose the Right Mode

**Fast Mode** (⚡):
- 2-4 minutes per paper
- Good quality
- Recommended for most papers

**Quality Mode** (🎯):
- 4-8 minutes per paper
- Best quality with self-reflection
- Use for complex/important papers

### 2. Batch Processing

Process multiple papers overnight:
```powershell
# Use the batch script
.\windows\process-all.bat
```

### 3. Organize Your Papers

```
lib\
├── computer-vision\
│   ├── paper1.pdf
│   └── paper2.pdf
├── nlp\
│   └── paper3.pdf
└── networking\
    └── paper4.pdf
```

Then process by category:
```powershell
.\rph.exe process lib\computer-vision\
```

### 4. Monitor Progress

Watch real-time logs:
```powershell
Get-Content .metadata\processing.log -Wait
```

## Performance Tips

### Speed Up Processing

1. **Use SSD** instead of HDD
2. **Add to antivirus exclusions**:
   - `rph.exe`
   - `tex_files\` folder
   - `reports\` folder
3. **Increase workers** (if you have powerful CPU):
   ```yaml
   # In config\config.yaml
   processing:
     max_workers: 8  # Match your CPU cores
   ```
4. **Use Fast Mode** for most papers
5. **Close other applications** during processing

### Optimize for Your Hardware

| Hardware | Recommended Workers | Mode |
|----------|-------------------|------|
| Basic (i3, 4GB) | 2 workers | Fast |
| Standard (i5, 8GB) | 4 workers | Fast/Quality |
| High-end (i7+, 16GB+) | 6-8 workers | Quality |

## Next Steps

### Learn More

- 📖 [Complete Windows Guide](README_WINDOWS.md)
- 🔧 [Troubleshooting Guide](TROUBLESHOOTING.md)
- 🎨 [TUI Guide](../TUI_GUIDE.md) (if available)
- ⚙️ [Configuration Guide](../config/README.md) (if available)

### Customize Configuration

Edit `config\config.yaml` to:
- Change AI models
- Adjust parallel workers
- Enable/disable agentic workflow
- Customize LaTeX compilation

### Advanced Features

Try these once you're comfortable:

**Agentic Workflow** (multi-stage analysis):
```yaml
gemini:
  agentic:
    enabled: true
    self_reflection: true
```

**Custom Models**:
```yaml
gemini:
  model: "gemini-1.5-pro"  # More powerful, slower
```

**Automated Processing**:
Set up Windows Task Scheduler to process papers automatically!

## Getting Help

### Built-in Help

```powershell
.\rph.exe --help
.\rph.exe process --help
```

### Documentation

- **Windows Setup**: `windows\README_WINDOWS.md`
- **Troubleshooting**: `windows\TROUBLESHOOTING.md`
- **Main README**: `README.md`

### Report Issues

If something's not working:

1. Run diagnostics:
   ```powershell
   .\rph.exe check
   systeminfo | findstr /B /C:"OS Name" /C:"OS Version"
   go version
   pdflatex --version
   ```

2. Check logs:
   ```powershell
   type .metadata\processing.log
   ```

3. Create GitHub issue with above information

## Success!

You're all set! 🎉

Now place your papers in `lib\` and run:
```powershell
.\rph.exe run
```

Happy researching! 📚✨

---

**Pro Tip**: Bookmark this page for quick reference!

**Need help?** Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) or create a GitHub issue.
