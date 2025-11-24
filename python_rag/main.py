 #!/usr/bin/env python3
"""
Simple standalone FastAPI server for RAG chatbot
Easy to test with FastAPI docs and ready for Go integration
"""
import os
import logging
from pathlib import Path
from typing import List, Optional
from dotenv import load_dotenv

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

from config import RAGConfig
from embeddings import create_embedding_provider
from vector_store import create_vector_store
from chunker import TextChunker
from indexer import DocumentIndexer
from retriever import Retriever
from chat_engine import ChatEngine

# Load environment variables
load_dotenv()

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Initialize FastAPI
app = FastAPI(
    title="Archivist RAG Chatbot API",
    description="Simple RAG chatbot for research papers (PDF support)",
    version="1.0.0",
    docs_url="/docs",  # Swagger UI
    redoc_url="/redoc"  # ReDoc
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Global instances (initialized on startup)
config = None
indexer = None
retriever = None
chat_engine = None

# Default lib path
LIB_PATH = Path("/home/shyan/Desktop/Code/Archivist/lib")


# ============================================================================
# REQUEST/RESPONSE MODELS
# ============================================================================

class IndexPDFRequest(BaseModel):
    """Request to index a PDF file"""
    pdf_path: str = Field(..., description="Full path to PDF file", example=str(LIB_PATH / "NIPS-2017-attention-is-all-you-need-Paper.pdf"))
    paper_title: Optional[str] = Field(None, description="Optional custom title for the paper")
    force_reindex: bool = Field(False, description="Force reindexing if already indexed")


class IndexResponse(BaseModel):
    """Response after indexing"""
    success: bool
    paper_title: str
    num_chunks: int
    message: str


class ChatSessionRequest(BaseModel):
    """Request to create a chat session"""
    paper_titles: List[str] = Field(default_factory=list, description="Papers to chat about (empty = all papers)")


class ChatSessionResponse(BaseModel):
    """Response with session info"""
    session_id: str
    paper_titles: List[str]
    message: str


class ChatMessageRequest(BaseModel):
    """Request to send a chat message"""
    session_id: str = Field(..., description="Session ID from create_session")
    message: str = Field(..., description="Your question or message", example="What is the attention mechanism?")


class ChatMessageResponse(BaseModel):
    """Response to a chat message"""
    role: str
    content: str
    citations: List[str]
    timestamp: float


# ============================================================================
# STARTUP
# ============================================================================

@app.on_event("startup")
async def startup_event():
    """Initialize RAG system on startup"""
    global config, indexer, retriever, chat_engine

    logger.info("🚀 Starting Archivist RAG API...")

    # Check API key
    if not os.getenv("GEMINI_API_KEY"):
        logger.warning("⚠️  GEMINI_API_KEY not set! Chat functionality will not work.")
        logger.warning("   Set it with: export GEMINI_API_KEY=your_key")

    # Load config
    config = RAGConfig.from_env()

    # Initialize embedding provider
    logger.info(f"📦 Loading embedding model: {config.embedding.model_name}")
    embedding_provider = create_embedding_provider(
        provider=config.embedding.provider,
        model_name=config.embedding.model_name,
        api_key=config.embedding.gemini_api_key
    )

    # Initialize vector store
    logger.info(f"💾 Loading vector store: {config.vector_store.provider}")
    vector_store = create_vector_store(
        provider=config.vector_store.provider,
        persist_directory=config.vector_store.persist_directory,
        collection_name=config.vector_store.collection_name,
        dimension=embedding_provider.dimension
    )

    # Initialize chunker
    chunker = TextChunker(
        chunk_size=config.chunking.chunk_size,
        chunk_overlap=config.chunking.chunk_overlap
    )

    # Initialize indexer
    indexer = DocumentIndexer(chunker, embedding_provider, vector_store)

    # Initialize retriever
    retriever = Retriever(
        vector_store,
        embedding_provider,
        top_k=config.retriever.top_k,
        score_threshold=config.retriever.score_threshold
    )

    # Initialize chat engine
    chat_engine = ChatEngine(
        retriever=retriever,
        llm_provider=config.chat.provider,
        gemini_api_key=config.chat.gemini_api_key,
        model_name=config.chat.gemini_model,
        temperature=config.chat.temperature
    )

    logger.info("✅ RAG system ready!")
    logger.info(f"📚 Currently indexed papers: {len(indexer.get_indexed_papers())}")


# ============================================================================
# HEALTH & INFO ENDPOINTS
# ============================================================================

@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "service": "Archivist RAG Chatbot",
        "status": "running",
        "docs": "/docs",
        "health": "/health"
    }


@app.get("/health")
async def health():
    """Health check"""
    return {"status": "healthy", "indexed_papers": len(indexer.get_indexed_papers())}


@app.get("/info")
async def get_info():
    """Get system information"""
    return {
        "total_papers": len(indexer.get_indexed_papers()),
        "indexed_papers": indexer.get_indexed_papers(),
        "embedding_model": f"{config.embedding.provider}:{config.embedding.model_name}",
        "vector_store": config.vector_store.provider,
        "lib_path": str(LIB_PATH),
        "available_pdfs": [f.name for f in LIB_PATH.glob("*.pdf")] if LIB_PATH.exists() else []
    }


# ============================================================================
# INDEXING ENDPOINTS
# ============================================================================

@app.post("/index/pdf", response_model=IndexResponse)
async def index_pdf(request: IndexPDFRequest):
    """
    Index a PDF file from the server

    Example:
    ```json
    {
      "pdf_path": "/home/shyan/Desktop/Code/Archivist/lib/NIPS-2017-attention-is-all-you-need-Paper.pdf"
    }
    ```
    """
    try:
        pdf_path = Path(request.pdf_path)

        if not pdf_path.exists():
            raise HTTPException(status_code=404, detail=f"PDF not found: {request.pdf_path}")

        # Index the PDF
        num_chunks = indexer.index_paper_from_pdf(
            pdf_path=str(pdf_path),
            paper_title=request.paper_title,
            force_reindex=request.force_reindex
        )

        # Get actual title used
        from pdf_utils import extract_title_from_pdf
        actual_title = request.paper_title or extract_title_from_pdf(str(pdf_path)) or pdf_path.stem

        return IndexResponse(
            success=True,
            paper_title=actual_title,
            num_chunks=num_chunks,
            message=f"Successfully indexed {num_chunks} chunks from PDF"
        )

    except Exception as e:
        logger.error(f"Failed to index PDF: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/index/papers")
async def list_indexed_papers():
    """List all indexed papers"""
    papers = indexer.get_indexed_papers()
    papers_info = []

    for paper in papers:
        info = indexer.get_index_info(paper)
        papers_info.append({
            "title": paper,
            "num_chunks": info.get("num_chunks", 0),
            "indexed_at": info.get("indexed_at", "")
        })

    return {
        "total": len(papers),
        "papers": papers_info
    }


@app.delete("/index/paper/{paper_title}")
async def delete_paper(paper_title: str):
    """Delete an indexed paper"""
    deleted = indexer.delete_paper_index(paper_title)

    if deleted == 0:
        raise HTTPException(status_code=404, detail="Paper not found")

    return {
        "success": True,
        "paper_title": paper_title,
        "chunks_deleted": deleted
    }


# ============================================================================
# CHAT ENDPOINTS
# ============================================================================

@app.post("/chat/session", response_model=ChatSessionResponse)
async def create_session(request: ChatSessionRequest):
    """
    Create a new chat session

    Example (chat with all papers):
    ```json
    {
      "paper_titles": []
    }
    ```

    Example (chat with specific papers):
    ```json
    {
      "paper_titles": ["NIPS-2017-attention-is-all-you-need-Paper", "Focus"]
    }
    ```
    """
    try:
        session = chat_engine.create_session(request.paper_titles)

        papers_msg = ", ".join(session.paper_titles) if session.paper_titles else "all indexed papers"

        return ChatSessionResponse(
            session_id=session.session_id,
            paper_titles=session.paper_titles,
            message=f"Session created. Chatting about: {papers_msg}"
        )

    except Exception as e:
        logger.error(f"Failed to create session: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/chat/message", response_model=ChatMessageResponse)
async def send_message(request: ChatMessageRequest):
    """
    Send a message in a chat session

    Example:
    ```json
    {
      "session_id": "your_session_id_here",
      "message": "What is the attention mechanism?"
    }
    ```
    """
    try:
        response = chat_engine.chat(
            session_id=request.session_id,
            user_message=request.message
        )

        return ChatMessageResponse(
            role=response.role,
            content=response.content,
            citations=response.citations,
            timestamp=response.timestamp
        )

    except ValueError as e:
        raise HTTPException(status_code=404, detail=str(e))
    except Exception as e:
        logger.error(f"Chat error: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/chat/session/{session_id}")
async def get_session(session_id: str):
    """Get chat session history"""
    session = chat_engine.get_session(session_id)

    if not session:
        raise HTTPException(status_code=404, detail="Session not found")

    return {
        "session_id": session.session_id,
        "paper_titles": session.paper_titles,
        "messages": [
            {
                "role": msg.role,
                "content": msg.content,
                "citations": msg.citations,
                "timestamp": msg.timestamp
            }
            for msg in session.messages
        ]
    }


@app.delete("/chat/session/{session_id}")
async def delete_session(session_id: str):
    """Delete a chat session"""
    deleted = chat_engine.delete_session(session_id)

    if not deleted:
        raise HTTPException(status_code=404, detail="Session not found")

    return {"success": True, "session_id": session_id}


# ============================================================================
# MAIN
# ============================================================================

if __name__ == "__main__":
    import uvicorn

    port = int(os.getenv("PORT", 8000))

    print("\n" + "=" * 70)
    print("🚀 ARCHIVIST RAG CHATBOT API")
    print("=" * 70)
    print(f"\n📡 Server starting on: http://localhost:{port}")
    print(f"📚 API Documentation: http://localhost:{port}/docs")
    print(f"📖 ReDoc: http://localhost:{port}/redoc")
    print("\n" + "=" * 70 + "\n")

    uvicorn.run(
        "simple_server:app",
        host="0.0.0.0",
        port=port,
        reload=False,
        log_level="info"
    )
