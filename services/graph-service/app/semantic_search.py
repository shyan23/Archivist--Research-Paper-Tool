"""
Semantic Search - Vector embeddings and similarity search for papers using Qdrant
"""

import logging
import numpy as np
from typing import List, Dict, Any, Optional
import google.generativeai as genai
from qdrant_client import QdrantClient
from qdrant_client.models import Distance, VectorParams, PointStruct
import os

logger = logging.getLogger(__name__)


class SemanticSearchEngine:
    """Handles embedding generation and semantic search using Qdrant"""

    def __init__(
        self,
        api_key: Optional[str] = None,
        embedding_model: str = "models/text-embedding-004",
        qdrant_url: str = "http://localhost:6333",
        collection_name: str = "papers"
    ):
        self.api_key = api_key or os.getenv("GEMINI_API_KEY")

        if not self.api_key:
            raise ValueError("GEMINI_API_KEY not found in environment")

        genai.configure(api_key=self.api_key)
        self.embedding_model = embedding_model
        self.collection_name = collection_name

        # Initialize Qdrant client
        self.qdrant_client = QdrantClient(url=qdrant_url)

        logger.info(f"✅ Semantic search engine initialized")
        logger.info(f"  - Embedding model: {embedding_model}")
        logger.info(f"  - Qdrant: {qdrant_url}")
        logger.info(f"  - Collection: {collection_name}")

    async def generate_paper_embedding(self, title: str, abstract: str, methods: List[str]) -> List[float]:
        """
        Generate embedding vector for a paper

        Combines title, abstract, and methods into a single semantic representation
        """
        # Create combined text for embedding
        combined_text = f"""Title: {title}
Abstract: {abstract}
Methods: {', '.join(methods)}"""

        try:
            # Generate embedding using Gemini
            result = genai.embed_content(
                model=self.embedding_model,
                content=combined_text,
                task_type="retrieval_document"
            )

            embedding = result['embedding']

            logger.info(f"  ✅ Generated embedding (dim: {len(embedding)})")

            return embedding

        except Exception as e:
            logger.error(f"  ❌ Embedding generation failed: {e}")
            return []

    async def store_paper_embedding(
        self,
        paper_id: str,
        title: str,
        embedding: List[float],
        metadata: Dict[str, Any]
    ) -> bool:
        """
        Store paper embedding in Qdrant

        Args:
            paper_id: Unique paper identifier
            title: Paper title
            embedding: Embedding vector
            metadata: Additional metadata (authors, year, etc.)

        Returns:
            True if successful
        """
        try:
            # Create point for Qdrant
            point = PointStruct(
                id=hash(paper_id) % (2**63),  # Convert to positive integer
                vector=embedding,
                payload={
                    "paper_id": paper_id,
                    "title": title,
                    "source": "graph",
                    **metadata
                }
            )

            # Upsert to Qdrant
            self.qdrant_client.upsert(
                collection_name=self.collection_name,
                points=[point]
            )

            logger.info(f"  ✅ Stored embedding for: {title}")
            return True

        except Exception as e:
            logger.error(f"  ❌ Failed to store embedding: {e}")
            return False

    async def generate_query_embedding(self, query: str) -> List[float]:
        """Generate embedding for a search query"""
        try:
            result = genai.embed_content(
                model=self.embedding_model,
                content=query,
                task_type="retrieval_query"
            )

            return result['embedding']

        except Exception as e:
            logger.error(f"  ❌ Query embedding failed: {e}")
            return []

    @staticmethod
    def cosine_similarity(vec1: List[float], vec2: List[float]) -> float:
        """Calculate cosine similarity between two vectors"""
        if not vec1 or not vec2:
            return 0.0

        v1 = np.array(vec1)
        v2 = np.array(vec2)

        # Normalize vectors
        v1_norm = v1 / (np.linalg.norm(v1) + 1e-8)
        v2_norm = v2 / (np.linalg.norm(v2) + 1e-8)

        # Compute cosine similarity
        similarity = np.dot(v1_norm, v2_norm)

        return float(similarity)

    async def search_similar_papers(
        self,
        query: str,
        top_k: int = 10,
        score_threshold: float = 0.7
    ) -> List[Dict[str, Any]]:
        """
        Search for similar papers using semantic search

        Args:
            query: Natural language query
            top_k: Number of results
            score_threshold: Minimum similarity score

        Returns:
            List of similar papers with scores
        """
        try:
            # Generate query embedding
            query_embedding = await self.generate_query_embedding(query)

            if not query_embedding:
                return []

            # Search in Qdrant
            results = self.qdrant_client.search(
                collection_name=self.collection_name,
                query_vector=query_embedding,
                limit=top_k,
                score_threshold=score_threshold,
                query_filter={
                    "must": [
                        {"key": "source", "match": {"value": "graph"}}
                    ]
                }
            )

            # Format results
            similar_papers = []
            for result in results:
                similar_papers.append({
                    "title": result.payload.get("title"),
                    "paper_id": result.payload.get("paper_id"),
                    "similarity_score": result.score,
                    "metadata": {
                        k: v for k, v in result.payload.items()
                        if k not in ["title", "paper_id", "source"]
                    }
                })

            logger.info(f"  ✅ Found {len(similar_papers)} similar papers")

            return similar_papers

        except Exception as e:
            logger.error(f"  ❌ Semantic search failed: {e}")
            return []

    async def find_similar_to_paper(
        self,
        paper_title: str,
        top_k: int = 10,
        score_threshold: float = 0.85
    ) -> List[Dict[str, Any]]:
        """
        Find papers similar to a given paper

        Args:
            paper_title: Title of the source paper
            top_k: Number of similar papers to return
            score_threshold: Minimum similarity (0.85 = moderate threshold)

        Returns:
            List of similar papers
        """
        try:
            # Search for the source paper first
            source_results = self.qdrant_client.scroll(
                collection_name=self.collection_name,
                scroll_filter={
                    "must": [
                        {"key": "title", "match": {"value": paper_title}},
                        {"key": "source", "match": {"value": "graph"}}
                    ]
                },
                limit=1
            )

            if not source_results[0]:
                logger.warning(f"  Paper not found in Qdrant: {paper_title}")
                return []

            source_point = source_results[0][0]

            # Get the embedding from the point
            # Since we can't directly get the vector, we need to search with a dummy query
            # or store it in payload (not recommended for large vectors)
            # For now, let's use the recommend API

            similar = self.qdrant_client.recommend(
                collection_name=self.collection_name,
                positive=[source_point.id],
                limit=top_k + 1,  # +1 to exclude source paper
                score_threshold=score_threshold
            )

            # Format and filter out source paper
            similar_papers = []
            for result in similar:
                if result.payload.get("title") != paper_title:
                    similar_papers.append({
                        "title": result.payload.get("title"),
                        "paper_id": result.payload.get("paper_id"),
                        "similarity_score": result.score,
                        "metadata": {
                            k: v for k, v in result.payload.items()
                            if k not in ["title", "paper_id", "source"]
                        }
                    })

            logger.info(f"  ✅ Found {len(similar_papers)} papers similar to '{paper_title}'")

            return similar_papers[:top_k]

        except Exception as e:
            logger.error(f"  ❌ Failed to find similar papers: {e}")
            return []

    def create_similarity_edges(
        self,
        papers: List[Dict[str, Any]],
        similarity_threshold: float = 0.85
    ) -> List[Dict[str, Any]]:
        """
        Create SIMILAR_TO relationships between papers

        Args:
            papers: List of papers with embeddings
            similarity_threshold: Minimum similarity to create edge

        Returns:
            List of similarity edges {source, target, score}
        """
        edges = []

        for i, paper1 in enumerate(papers):
            if not paper1.get("embedding"):
                continue

            for paper2 in papers[i+1:]:  # Avoid duplicates
                if not paper2.get("embedding"):
                    continue

                similarity = self.cosine_similarity(
                    paper1["embedding"],
                    paper2["embedding"]
                )

                if similarity >= similarity_threshold:
                    edges.append({
                        "source": paper1["title"],
                        "target": paper2["title"],
                        "similarity": similarity
                    })

        logger.info(f"  ✅ Created {len(edges)} similarity edges")

        return edges
