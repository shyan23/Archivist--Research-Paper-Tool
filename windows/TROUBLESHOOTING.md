# Windows Troubleshooting Guide

This guide covers common issues Windows users might encounter and their solutions.

## Table of Contents

1. [Installation Issues](#installation-issues)
2. [LaTeX Problems](#latex-problems)
3. [API and Network Issues](#api-and-network-issues)
4. [Performance Issues](#performance-issues)
5. [File Path Issues](#file-path-issues)
6. [Build Errors](#build-errors)

---

## Installation Issues

### Problem: "go is not recognized as an internal or external command"

**Cause:** Go is not installed or not in PATH.

**Solution:**
1. Download Go from [https://go.dev/dl/](https://go.dev/dl/)
2. Run the `.msi` installer
3. Verify installation:
   ```powershell
   go version
   ```
4. If still not working, add to PATH manually:
   - Open System Properties → Advanced → Environment Variables
   - Add `C:\Go\bin` to PATH
   - Restart Command Prompt/PowerShell

### Problem: "git is not recognized"

**Cause:** Git is not installed.

**Solution:**
1. Download Git from [https://git-scm.com/download/win](https://git-scm.com/download/win)
2. Install with default options
3. Restart Command Prompt/PowerShell

### Problem: Build fails with "cannot find module"

**Cause:** Dependencies not downloaded.

**Solution:**
```powershell
go mod download
go mod tidy
```

---

## LaTeX Problems

### Problem: "pdflatex is not recognized"

**Cause:** LaTeX not installed or not in PATH.

**Solution:**

**For MiKTeX:**
1. Install MiKTeX from [https://miktex.org/download](https://miktex.org/download)
2. Add to PATH:
   ```
   C:\Program Files\MiKTeX\miktex\bin\x64
   ```
3. Restart terminal

**For TeX Live:**
1. Install from [https://www.tug.org/texlive/](https://www.tug.org/texlive/)
2. Add to PATH:
   ```
   C:\texlive\2024\bin\windows
   ```
3. Restart terminal

### Problem: "LaTeX Error: File `xxx.sty' not found"

**Cause:** Missing LaTeX package.

**Solution for MiKTeX:**
1. Open MiKTeX Console
2. Go to Settings → General
3. Set "Install missing packages on-the-fly" to "Always"
4. Try processing again

**Solution for TeX Live:**
```powershell
tlmgr install <package-name>
# Or update all packages:
tlmgr update --all
```

### Problem: "LaTeX compilation timeout"

**Cause:** Large paper or slow compilation.

**Solution:**
1. Check LaTeX logs in `tex_files\` directory
2. Reduce parallel workers in `config\config.yaml`:
   ```yaml
   processing:
     max_workers: 2
   ```
3. Try compiling individual `.tex` files manually:
   ```powershell
   cd tex_files
   pdflatex paper.tex
   ```

### Problem: Auxiliary files not cleaning

**Cause:** Files locked by other process.

**Solution:**
1. Close any PDF viewers
2. Run:
   ```powershell
   .\rph.exe clean
   ```
3. If still locked, manually delete from `tex_files\`

---

## API and Network Issues

### Problem: "Failed to load API key"

**Cause:** Missing or invalid `.env` file.

**Solution:**
1. Check `.env` exists in project root:
   ```powershell
   type .env
   ```
2. Should contain:
   ```
   GEMINI_API_KEY=your_actual_key_here
   ```
3. No spaces around `=`
4. No quotes around key
5. Restart `rph.exe` after editing

### Problem: "API quota exceeded" or "Resource exhausted"

**Cause:** Reached Gemini API quota limit.

**Solution:**
1. Check quota at [Google AI Studio](https://aistudio.google.com/)
2. Reduce processing rate:
   ```yaml
   processing:
     max_workers: 1  # Process one at a time
   ```
3. Wait for quota reset (usually daily)
4. Consider upgrading API plan

### Problem: "Failed to connect to Gemini API"

**Cause:** Network connectivity issues.

**Solution:**
1. Check internet connection
2. Test API key:
   ```powershell
   curl -H "Content-Type: application/json" -d "{\"contents\":[{\"parts\":[{\"text\":\"test\"}]}]}" "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent?key=YOUR_API_KEY"
   ```
3. Check firewall settings
4. Try different network (VPN, mobile hotspot)

### Problem: SSL/TLS certificate errors

**Cause:** Corporate proxy or antivirus interference.

**Solution:**
1. Temporarily disable antivirus/proxy
2. Or configure Go to use system certificates:
   ```powershell
   $env:SSL_CERT_FILE = "C:\path\to\cert.pem"
   ```

---

## Performance Issues

### Problem: Processing is very slow

**Possible causes and solutions:**

**1. Too few workers:**
```yaml
# In config\config.yaml
processing:
  max_workers: 8  # Increase based on CPU cores
```

**2. Antivirus scanning:**
- Add `rph.exe` to antivirus exclusions
- Exclude `tex_files\` and `reports\` directories

**3. Power plan:**
- Set to "High Performance" in Windows Power Options

**4. Disk I/O:**
- Use SSD instead of HDD if possible
- Close other disk-intensive applications

### Problem: High memory usage

**Cause:** Processing too many papers simultaneously.

**Solution:**
1. Reduce workers:
   ```yaml
   processing:
     max_workers: 2
   ```
2. Process in smaller batches
3. Close other applications

### Problem: Computer becomes unresponsive

**Cause:** CPU overload.

**Solution:**
1. Reduce parallel workers to 1-2
2. Process papers one at a time:
   ```powershell
   .\rph.exe process lib\paper1.pdf
   .\rph.exe process lib\paper2.pdf
   ```

---

## File Path Issues

### Problem: "The system cannot find the path specified"

**Cause:** Incorrect path format or non-existent directory.

**Solution:**
1. Use Windows path separators:
   ```powershell
   .\rph.exe process lib\paper.pdf  # Correct
   .\rph.exe process lib/paper.pdf  # Also works
   ```
2. Check directory exists:
   ```powershell
   dir lib
   ```
3. Use absolute paths if needed:
   ```powershell
   .\rph.exe process C:\Users\Name\Documents\archivist\lib\paper.pdf
   ```

### Problem: Files with special characters fail

**Cause:** Filenames with special characters.

**Solution:**
1. Rename PDFs to avoid:
   - Special characters: `<>:"|?*`
   - Unicode characters
2. Use simple names: `paper_title.pdf`

### Problem: "Access denied" errors

**Cause:** Insufficient permissions.

**Solution:**
1. Run Command Prompt/PowerShell as Administrator
2. Check file permissions:
   ```powershell
   icacls lib\paper.pdf
   ```
3. Grant full control:
   ```powershell
   icacls lib\paper.pdf /grant Users:F
   ```

---

## Build Errors

### Problem: "build constraints exclude all Go files"

**Cause:** Wrong GOOS/GOARCH environment variables.

**Solution:**
```powershell
# Clear environment variables
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -o rph.exe ./cmd/main
```

### Problem: "undefined: syscall.Dup2"

**Cause:** Linux-specific code in Windows build.

**Solution:**
This is already handled in the codebase. If you see this:
1. Update Go version: `go version` (need 1.20+)
2. Clean and rebuild:
   ```powershell
   go clean -cache
   .\windows\build.bat
   ```

### Problem: "cannot load such file -- xxxx"

**Cause:** Missing dependencies.

**Solution:**
```powershell
go mod download
go mod verify
go mod tidy
```

---

## Advanced Troubleshooting

### Enable Debug Logging

Edit `config\config.yaml`:
```yaml
logging:
  level: debug
  file: .metadata\debug.log
```

Then check logs:
```powershell
type .metadata\debug.log
```

### Test Individual Components

**Test PDF parsing:**
```powershell
.\rph.exe status lib\test.pdf
```

**Test LaTeX compilation:**
```powershell
cd tex_files
pdflatex -interaction=nonstopmode test.tex
```

**Test API connection:**
Create `test_api.go`:
```go
package main

import (
    "fmt"
    "os"
    "github.com/google/generative-ai-go/genai"
    "context"
)

func main() {
    ctx := context.Background()
    client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    defer client.Close()
    fmt.Println("API connection successful!")
}
```

Run:
```powershell
go run test_api.go
```

### Check System Resources

**CPU usage:**
```powershell
Get-Process rph | Select-Object CPU
```

**Memory usage:**
```powershell
Get-Process rph | Select-Object WS
```

**Disk space:**
```powershell
Get-PSDrive C | Select-Object Free, Used
```

---

## Getting More Help

If your issue isn't covered here:

1. **Check logs:**
   ```powershell
   type .metadata\processing.log
   ```

2. **Run diagnostics:**
   ```powershell
   .\rph.exe check
   ```

3. **Create GitHub issue with:**
   - Windows version: `winver` or `systeminfo`
   - Go version: `go version`
   - LaTeX version: `pdflatex --version`
   - Full error message
   - Log file contents
   - Steps to reproduce

4. **Common info to include:**
   ```powershell
   # System info
   systeminfo | findstr /B /C:"OS Name" /C:"OS Version" /C:"System Type"

   # Go environment
   go env

   # Installed packages
   where pdflatex
   where latexmk
   ```

---

## Known Limitations on Windows

1. **Path length limit:** Windows has 260 character path limit (MAX_PATH)
   - Use shorter filenames
   - Place project closer to root (e.g., `C:\archivist`)

2. **Case insensitivity:** Windows filesystem is case-insensitive
   - `Paper.pdf` and `paper.pdf` are the same file

3. **File locking:** Windows locks open files
   - Close PDF viewers before processing
   - Don't open files while processing

4. **Terminal colors:** Command Prompt has limited color support
   - Use Windows Terminal for better experience
   - Or PowerShell 7+

5. **Line endings:** Windows uses CRLF, Linux uses LF
   - Git should handle this automatically
   - Configure Git: `git config --global core.autocrlf true`

---

## Performance Benchmarks (Windows)

Typical performance on Windows 10/11:

| Hardware | Single Paper | 10 Papers (4 workers) |
|----------|--------------|----------------------|
| i3, 4GB RAM | 3-6 min | 30-40 min |
| i5, 8GB RAM | 2-4 min | 15-25 min |
| i7, 16GB RAM | 1-3 min | 8-15 min |

**Fast mode** is ~40% faster than **Quality mode**.

---

## Quick Reference

**Check everything is working:**
```powershell
.\rph.exe check
go version
pdflatex --version
type .env
```

**Reset to clean state:**
```powershell
.\rph.exe clean
del /F /Q .metadata\*.db
del /F /Q tex_files\*
del /F /Q reports\*
```

**Rebuild from scratch:**
```powershell
go clean -cache
go mod download
go mod tidy
.\windows\build.bat
```

**Process single paper with verbose output:**
```powershell
.\rph.exe process lib\paper.pdf --parallel 1
```
