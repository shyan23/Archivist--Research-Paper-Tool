#!/bin/bash
# Run the simple FastAPI server with proper Python path

cd "$(dirname "$0")"
export PYTHONPATH="${PYTHONPATH}:$(pwd)"

echo "======================================================================="
echo "🚀 Starting Archivist RAG Chatbot API Server"
echo "======================================================================="
echo ""
echo "📁 Working directory: $(pwd)"
echo "🐍 Python path: $PYTHONPATH"
echo ""

# Check for API key
if [ -z "$GEMINI_API_KEY" ]; then
    echo "⚠️  WARNING: GEMINI_API_KEY not set!"
    echo "   Set it with: export GEMINI_API_KEY=your_key"
    echo ""
fi

# Run the server
python simple_server.py
