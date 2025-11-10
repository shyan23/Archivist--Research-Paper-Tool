"""
RAG retrieval system
Retrieves relevant context from vector store for queries
"""
import logging
from typing import List, Optional, Dict, Any
from dataclasses import dataclass

from .embeddings import EmbeddingProvider
from .vector_store import VectorStore, SearchResult

logger = logging.getLogger(__name__)


@dataclass
class RetrievedContext:
    """Retrieved context with metadata"""
    chunks: List[SearchResult]
    context_text: str
    sources: List[str]
    sections: List[str]
    total_chunks: int


class Retriever:
    """RAG retrieval system"""

    def __init__(
        self,
        vector_store: VectorStore,
        embedding_provider: EmbeddingProvider,
        top_k: int = 5,
        score_threshold: float = 0.3,
        max_context_length: int = 8000
    ):
        """
        Initialize retriever

        Args:
            vector_store: Vector store to search
            embedding_provider: Embedding provider for query encoding
            top_k: Number of chunks to retrieve
            score_threshold: Minimum similarity score
            max_context_length: Maximum context length in characters
        """
        self.vector_store = vector_store
        self.embedding_provider = embedding_provider
        self.top_k = top_k
        self.score_threshold = score_threshold
        self.max_context_length = max_context_length

    def retrieve(
        self,
        query: str,
        filter: Optional[Dict[str, Any]] = None,
        top_k: Optional[int] = None
    ) -> RetrievedContext:
        """
        Retrieve relevant context for a query

        Args:
            query: User query
            filter: Optional filter (e.g., {"source": "paper_name"})
            top_k: Override default top_k

        Returns:
            RetrievedContext with chunks and formatted context
        """
        if not query or not query.strip():
            raise ValueError("Empty query provided")

        top_k = top_k or self.top_k

        logger.info(f"🔍 Retrieving context for query: {self._truncate(query, 60)}")

        # Generate query embedding
        query_embedding = self.embedding_provider.embed_text(query)

        # Search vector store
        results = self.vector_store.search(
            query_embedding=query_embedding,
            top_k=top_k * 2,  # Get more for filtering
            filter=filter
        )

        # Filter by score threshold
        filtered_results = [
            r for r in results
            if r.score >= self.score_threshold
        ][:top_k]

        if not filtered_results:
            logger.warning("⚠️  No relevant chunks found above threshold")
            return RetrievedContext(
                chunks=[],
                context_text="No relevant context found.",
                sources=[],
                sections=[],
                total_chunks=0
            )

        logger.info(f"✓ Retrieved {len(filtered_results)} relevant chunks")

        # Build context
        context = self._build_context(filtered_results)

        return context

    def retrieve_from_paper(
        self,
        query: str,
        paper_title: str,
        top_k: Optional[int] = None
    ) -> RetrievedContext:
        """
        Retrieve context from a specific paper

        Args:
            query: User query
            paper_title: Paper to search in
            top_k: Override default top_k

        Returns:
            RetrievedContext
        """
        filter = {"source": paper_title}
        return self.retrieve(query, filter, top_k)

    def retrieve_multi_paper(
        self,
        query: str,
        paper_titles: List[str],
        top_k: Optional[int] = None
    ) -> RetrievedContext:
        """
        Retrieve context from multiple papers

        Args:
            query: User query
            paper_titles: List of papers to search
            top_k: Override default top_k

        Returns:
            RetrievedContext with chunks from all papers
        """
        top_k = top_k or self.top_k

        # Retrieve from each paper
        all_results = []

        for paper_title in paper_titles:
            try:
                context = self.retrieve_from_paper(query, paper_title, top_k)
                all_results.extend(context.chunks)
            except Exception as e:
                logger.warning(f"⚠️  Failed to retrieve from {paper_title}: {e}")

        if not all_results:
            logger.warning("⚠️  No relevant chunks found in any paper")
            return RetrievedContext(
                chunks=[],
                context_text="No relevant context found.",
                sources=[],
                sections=[],
                total_chunks=0
            )

        # Sort by score and take top_k
        all_results.sort(key=lambda x: x.score, reverse=True)
        all_results = all_results[:top_k]

        # Build context
        context = self._build_context(all_results)

        return context

    def _build_context(self, results: List[SearchResult]) -> RetrievedContext:
        """Build formatted context from search results"""
        # Track unique sources and sections
        sources = set()
        sections = set()

        # Build context text
        context_parts = []
        current_length = 0

        for i, result in enumerate(results, 1):
            doc = result.document

            # Track metadata
            sources.add(doc.metadata.get('source', 'unknown'))
            if section := doc.metadata.get('section'):
                sections.add(section)

            # Format chunk with citation
            chunk_header = f"\n[Chunk {i}"

            if source := doc.metadata.get('source'):
                chunk_header += f" | Source: {source}"

            if section := doc.metadata.get('section'):
                chunk_header += f" | Section: {section}"

            chunk_header += f" | Score: {result.score:.3f}]\n"

            chunk_text = chunk_header + doc.text + "\n"

            # Check context length limit
            if self.max_context_length > 0:
                if current_length + len(chunk_text) > self.max_context_length:
                    logger.info(f"⚠️  Reached max context length, truncating at {i-1} chunks")
                    break

            context_parts.append(chunk_text)
            current_length += len(chunk_text)

        # Combine context
        context_text = "".join(context_parts)

        return RetrievedContext(
            chunks=results[:len(context_parts)],
            context_text=context_text,
            sources=sorted(list(sources)),
            sections=sorted(list(sections)),
            total_chunks=len(context_parts)
        )

    @staticmethod
    def _truncate(text: str, max_len: int) -> str:
        """Truncate text to max length"""
        if len(text) <= max_len:
            return text
        return text[:max_len] + "..."
