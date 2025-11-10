"""
Embedding generation using multiple providers
Supports: Sentence Transformers, OpenAI, Gemini
"""
import logging
from abc import ABC, abstractmethod
from typing import List, Optional

import numpy as np

logger = logging.getLogger(__name__)


class EmbeddingProvider(ABC):
    """Abstract base class for embedding providers"""

    @abstractmethod
    def embed_text(self, text: str) -> np.ndarray:
        """Embed a single text"""
        pass

    @abstractmethod
    def embed_batch(self, texts: List[str]) -> List[np.ndarray]:
        """Embed multiple texts"""
        pass

    @property
    @abstractmethod
    def dimension(self) -> int:
        """Embedding dimension"""
        pass


class SentenceTransformerEmbedding(EmbeddingProvider):
    """Sentence Transformers embedding provider (best for local/offline)"""

    def __init__(self, model_name: str = "all-MiniLM-L6-v2"):
        """
        Initialize Sentence Transformers embedder

        Recommended models:
        - all-MiniLM-L6-v2: Fast, 384 dims, good quality
        - all-mpnet-base-v2: Better quality, 768 dims, slower
        - multi-qa-mpnet-base-dot-v1: Optimized for Q&A tasks
        """
        try:
            from sentence_transformers import SentenceTransformer
        except ImportError:
            raise ImportError("sentence-transformers not installed. Run: pip install sentence-transformers")

        logger.info(f"Loading Sentence Transformer model: {model_name}")
        self.model = SentenceTransformer(model_name)
        self._dimension = self.model.get_sentence_embedding_dimension()
        logger.info(f"✓ Model loaded (dimension: {self._dimension})")

    def embed_text(self, text: str) -> np.ndarray:
        """Embed a single text"""
        embedding = self.model.encode(text, convert_to_numpy=True)
        return embedding.astype(np.float32)

    def embed_batch(self, texts: List[str], batch_size: int = 32, show_progress: bool = False) -> List[np.ndarray]:
        """Embed multiple texts with batching"""
        embeddings = self.model.encode(
            texts,
            batch_size=batch_size,
            show_progress_bar=show_progress,
            convert_to_numpy=True
        )
        return [emb.astype(np.float32) for emb in embeddings]

    @property
    def dimension(self) -> int:
        return self._dimension


class GeminiEmbedding(EmbeddingProvider):
    """Google Gemini embedding provider"""

    def __init__(self, api_key: str, model: str = "models/text-embedding-004"):
        """Initialize Gemini embedder"""
        try:
            import google.generativeai as genai
        except ImportError:
            raise ImportError("google-generativeai not installed. Run: pip install google-generativeai")

        genai.configure(api_key=api_key)
        self.model_name = model
        self._dimension = 768  # text-embedding-004 dimension
        logger.info(f"✓ Gemini embedder initialized: {model}")

    def embed_text(self, text: str) -> np.ndarray:
        """Embed a single text"""
        import google.generativeai as genai

        result = genai.embed_content(
            model=self.model_name,
            content=text,
            task_type="retrieval_document"
        )
        embedding = np.array(result['embedding'], dtype=np.float32)
        return embedding

    def embed_batch(self, texts: List[str]) -> List[np.ndarray]:
        """Embed multiple texts (sequential for Gemini)"""
        # Note: Gemini doesn't have official batch API yet
        embeddings = []
        for text in texts:
            embeddings.append(self.embed_text(text))
        return embeddings

    @property
    def dimension(self) -> int:
        return self._dimension


class OpenAIEmbedding(EmbeddingProvider):
    """OpenAI embedding provider"""

    def __init__(self, api_key: str, model: str = "text-embedding-3-large"):
        """Initialize OpenAI embedder"""
        try:
            from openai import OpenAI
        except ImportError:
            raise ImportError("openai not installed. Run: pip install openai")

        self.client = OpenAI(api_key=api_key)
        self.model = model

        # Dimensions for different models
        dims = {
            "text-embedding-3-large": 3072,
            "text-embedding-3-small": 1536,
            "text-embedding-ada-002": 1536,
        }
        self._dimension = dims.get(model, 1536)
        logger.info(f"✓ OpenAI embedder initialized: {model}")

    def embed_text(self, text: str) -> np.ndarray:
        """Embed a single text"""
        response = self.client.embeddings.create(
            model=self.model,
            input=text
        )
        embedding = np.array(response.data[0].embedding, dtype=np.float32)
        return embedding

    def embed_batch(self, texts: List[str], batch_size: int = 100) -> List[np.ndarray]:
        """Embed multiple texts with batching"""
        embeddings = []

        for i in range(0, len(texts), batch_size):
            batch = texts[i:i + batch_size]
            response = self.client.embeddings.create(
                model=self.model,
                input=batch
            )
            batch_embeddings = [np.array(item.embedding, dtype=np.float32) for item in response.data]
            embeddings.extend(batch_embeddings)

        return embeddings

    @property
    def dimension(self) -> int:
        return self._dimension


def create_embedding_provider(
    provider: str = "sentence-transformers",
    model_name: Optional[str] = None,
    api_key: Optional[str] = None
) -> EmbeddingProvider:
    """
    Factory function to create embedding provider

    Args:
        provider: One of "sentence-transformers", "gemini", "openai"
        model_name: Specific model name
        api_key: API key for cloud providers

    Returns:
        EmbeddingProvider instance
    """
    provider = provider.lower()

    if provider == "sentence-transformers":
        model = model_name or "all-MiniLM-L6-v2"
        return SentenceTransformerEmbedding(model)

    elif provider == "gemini":
        if not api_key:
            raise ValueError("Gemini API key required")
        model = model_name or "models/text-embedding-004"
        return GeminiEmbedding(api_key, model)

    elif provider == "openai":
        if not api_key:
            raise ValueError("OpenAI API key required")
        model = model_name or "text-embedding-3-large"
        return OpenAIEmbedding(api_key, model)

    else:
        raise ValueError(f"Unknown provider: {provider}. Choose from: sentence-transformers, gemini, openai")
