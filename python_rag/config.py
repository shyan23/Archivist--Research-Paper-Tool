"""
Configuration for Python RAG system
"""
import os
from pathlib import Path
from typing import Optional
from pydantic import BaseModel, Field


class EmbeddingConfig(BaseModel):
    """Embedding model configuration"""
    # Options: "sentence-transformers", "openai", "gemini"
    provider: str = "sentence-transformers"

    # For sentence-transformers
    model_name: str = "all-MiniLM-L6-v2"  # Fast and good quality
    # Alternative: "all-mpnet-base-v2" (better quality, slower)
    # Alternative: "multi-qa-mpnet-base-dot-v1" (optimized for QA)

    # For OpenAI
    openai_api_key: Optional[str] = None
    openai_model: str = "text-embedding-3-large"

    # For Gemini
    gemini_api_key: Optional[str] = None
    gemini_model: str = "models/text-embedding-004"

    dimension: int = 384  # Depends on model


class VectorStoreConfig(BaseModel):
    """Vector store configuration"""
    # Options: "chromadb", "faiss", "redis"
    provider: str = "chromadb"

    # ChromaDB settings
    persist_directory: str = ".metadata/chromadb"
    collection_name: str = "research_papers"

    # FAISS settings
    faiss_index_path: str = ".metadata/faiss_index"

    # Redis settings
    redis_host: str = "localhost"
    redis_port: int = 6379
    redis_db: int = 0
    redis_password: Optional[str] = None


class ChunkingConfig(BaseModel):
    """Text chunking configuration"""
    chunk_size: int = 1000  # Characters
    chunk_overlap: int = 200  # Characters
    separator: str = "\n\n"

    # Section-aware chunking
    respect_sections: bool = True
    section_patterns: list[str] = Field(default_factory=lambda: [
        r"\\section\{([^}]+)\}",
        r"\\subsection\{([^}]+)\}",
        r"^#+\s+(.+)$",  # Markdown headers
    ])


class RetrieverConfig(BaseModel):
    """Retrieval configuration"""
    top_k: int = 5  # Number of chunks to retrieve
    score_threshold: float = 0.3  # Minimum similarity score
    max_context_length: int = 8000  # Maximum total context characters

    # Reranking
    enable_reranking: bool = False
    reranker_model: str = "cross-encoder/ms-marco-MiniLM-L-6-v2"


class ChatConfig(BaseModel):
    """Chat/LLM configuration"""
    # LLM provider
    provider: str = "gemini"  # Options: "gemini", "openai", "anthropic"

    # Gemini settings
    gemini_api_key: Optional[str] = None
    gemini_model: str = "models/gemini-2.0-flash-exp"
    temperature: float = 0.7
    max_tokens: int = 8000

    # System prompt
    system_prompt: str = """You are a helpful AI research assistant for CS students studying AI/ML, Computer Vision, and Networking papers.
Your role is to:
- Answer questions accurately using the provided context from research papers
- Explain complex concepts in a student-friendly manner
- Cite specific sections when referencing information
- Be clear, concise, and technically accurate
- Admit when you don't have enough information to answer

Always ground your responses in the provided context."""


class RAGConfig(BaseModel):
    """Main RAG system configuration"""
    embedding: EmbeddingConfig = Field(default_factory=EmbeddingConfig)
    vector_store: VectorStoreConfig = Field(default_factory=VectorStoreConfig)
    chunking: ChunkingConfig = Field(default_factory=ChunkingConfig)
    retriever: RetrieverConfig = Field(default_factory=RetrieverConfig)
    chat: ChatConfig = Field(default_factory=ChatConfig)

    # Paths
    data_dir: str = "./lib"
    reports_dir: str = "./reports"
    tex_files_dir: str = "./tex_files"

    # Logging
    log_level: str = "INFO"

    @classmethod
    def from_env(cls) -> "RAGConfig":
        """Load configuration from environment variables"""
        config = cls()

        # Load API keys from environment
        if api_key := os.getenv("GEMINI_API_KEY"):
            config.embedding.gemini_api_key = api_key
            config.chat.gemini_api_key = api_key

        if api_key := os.getenv("OPENAI_API_KEY"):
            config.embedding.openai_api_key = api_key

        # Load from config file if exists
        config_path = Path("config/python_rag.yaml")
        if config_path.exists():
            import yaml
            with open(config_path) as f:
                data = yaml.safe_load(f)
                # Update config with file data
                # (simplified - in production, use proper merging)

        return config


# Default configuration instance
default_config = RAGConfig()
