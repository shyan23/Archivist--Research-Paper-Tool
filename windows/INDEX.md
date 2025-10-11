# Windows Files Index

This directory contains all Windows-specific files for Archivist.

## 📁 Files Overview

### Documentation

| File | Purpose | When to Use |
|------|---------|-------------|
| **[QUICKSTART.md](QUICKSTART.md)** | 5-minute setup guide | **Start here!** First-time installation |
| **[README_WINDOWS.md](README_WINDOWS.md)** | Complete Windows guide | Comprehensive reference, detailed instructions |
| **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)** | Problem-solving guide | When things go wrong |
| **INDEX.md** (this file) | Directory overview | Navigate Windows files |

### Scripts

| File | Purpose | Usage |
|------|---------|-------|
| **[build.bat](build.bat)** | Build executable | `.\windows\build.bat` |
| **[install.bat](install.bat)** | Automated installer | `.\windows\install.bat` |
| **[process-all.bat](process-all.bat)** | Batch processing | `.\windows\process-all.bat` |

### Build Tools

| File | Purpose | Usage |
|------|---------|-------|
| **[Makefile.windows](Makefile.windows)** | Windows Makefile | `make -f windows\Makefile.windows build` |

## 🚀 Quick Start Path

Follow this path for the smoothest experience:

```
1. QUICKSTART.md         → Get running in 5 minutes
   ↓
2. install.bat           → Run automated installer
   ↓
3. README_WINDOWS.md     → Learn all features
   ↓
4. TROUBLESHOOTING.md    → (Only if you hit issues)
```

## 📖 Documentation Deep Dive

### QUICKSTART.md
**Best for:** First-time users, quick setup

**Contains:**
- ✅ Prerequisites checklist
- ✅ 5-minute installation
- ✅ First run guide
- ✅ Common commands
- ✅ Quick troubleshooting

**Read if:** You want to get started ASAP

---

### README_WINDOWS.md
**Best for:** Complete understanding, advanced usage

**Contains:**
- 📦 Detailed installation steps
- 🎯 All usage commands
- ⚙️ Configuration guide
- 🔧 Advanced features
- 📊 Performance tips
- ❓ Comprehensive FAQ

**Read if:** You want to master Archivist on Windows

---

### TROUBLESHOOTING.md
**Best for:** Solving problems, debugging issues

**Contains:**
- 🐛 Installation issues
- 📄 LaTeX problems
- 🌐 API/network issues
- ⚡ Performance optimization
- 🛠️ Advanced diagnostics
- 📋 Known limitations

**Read if:** Something's not working

---

## 🔧 Scripts Guide

### build.bat
**Purpose:** Build Windows executable (`rph.exe`)

**When to use:**
- First-time setup (after cloning)
- After updating code
- After `go.mod` changes

**How to use:**
```powershell
.\windows\build.bat
```

**What it does:**
1. Checks Go installation
2. Runs `go mod tidy`
3. Builds `rph.exe` for Windows AMD64
4. Verifies executable

**Output:** Creates `rph.exe` in project root

---

### install.bat
**Purpose:** Complete automated installation

**When to use:**
- First-time setup (recommended)
- Setting up on a new Windows machine
- After major updates

**How to use:**
```powershell
.\windows\install.bat
```

**What it does:**
1. ✅ Checks Go installation
2. ✅ Checks LaTeX installation
3. ✅ Downloads dependencies
4. ✅ Builds executable
5. ✅ Creates `.env` file (prompts for API key)
6. ✅ Creates required directories
7. ✅ Verifies installation

**Interactive:** Prompts for API key during setup

---

### process-all.bat
**Purpose:** Batch process all papers with user-friendly interface

**When to use:**
- Processing multiple papers at once
- Overnight batch jobs
- When you want guided processing

**How to use:**
```powershell
.\windows\process-all.bat
```

**What it does:**
1. Checks dependencies
2. Counts PDF files in `lib\`
3. Prompts for processing mode (Fast/Quality)
4. Prompts for worker count
5. Confirms before processing
6. Processes all papers
7. Shows summary

**Interactive:** Guides you through options

---

### Makefile.windows
**Purpose:** Windows-compatible Makefile

**When to use:**
- If you prefer `make` commands
- For CI/CD on Windows
- Advanced users familiar with Make

**Requirements:** GNU Make for Windows

**How to use:**
```powershell
make -f windows\Makefile.windows build
make -f windows\Makefile.windows test
make -f windows\Makefile.windows clean
```

**Available targets:**
- `build` - Build executable
- `test` - Run tests
- `clean` - Clean artifacts
- `run` - Run TUI
- `process` - Process papers

**Note:** Most users should use `.bat` scripts instead

---

## 🎯 Common Tasks

### Task: First Time Installation

**Files to use:**
1. `QUICKSTART.md` (read)
2. `install.bat` (run)

**Commands:**
```powershell
# Read the guide first
notepad windows\QUICKSTART.md

# Run installer
.\windows\install.bat
```

---

### Task: Build After Code Changes

**Files to use:**
1. `build.bat`

**Commands:**
```powershell
.\windows\build.bat
```

---

### Task: Process Multiple Papers

**Files to use:**
1. `process-all.bat`

**Commands:**
```powershell
# Place PDFs in lib\
copy C:\Downloads\*.pdf lib\

# Run batch processor
.\windows\process-all.bat
```

---

### Task: Troubleshoot Installation Issues

**Files to use:**
1. `TROUBLESHOOTING.md` (read)
2. Check logs

**Commands:**
```powershell
# Read troubleshooting guide
notepad windows\TROUBLESHOOTING.md

# Check system status
.\rph.exe check

# View logs
type .metadata\processing.log
```

---

### Task: Learn All Features

**Files to use:**
1. `README_WINDOWS.md`

**Commands:**
```powershell
notepad windows\README_WINDOWS.md
```

---

## 💡 Tips for Windows Users

### Path Separators
Both work, but backslash is native:
```powershell
.\rph.exe process lib\paper.pdf  # ✅ Windows native
.\rph.exe process lib/paper.pdf  # ✅ Also works
```

### PowerShell vs Command Prompt
Both work! PowerShell has better colors:
- **Command Prompt**: Basic, works everywhere
- **PowerShell**: Better colors, more features
- **Windows Terminal**: Best experience (recommended)

### Execution Policy
If scripts won't run in PowerShell:
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### Running from Any Directory
Add to PATH to use `rph.exe` anywhere:
1. Right-click "This PC" → Properties
2. Advanced system settings → Environment Variables
3. Edit PATH, add: `C:\path\to\archivist`
4. Restart terminal

### Antivirus Exclusions
For better performance, exclude:
- `rph.exe`
- `tex_files\` folder
- `reports\` folder

---

## 🆘 Getting Help

### Problem Solving Order

1. **Check QUICKSTART.md** - Quick fixes
2. **Check TROUBLESHOOTING.md** - Common issues
3. **Run diagnostics:**
   ```powershell
   .\rph.exe check
   ```
4. **Check logs:**
   ```powershell
   type .metadata\processing.log
   ```
5. **Read README_WINDOWS.md** - Detailed info
6. **Create GitHub issue** - Include logs and system info

### System Information to Include

When reporting issues:
```powershell
# OS version
systeminfo | findstr /B /C:"OS Name" /C:"OS Version"

# Go version
go version

# LaTeX version
pdflatex --version

# Check status
.\rph.exe check
```

---

## 📚 Additional Resources

### Main Documentation
- **[Main README](../README.md)** - Overview and Linux/macOS instructions
- **[Configuration Guide](../config/config.yaml)** - Configuration options
- **[TUI Guide](../TUI_GUIDE.md)** - Interactive UI documentation (if available)

### External Resources
- **Go Downloads**: https://go.dev/dl/
- **MiKTeX**: https://miktex.org/download
- **TeX Live**: https://www.tug.org/texlive/
- **Gemini API**: https://aistudio.google.com/app/apikey
- **GitHub Issues**: [Create an issue](https://github.com/yourusername/archivist/issues)

---

## 🎉 Success Stories

Once you're up and running:

1. ✅ **Built `rph.exe`** successfully
2. ✅ **Processed first paper** in Fast Mode
3. ✅ **Opened report PDF** - looks amazing!
4. ✅ **Batch processed** 10 papers overnight
5. ✅ **Customized config** for your needs

### Share Your Experience

If Archivist helped you, consider:
- ⭐ Starring the repository
- 📝 Writing about your experience
- 🐛 Reporting bugs you find
- 💡 Suggesting improvements

---

## 📝 Version Information

**Windows Compatibility:** Windows 10, Windows 11
**Architecture:** AMD64 (x86_64)
**Go Version Required:** 1.20+
**LaTeX Required:** MiKTeX or TeX Live

---

## 🔄 Keeping Updated

### Update Archivist

```powershell
# Pull latest changes
git pull origin main

# Rebuild
.\windows\build.bat
```

### Update Dependencies

```powershell
go mod tidy
go mod download
```

---

**Need help?** Start with [QUICKSTART.md](QUICKSTART.md) or [TROUBLESHOOTING.md](TROUBLESHOOTING.md)

**Ready to start?** Run `.\windows\install.bat`

**Questions?** Check [README_WINDOWS.md](README_WINDOWS.md)
