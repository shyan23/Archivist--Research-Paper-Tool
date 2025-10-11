@echo off
REM Build script for Archivist on Windows
REM This script builds the rph executable for Windows

echo ╔══════════════════════════════════════════════════════════════╗
echo ║                Archivist Build Script                        ║
echo ║              Building for Windows (AMD64)                    ║
echo ╚══════════════════════════════════════════════════════════════╝
echo.

REM Check if Go is installed
where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Go is not installed or not in PATH
    echo Please install Go from: https://go.dev/dl/
    pause
    exit /b 1
)

echo [INFO] Go version:
go version
echo.

REM Navigate to project root (parent of windows directory)
cd /d "%~dp0\.."

echo [INFO] Current directory: %CD%
echo.

echo [STEP 1/3] Tidying Go modules...
go mod tidy
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Failed to tidy Go modules
    pause
    exit /b 1
)
echo [SUCCESS] Modules tidied
echo.

echo [STEP 2/3] Building executable...
REM Build for Windows AMD64
set GOOS=windows
set GOARCH=amd64
go build -o rph.exe ./cmd/main
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Build failed
    pause
    exit /b 1
)
echo [SUCCESS] Build complete
echo.

echo [STEP 3/3] Verifying executable...
if exist rph.exe (
    echo [SUCCESS] Executable created: rph.exe
    echo.
    dir rph.exe | findstr /C:"rph.exe"
) else (
    echo [ERROR] Executable not found
    pause
    exit /b 1
)

echo.
echo ╔══════════════════════════════════════════════════════════════╗
echo ║                    Build Complete!                           ║
echo ╚══════════════════════════════════════════════════════════════╝
echo.
echo Next steps:
echo   1. Create .env file with your GEMINI_API_KEY
echo   2. Run: rph.exe check
echo   3. Place PDFs in lib\ directory
echo   4. Run: rph.exe run
echo.
echo For detailed instructions, see: windows\README_WINDOWS.md
echo.
pause
