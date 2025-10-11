@echo off
REM Automated installation script for Archivist on Windows
REM This script guides users through the complete setup process

echo ╔══════════════════════════════════════════════════════════════╗
echo ║           Archivist - Automated Installation                 ║
echo ║         Research Paper Helper for Windows                    ║
echo ╚══════════════════════════════════════════════════════════════╝
echo.

REM Check Go installation
echo [STEP 1/5] Checking Go installation...
where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Go is not installed
    echo.
    echo Please install Go 1.20+ from: https://go.dev/dl/
    echo After installation, restart this script.
    echo.
    pause
    exit /b 1
)
echo [SUCCESS] Go is installed
go version
echo.

REM Check LaTeX installation
echo [STEP 2/5] Checking LaTeX installation...
where pdflatex >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo [WARNING] pdflatex not found in PATH
    echo.
    echo Please install a LaTeX distribution:
    echo   - MiKTeX: https://miktex.org/download
    echo   - TeX Live: https://www.tug.org/texlive/
    echo.
    echo After installation, add LaTeX to PATH and restart this script.
    echo.
    set /p CONTINUE="Continue anyway? (y/N): "
    if /i not "%CONTINUE%"=="y" (
        exit /b 1
    )
) else (
    echo [SUCCESS] LaTeX is installed
    pdflatex --version | findstr pdflatex
)
echo.

REM Navigate to project root
cd /d "%~dp0\.."

REM Install dependencies
echo [STEP 3/5] Installing Go dependencies...
go mod download
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Failed to download dependencies
    pause
    exit /b 1
)
go mod tidy
echo [SUCCESS] Dependencies installed
echo.

REM Build executable
echo [STEP 4/5] Building executable...
call "%~dp0build.bat"
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Build failed
    pause
    exit /b 1
)
echo.

REM Setup configuration
echo [STEP 5/5] Setting up configuration...
echo.

REM Check for .env file
if not exist .env (
    echo [INFO] Creating .env file...
    echo.
    set /p API_KEY="Enter your Gemini API key (or press Enter to skip): "
    if not "!API_KEY!"=="" (
        echo GEMINI_API_KEY=!API_KEY!> .env
        echo [SUCCESS] .env file created
    ) else (
        echo GEMINI_API_KEY=your_api_key_here> .env
        echo [WARNING] .env created with placeholder
        echo Please edit .env and add your actual API key
    )
) else (
    echo [INFO] .env file already exists
)
echo.

REM Create directories
echo [INFO] Creating directories...
if not exist lib mkdir lib
if not exist tex_files mkdir tex_files
if not exist reports mkdir reports
if not exist .metadata mkdir .metadata
echo [SUCCESS] Directories created
echo.

REM Verify installation
echo.
echo ╔══════════════════════════════════════════════════════════════╗
echo ║               Installation Complete!                         ║
echo ╚══════════════════════════════════════════════════════════════╝
echo.
echo Verifying installation...
echo.
rph.exe check
echo.

echo ═══════════════════════════════════════════════════════════════
echo                     Quick Start Guide
echo ═══════════════════════════════════════════════════════════════
echo.
echo 1. Edit .env file and add your Gemini API key (if not done)
echo 2. Place your PDF papers in the lib\ directory
echo 3. Run: rph.exe run (for interactive TUI)
echo    OR
echo    Run: rph.exe process lib\ (for command-line processing)
echo.
echo For detailed documentation, see: windows\README_WINDOWS.md
echo.
echo Press any key to exit...
pause >nul
