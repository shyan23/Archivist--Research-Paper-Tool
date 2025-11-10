"""
FastAPI server for Python RAG system
Provides HTTP API for Go integration
"""
import logging
import os
from typing import List, Optional
from pathlib import Path

from fastapi import FastAPI, HTTPException, BackgroundTasks
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

from .config import RAGConfig
from .embeddings import create_embedding_provider
from .vector_store import create_vector_store
from .chunker import TextChunker
from .indexer import DocumentIndexer
from .retriever import Retriever
from .chat_engine import ChatEngine, Message, ChatSession

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Initialize FastAPI app
app = FastAPI(
    title="Archivist RAG API",
    description="Python RAG system for research paper chat",
    version="1.0.0"
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Global instances
config: Optional[RAGConfig] = None
embedding_provider = None
vector_store = None
indexer: Optional[DocumentIndexer] = None
retriever: Optional[Retriever] = None
chat_engine: Optional[ChatEngine] = None


# Request/Response models
class IndexPaperRequest(BaseModel):
    paper_title: str
    latex_content: str
    pdf_path: Optional[str] = None
    force_reindex: bool = False


class IndexPaperResponse(BaseModel):
    success: bool
    paper_title: str
    num_chunks: int
    message: str


class CreateSessionRequest(BaseModel):
    paper_titles: List[str] = Field(default_factory=list)


class CreateSessionResponse(BaseModel):
    session_id: str
    paper_titles: List[str]
    created_at: float


class ChatRequest(BaseModel):
    session_id: str
    message: str


class ChatResponse(BaseModel):
    role: str
    content: str
    timestamp: float
    citations: List[str]


class RetrieveRequest(BaseModel):
    query: str
    paper_titles: Optional[List[str]] = None
    top_k: int = 5


class RetrieveResponse(BaseModel):
    query: str
    context_text: str
    sources: List[str]
    sections: List[str]
    total_chunks: int


class SystemInfoResponse(BaseModel):
    status: str
    total_documents: int
    indexed_papers: List[str]
    embedding_model: str
    embedding_dimension: int
    vector_store: str


@app.on_event("startup")
async def startup_event():
    """Initialize RAG system on startup"""
    global config, embedding_provider, vector_store, indexer, retriever, chat_engine

    logger.info("🚀 Starting Archivist RAG API server...")

    # Load configuration
    config = RAGConfig.from_env()

    # Create embedding provider
    logger.info("Initializing embedding provider...")
    embedding_provider = create_embedding_provider(
        provider=config.embedding.provider,
        model_name=config.embedding.model_name,
        api_key=config.embedding.gemini_api_key or config.embedding.openai_api_key
    )

    # Create vector store
    logger.info("Initializing vector store...")
    vector_store = create_vector_store(
        provider=config.vector_store.provider,
        persist_directory=config.vector_store.persist_directory,
        collection_name=config.vector_store.collection_name,
        dimension=embedding_provider.dimension
    )

    # Create chunker
    chunker = TextChunker(
        chunk_size=config.chunking.chunk_size,
        chunk_overlap=config.chunking.chunk_overlap,
        respect_sections=config.chunking.respect_sections
    )

    # Create indexer
    indexer = DocumentIndexer(
        chunker=chunker,
        embedding_provider=embedding_provider,
        vector_store=vector_store
    )

    # Create retriever
    retriever = Retriever(
        vector_store=vector_store,
        embedding_provider=embedding_provider,
        top_k=config.retriever.top_k,
        score_threshold=config.retriever.score_threshold,
        max_context_length=config.retriever.max_context_length
    )

    # Create chat engine
    chat_engine = ChatEngine(
        retriever=retriever,
        llm_provider=config.chat.provider,
        gemini_api_key=config.chat.gemini_api_key,
        model_name=config.chat.gemini_model,
        temperature=config.chat.temperature,
        max_tokens=config.chat.max_tokens
    )

    logger.info("✅ RAG system initialized successfully!")


@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "service": "Archivist RAG API",
        "version": "1.0.0",
        "status": "running"
    }


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy"}


@app.get("/system/info", response_model=SystemInfoResponse)
async def get_system_info():
    """Get system information"""
    return SystemInfoResponse(
        status="ready",
        total_documents=vector_store.count(),
        indexed_papers=indexer.get_indexed_papers(),
        embedding_model=f"{config.embedding.provider}:{config.embedding.model_name}",
        embedding_dimension=embedding_provider.dimension,
        vector_store=config.vector_store.provider
    )


@app.post("/index/paper", response_model=IndexPaperResponse)
async def index_paper(request: IndexPaperRequest, background_tasks: BackgroundTasks):
    """Index a research paper"""
    try:
        num_chunks = indexer.index_paper(
            paper_title=request.paper_title,
            latex_content=request.latex_content,
            pdf_path=request.pdf_path,
            force_reindex=request.force_reindex
        )

        return IndexPaperResponse(
            success=True,
            paper_title=request.paper_title,
            num_chunks=num_chunks,
            message=f"Successfully indexed {num_chunks} chunks"
        )

    except Exception as e:
        logger.error(f"Failed to index paper: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/index/papers")
async def get_indexed_papers():
    """Get list of indexed papers"""
    return {
        "papers": indexer.get_indexed_papers(),
        "total": len(indexer.get_indexed_papers())
    }


@app.get("/index/paper/{paper_title}")
async def get_paper_info(paper_title: str):
    """Get information about an indexed paper"""
    info = indexer.get_index_info(paper_title)

    if not info:
        raise HTTPException(status_code=404, detail="Paper not found")

    return info


@app.delete("/index/paper/{paper_title}")
async def delete_paper_index(paper_title: str):
    """Delete index for a paper"""
    deleted = indexer.delete_paper_index(paper_title)

    if deleted == 0:
        raise HTTPException(status_code=404, detail="Paper not found")

    return {
        "success": True,
        "paper_title": paper_title,
        "chunks_deleted": deleted
    }


@app.post("/retrieve", response_model=RetrieveResponse)
async def retrieve_context(request: RetrieveRequest):
    """Retrieve relevant context for a query"""
    try:
        if request.paper_titles:
            if len(request.paper_titles) == 1:
                context = retriever.retrieve_from_paper(
                    request.query,
                    request.paper_titles[0],
                    top_k=request.top_k
                )
            else:
                context = retriever.retrieve_multi_paper(
                    request.query,
                    request.paper_titles,
                    top_k=request.top_k
                )
        else:
            context = retriever.retrieve(
                request.query,
                top_k=request.top_k
            )

        return RetrieveResponse(
            query=request.query,
            context_text=context.context_text,
            sources=context.sources,
            sections=context.sections,
            total_chunks=context.total_chunks
        )

    except Exception as e:
        logger.error(f"Failed to retrieve context: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/chat/session", response_model=CreateSessionResponse)
async def create_chat_session(request: CreateSessionRequest):
    """Create a new chat session"""
    try:
        session = chat_engine.create_session(request.paper_titles)

        return CreateSessionResponse(
            session_id=session.session_id,
            paper_titles=session.paper_titles,
            created_at=session.created_at
        )

    except Exception as e:
        logger.error(f"Failed to create session: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/chat/message", response_model=ChatResponse)
async def send_chat_message(request: ChatRequest):
    """Send a chat message"""
    try:
        message = chat_engine.chat(
            session_id=request.session_id,
            user_message=request.message
        )

        return ChatResponse(
            role=message.role,
            content=message.content,
            timestamp=message.timestamp,
            citations=message.citations
        )

    except ValueError as e:
        raise HTTPException(status_code=404, detail=str(e))
    except Exception as e:
        logger.error(f"Failed to process chat message: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/chat/session/{session_id}")
async def get_chat_session(session_id: str):
    """Get chat session details"""
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
                "timestamp": msg.timestamp,
                "citations": msg.citations
            }
            for msg in session.messages
        ],
        "created_at": session.created_at,
        "last_updated": session.last_updated
    }


@app.delete("/chat/session/{session_id}")
async def delete_chat_session(session_id: str):
    """Delete a chat session"""
    deleted = chat_engine.delete_session(session_id)

    if not deleted:
        raise HTTPException(status_code=404, detail="Session not found")

    return {"success": True, "session_id": session_id}


@app.get("/chat/sessions")
async def list_chat_sessions():
    """List all active chat sessions"""
    sessions = chat_engine.get_all_sessions()

    return {
        "sessions": [
            {
                "session_id": s.session_id,
                "paper_titles": s.paper_titles,
                "message_count": len(s.messages),
                "created_at": s.created_at,
                "last_updated": s.last_updated
            }
            for s in sessions
        ],
        "total": len(sessions)
    }


if __name__ == "__main__":
    import uvicorn

    # Get port from environment or use default
    port = int(os.getenv("PORT", 8000))

    uvicorn.run(
        "api_server:app",
        host="0.0.0.0",
        port=port,
        reload=False,
        log_level="info"
    )
