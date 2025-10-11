# Archivist - Windows Installation Guide

This guide provides complete instructions for Windows users to install and use Archivist.

## Prerequisites

### 1. Install Go

1. Download Go 1.20+ from [https://go.dev/dl/](https://go.dev/dl/)
2. Run the installer (`.msi` file)
3. Verify installation by opening Command Prompt or PowerShell:
   ```powershell
   go version
   ```

### 2. Install LaTeX Distribution

**Option A: MiKTeX (Recommended for Windows)**

1. Download MiKTeX from [https://miktex.org/download](https://miktex.org/download)
2. Run the installer
3. During installation:
   - Choose "Install missing packages on-the-fly: Yes"
   - This allows MiKTeX to automatically download required packages
4. After installation, update MiKTeX:
   ```powershell
   miktex-console
   ```
   - Go to "Updates" tab and click "Check for updates"
   - Install all available updates

**Option B: TeX Live**

1. Download TeX Live from [https://www.tug.org/texlive/windows.html](https://www.tug.org/texlive/windows.html)
2. Run `install-tl-windows.exe`
3. Choose "Full installation" (requires ~7GB)
4. Installation may take 1-2 hours

### 3. Get Google Gemini API Key

1. Visit [Google AI Studio](https://aistudio.google.com/app/apikey)
2. Create a new API key
3. Save it securely - you'll need it later

## Installation

### Method 1: Using PowerShell (Recommended)

1. **Clone the repository:**
   ```powershell
   cd C:\Users\YourUsername\Documents
   git clone https://github.com/yourusername/archivist.git
   cd archivist
   ```

2. **Set up environment variables:**

   Create a `.env` file in the project root:
   ```powershell
   notepad .env
   ```

   Add your API key:
   ```
   GEMINI_API_KEY=your_api_key_here
   ```

   Save and close the file.

3. **Install dependencies:**
   ```powershell
   go mod tidy
   ```

4. **Build the application:**
   ```powershell
   .\windows\build.bat
   ```

   This creates `rph.exe` in the project root.

5. **Verify installation:**
   ```powershell
   .\rph.exe check
   ```

### Method 2: Using Windows Batch Script

Simply double-click `windows\install.bat` and follow the prompts.

## Usage

### Launch Interactive TUI

```powershell
.\rph.exe run
```

The TUI provides a beautiful interface with:
- Browse all papers in your library
- View processed papers
- Select and process papers
- Choose between Fast and Quality modes

**Navigation:**
- Use Arrow keys or `j/k` to navigate
- Press `Enter` to select
- Press `ESC` to go back, `Q` to quit

### Command Line Usage

**Check dependencies:**
```powershell
.\rph.exe check
```

**Process a single paper:**
```powershell
.\rph.exe process lib\paper.pdf
```

**Process all papers in a directory:**
```powershell
.\rph.exe process lib\
```

**Process with specific number of workers:**
```powershell
.\rph.exe process lib\ --parallel 4
```

**Force reprocess already processed papers:**
```powershell
.\rph.exe process lib\ --force
```

**List processed papers:**
```powershell
.\rph.exe list
```

**Show unprocessed papers:**
```powershell
.\rph.exe list --unprocessed
```

**Check status of a specific paper:**
```powershell
.\rph.exe status lib\paper.pdf
```

**Clean temporary files:**
```powershell
.\rph.exe clean
```

## Directory Structure

After installation, your project will look like:

```
archivist\
├── rph.exe                 # Main executable
├── .env                    # Your API key (keep private!)
├── config\
│   └── config.yaml        # Configuration file
├── lib\                   # Place your PDF papers here
├── tex_files\             # Generated LaTeX files
├── reports\               # Final PDF reports (output)
└── .metadata\             # Processing metadata
```

## Configuration

Edit `config\config.yaml` to customize settings:

```yaml
input_dir: "./lib"
tex_output_dir: "./tex_files"
report_output_dir: "./reports"
metadata_dir: "./.metadata"

processing:
  max_workers: 4              # Adjust based on your CPU cores

gemini:
  model: "gemini-2.0-flash"

  agentic:
    enabled: true
    max_iterations: 3
    self_reflection: true

latex:
  compiler: "pdflatex"        # or "xelatex", "lualatex"
  engine: "latexmk"
  clean_aux: true
```

## Troubleshooting

### "pdflatex is not recognized as an internal or external command"

**Solution:** LaTeX is not in your PATH.

1. Open System Properties → Advanced → Environment Variables
2. Add to PATH:
   - MiKTeX: `C:\Program Files\MiKTeX\miktex\bin\x64`
   - TeX Live: `C:\texlive\2024\bin\windows`
3. Restart Command Prompt/PowerShell

### "rph.exe: The system cannot find the path specified"

**Solution:** You're not in the correct directory.

```powershell
cd C:\Users\YourUsername\Documents\archivist
.\rph.exe check
```

### "Failed to load API key"

**Solution:**
1. Verify `.env` file exists in project root
2. Check that it contains: `GEMINI_API_KEY=your_actual_key`
3. No spaces around the `=` sign
4. No quotes around the key

### LaTeX compilation errors

**For MiKTeX:**
1. Open MiKTeX Console
2. Go to Settings → General
3. Set "Install missing packages" to "Always"
4. Try processing again

**For TeX Live:**
```powershell
tlmgr update --self --all
```

### Out of memory errors

**Solution:** Reduce parallel workers in `config\config.yaml`:

```yaml
processing:
  max_workers: 2  # Lower this value
```

### Gemini API quota exceeded

**Solution:**
1. Check your quota at [Google AI Studio](https://aistudio.google.com/)
2. Reduce `max_workers` to process papers more slowly
3. Wait for quota to reset (usually daily)

## Windows-Specific Notes

### File Paths

- Windows uses backslashes (`\`) in paths
- The tool automatically handles path conversions
- You can use forward slashes (`/`) in commands - they work too!

### Performance Tips

1. **Antivirus:** Add `rph.exe` to antivirus exclusions for faster processing
2. **Power Plan:** Use "High Performance" power plan for batch processing
3. **Workers:** Set `max_workers` to match your CPU cores (check Task Manager)

### Running from Any Directory

To use `rph` from anywhere:

1. Add Archivist directory to PATH:
   - Open System Properties → Environment Variables
   - Edit PATH variable
   - Add: `C:\Users\YourUsername\Documents\archivist`
   - Click OK

2. Now you can run from any directory:
   ```powershell
   cd C:\MyPapers
   rph.exe process .
   ```

## Advanced: Using Windows Terminal

For a better experience, use [Windows Terminal](https://apps.microsoft.com/store/detail/windows-terminal/9N0DX20HK701):

1. Install from Microsoft Store
2. Open Windows Terminal
3. Navigate to Archivist directory
4. Run `.\rph.exe run` for beautiful TUI colors

## Batch Processing Script

For Windows users who want to automate processing, use the included batch script:

```powershell
.\windows\process-all.bat
```

This script:
- Checks dependencies
- Processes all PDFs in `lib\` directory
- Shows progress
- Generates a summary report

## Getting Help

If you encounter issues:

1. Run `.\rph.exe check` to verify installation
2. Check logs in `.metadata\processing.log`
3. Create an issue on GitHub with:
   - Windows version (run `winver`)
   - Go version (`go version`)
   - LaTeX distribution and version
   - Error message and logs

## Updating

To update Archivist:

```powershell
git pull origin main
go mod tidy
.\windows\build.bat
```

## Uninstalling

To remove Archivist:

1. Delete the project directory
2. Remove from PATH (if added)
3. Optionally uninstall MiKTeX/TeX Live

Your processed papers in `reports\` directory will remain unless you delete them manually.

## Performance Benchmarks

On a typical Windows machine (Intel i5, 8GB RAM):

- Single paper: 2-5 minutes
- Batch (10 papers, 4 workers): 15-30 minutes
- Fast mode: ~40% faster than Quality mode

## FAQ

**Q: Can I use WSL (Windows Subsystem for Linux)?**

A: Yes! If you have WSL installed, you can use the Linux installation method instead. However, the native Windows version is recommended for better performance.

**Q: Does this work on Windows 11?**

A: Yes! All features are compatible with Windows 10 and 11.

**Q: Do I need administrator privileges?**

A: Only for installing Go and LaTeX. Running `rph.exe` does not require admin rights.

**Q: Can I process papers offline?**

A: No, Gemini API requires internet connection. However, LaTeX compilation works offline.

**Q: Where are temporary files stored?**

A: In the project's `tex_files\` directory. Use `rph.exe clean` to remove them.

## Next Steps

1. Place your research papers (PDFs) in the `lib\` directory
2. Run `.\rph.exe run` to launch the interactive TUI
3. Select papers and choose your preferred processing mode
4. Find your reports in the `reports\` directory

Happy researching! 🎓📚
