@echo off
REM Batch processing script for Windows
REM Processes all PDFs in the lib directory with progress tracking

setlocal enabledelayedexpansion

echo ╔══════════════════════════════════════════════════════════════╗
echo ║           Archivist - Batch Processing                       ║
echo ╚══════════════════════════════════════════════════════════════╝
echo.

REM Navigate to project root
cd /d "%~dp0\.."

REM Check if executable exists
if not exist rph.exe (
    echo [ERROR] rph.exe not found
    echo Please run build.bat first
    pause
    exit /b 1
)

REM Check dependencies
echo [INFO] Checking dependencies...
rph.exe check
if %ERRORLEVEL% NEQ 0 (
    echo.
    echo [ERROR] Dependency check failed
    echo Please fix the above issues before proceeding
    pause
    exit /b 1
)
echo.

REM Count PDF files
echo [INFO] Scanning for PDF files in lib\...
set PDF_COUNT=0
for %%f in (lib\*.pdf) do (
    set /a PDF_COUNT+=1
)

if %PDF_COUNT%==0 (
    echo [WARNING] No PDF files found in lib\ directory
    echo.
    echo Please add PDF papers to the lib\ directory
    pause
    exit /b 0
)

echo [INFO] Found %PDF_COUNT% PDF file(s)
echo.

REM Prompt for processing mode
echo Select processing mode:
echo   1. Fast Mode (Faster, good quality)
echo   2. Quality Mode (Slower, best quality)
echo.
set /p MODE_CHOICE="Enter your choice (1/2) [default: 1]: "
if "%MODE_CHOICE%"=="" set MODE_CHOICE=1

if "%MODE_CHOICE%"=="1" (
    set MODE=fast
    echo [INFO] Using Fast Mode
) else if "%MODE_CHOICE%"=="2" (
    set MODE=quality
    echo [INFO] Using Quality Mode
) else (
    echo [WARNING] Invalid choice, using Fast Mode
    set MODE=fast
)
echo.

REM Prompt for parallel workers
echo Enter number of parallel workers (1-8):
set /p WORKERS="[default: 4]: "
if "%WORKERS%"=="" set WORKERS=4
echo [INFO] Using %WORKERS% parallel workers
echo.

REM Confirm processing
echo ═══════════════════════════════════════════════════════════════
echo Ready to process %PDF_COUNT% papers
echo Mode: %MODE%
echo Workers: %WORKERS%
echo ═══════════════════════════════════════════════════════════════
echo.
set /p CONFIRM="Continue? (Y/n): "
if /i "%CONFIRM%"=="n" (
    echo Processing cancelled
    pause
    exit /b 0
)

REM Start processing
echo.
echo [INFO] Starting batch processing...
echo [INFO] Start time: %TIME%
echo.

REM Run processing
rph.exe process lib\ --mode %MODE% --parallel %WORKERS% --interactive=false

REM Check result
if %ERRORLEVEL% NEQ 0 (
    echo.
    echo [ERROR] Processing failed
    echo Check .metadata\processing.log for details
    pause
    exit /b 1
)

echo.
echo [INFO] End time: %TIME%
echo.

REM Show results
echo ╔══════════════════════════════════════════════════════════════╗
echo ║                Processing Complete!                          ║
echo ╚══════════════════════════════════════════════════════════════╝
echo.
echo Summary:
rph.exe list
echo.
echo Reports are available in: reports\
echo LaTeX source files in: tex_files\
echo.
echo Press any key to exit...
pause >nul
