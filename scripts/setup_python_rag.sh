#!/bin/bash

# Setup script for Python RAG system
# This script installs dependencies and configures the system

set -e

echo "🚀 Setting up Archivist Python RAG System..."
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check Python version
echo "Checking Python installation..."
if command -v python3 &> /dev/null; then
    PYTHON_CMD=python3
    PYTHON_VERSION=$(python3 --version | cut -d' ' -f2)
    echo -e "${GREEN}✓${NC} Found Python $PYTHON_VERSION"
elif command -v python &> /dev/null; then
    PYTHON_CMD=python
    PYTHON_VERSION=$(python --version | cut -d' ' -f2)
    echo -e "${GREEN}✓${NC} Found Python $PYTHON_VERSION"
else
    echo -e "${RED}✗${NC} Python not found. Please install Python 3.8 or higher."
    exit 1
fi

# Check Python version >= 3.8
PYTHON_MAJOR=$($PYTHON_CMD -c 'import sys; print(sys.version_info.major)')
PYTHON_MINOR=$($PYTHON_CMD -c 'import sys; print(sys.version_info.minor)')

if [ "$PYTHON_MAJOR" -lt 3 ] || ([ "$PYTHON_MAJOR" -eq 3 ] && [ "$PYTHON_MINOR" -lt 8 ]); then
    echo -e "${RED}✗${NC} Python 3.8 or higher required. Found $PYTHON_VERSION"
    exit 1
fi

# Check pip
echo ""
echo "Checking pip installation..."
if ! $PYTHON_CMD -m pip --version &> /dev/null; then
    echo -e "${RED}✗${NC} pip not found. Installing..."
    $PYTHON_CMD -m ensurepip --default-pip
fi
echo -e "${GREEN}✓${NC} pip is installed"

# Create virtual environment (optional but recommended)
echo ""
echo "Do you want to create a virtual environment? (recommended) [y/N]"
read -r response
if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
    echo "Creating virtual environment..."
    $PYTHON_CMD -m venv venv
    source venv/bin/activate
    echo -e "${GREEN}✓${NC} Virtual environment created and activated"
fi

# Install requirements
echo ""
echo "Installing Python dependencies..."
cd python_rag
$PYTHON_CMD -m pip install --upgrade pip
$PYTHON_CMD -m pip install -r requirements.txt

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓${NC} Python dependencies installed successfully"
else
    echo -e "${RED}✗${NC} Failed to install dependencies"
    exit 1
fi

cd ..

# Check for API key
echo ""
echo "Checking for Gemini API key..."
if [ -z "$GEMINI_API_KEY" ]; then
    echo -e "${YELLOW}⚠${NC}  GEMINI_API_KEY environment variable not set"
    echo ""
    echo "To use the RAG system, you need a Gemini API key."
    echo "Get one at: https://aistudio.google.com/app/apikey"
    echo ""
    echo "Set it with:"
    echo "  export GEMINI_API_KEY='your_api_key_here'"
    echo ""
    echo "Or add it to your ~/.bashrc or ~/.zshrc:"
    echo "  echo 'export GEMINI_API_KEY=\"your_api_key_here\"' >> ~/.bashrc"
else
    echo -e "${GREEN}✓${NC} GEMINI_API_KEY is set"
fi

# Create metadata directories
echo ""
echo "Creating metadata directories..."
mkdir -p .metadata/chromadb
mkdir -p .metadata/faiss_index
echo -e "${GREEN}✓${NC} Directories created"

# Test installation
echo ""
echo "Testing installation..."
if $PYTHON_CMD -c "import chromadb; import sentence_transformers; import fastapi" &> /dev/null; then
    echo -e "${GREEN}✓${NC} All core dependencies are working"
else
    echo -e "${RED}✗${NC} Some dependencies failed to import"
    exit 1
fi

# Download embedding model (optional)
echo ""
echo "Do you want to pre-download the embedding model? (recommended) [y/N]"
read -r response
if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
    echo "Downloading Sentence Transformer model..."
    $PYTHON_CMD -c "from sentence_transformers import SentenceTransformer; SentenceTransformer('all-MiniLM-L6-v2')"
    echo -e "${GREEN}✓${NC} Model downloaded"
fi

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}✅ Setup Complete!${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Next steps:"
echo ""
echo "1. Set your Gemini API key (if not done already):"
echo "   export GEMINI_API_KEY='your_api_key_here'"
echo ""
echo "2. Index some papers:"
echo "   python -m python_rag.cli index tex_files/"
echo ""
echo "3. Start chatting:"
echo "   python -m python_rag.cli chat"
echo ""
echo "4. Or start the API server:"
echo "   python -m python_rag.cli server"
echo ""
echo "For more information, see python_rag/README.md"
echo ""
