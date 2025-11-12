"""
Vector store for semantic search using Qdrant and Gemini embeddings.
Lightweight implementation using Google's Gemini API for embeddings.
"""

import os
import asyncio
from typing import List, Dict, Any, Optional
from datetime import datetime
import google.generativeai as genai

try:
    from qdrant_client import QdrantClient
    from qdrant_client.models import Distance, VectorParams, PointStruct
    QDRANT_AVAILABLE = True
except ImportError:
    QDRANT_AVAILABLE = False
    print("Qdrant not available - vector search disabled")


class VectorStore:
    """
    Vector store for semantic search of research papers.
    Uses Qdrant for vector storage and Gemini API for embeddings.
    """

    def __init__(
        self,
        collection_name: str = "papers",
        storage_path: str = "./data/qdrant",
        gemini_api_key: Optional[str] = None
    ):
        """
        Initialize vector store.

        Args:
            collection_name: Name of the Qdrant collection
            storage_path: Path to store Qdrant data locally
            gemini_api_key: Gemini API key (reads from GEMINI_API_KEY env if not provided)
        """
        self.collection_name = collection_name
        self.storage_path = storage_path
        self.embedding_dim = 768  # Gemini embedding dimension

        # Initialize Gemini
        api_key = gemini_api_key or os.getenv("GEMINI_API_KEY")
        if not api_key:
            raise ValueError("GEMINI_API_KEY not found in environment")

        genai.configure(api_key=api_key)
        self.embedding_model = "models/text-embedding-004"

        if not QDRANT_AVAILABLE:
            print("Warning: Qdrant not installed, using in-memory storage only")
            self.client = None
            self.memory_store = []
            return

        # Initialize Qdrant client
        os.makedirs(storage_path, exist_ok=True)
        self.client = QdrantClient(path=storage_path)

        # Create collection if it doesn't exist
        self._initialize_collection()

    def _initialize_collection(self):
        """Create Qdrant collection if it doesn't exist."""
        try:
            collections = self.client.get_collections().collections
            collection_names = [col.name for col in collections]

            if self.collection_name not in collection_names:
                print(f"Creating collection: {self.collection_name}")
                self.client.create_collection(
                    collection_name=self.collection_name,
                    vectors_config=VectorParams(
                        size=self.embedding_dim,
                        distance=Distance.COSINE
                    )
                )
                print(f"Collection created with dimension {self.embedding_dim}")
            else:
                print(f"Collection {self.collection_name} already exists")
        except Exception as e:
            print(f"Error initializing collection: {e}")

    def generate_embedding(self, text: str) -> List[float]:
        """
        Generate embedding vector for text using Gemini.

        Args:
            text: Input text

        Returns:
            Embedding vector
        """
        try:
            result = genai.embed_content(
                model=self.embedding_model,
                content=text,
                task_type="retrieval_document"
            )
            return result['embedding']
        except Exception as e:
            print(f"Error generating embedding: {e}")
            # Return zero vector on error
            return [0.0] * self.embedding_dim

    def generate_embeddings_batch(self, texts: List[str]) -> List[List[float]]:
        """
        Generate embeddings for multiple texts efficiently.

        Args:
            texts: List of input texts

        Returns:
            List of embedding vectors
        """
        embeddings = []
        for text in texts:
            embeddings.append(self.generate_embedding(text))
        return embeddings

    async def index_paper(
        self,
        paper_id: str,
        title: str,
        abstract: str,
        authors: List[str],
        metadata: Dict[str, Any]
    ) -> bool:
        """
        Index a single paper in the vector store.

        Args:
            paper_id: Unique paper identifier
            title: Paper title
            abstract: Paper abstract
            authors: List of authors
            metadata: Additional metadata (source, venue, published_at, etc.)

        Returns:
            True if successful, False otherwise
        """
        try:
            # Combine title and abstract for better semantic representation
            text = f"{title} {abstract}"

            # Generate embedding
            embedding = self.generate_embedding(text)

            # Prepare payload
            payload = {
                "paper_id": paper_id,
                "title": title,
                "abstract": abstract,
                "authors": authors,
                **metadata
            }

            # Convert datetime to string for storage
            if "published_at" in payload and isinstance(payload["published_at"], datetime):
                payload["published_at"] = payload["published_at"].isoformat()

            # Upsert point to Qdrant
            point = PointStruct(
                id=hash(paper_id) % (2**63),  # Convert string ID to int
                vector=embedding,
                payload=payload
            )

            self.client.upsert(
                collection_name=self.collection_name,
                points=[point]
            )

            return True

        except Exception as e:
            print(f"Error indexing paper {paper_id}: {e}")
            return False

    async def index_papers_batch(self, papers: List[Dict[str, Any]]) -> int:
        """
        Index multiple papers efficiently.

        Args:
            papers: List of paper dictionaries with keys:
                   paper_id, title, abstract, authors, metadata

        Returns:
            Number of successfully indexed papers
        """
        try:
            # Generate all embeddings at once
            texts = [f"{p['title']} {p['abstract']}" for p in papers]
            embeddings = self.generate_embeddings_batch(texts)

            # Prepare points
            points = []
            for paper, embedding in zip(papers, embeddings):
                payload = {
                    "paper_id": paper["paper_id"],
                    "title": paper["title"],
                    "abstract": paper["abstract"],
                    "authors": paper.get("authors", []),
                    **paper.get("metadata", {})
                }

                # Convert datetime to string
                if "published_at" in payload and isinstance(payload["published_at"], datetime):
                    payload["published_at"] = payload["published_at"].isoformat()

                point = PointStruct(
                    id=hash(paper["paper_id"]) % (2**63),
                    vector=embedding,
                    payload=payload
                )
                points.append(point)

            # Batch upsert
            self.client.upsert(
                collection_name=self.collection_name,
                points=points
            )

            return len(points)

        except Exception as e:
            print(f"Error in batch indexing: {e}")
            return 0

    async def semantic_search(
        self,
        query: str,
        limit: int = 20,
        score_threshold: float = 0.5,
        filters: Optional[Dict[str, Any]] = None
    ) -> List[Dict[str, Any]]:
        """
        Perform semantic search using vector similarity.

        Args:
            query: Search query text
            limit: Maximum number of results
            score_threshold: Minimum similarity score (0-1)
            filters: Optional metadata filters

        Returns:
            List of search results with scores
        """
        try:
            # Generate query embedding
            query_embedding = self.generate_embedding(query)

            # Build filter if provided
            search_filter = None
            if filters:
                conditions = []
                for key, value in filters.items():
                    conditions.append(
                        FieldCondition(
                            key=key,
                            match=MatchValue(value=value)
                        )
                    )
                if conditions:
                    search_filter = Filter(must=conditions)

            # Perform search
            results = self.client.search(
                collection_name=self.collection_name,
                query_vector=query_embedding,
                limit=limit,
                score_threshold=score_threshold,
                query_filter=search_filter
            )

            # Format results
            formatted_results = []
            for result in results:
                paper_data = result.payload
                paper_data["similarity_score"] = result.score

                # Convert ISO string back to datetime if needed
                if "published_at" in paper_data and isinstance(paper_data["published_at"], str):
                    try:
                        paper_data["published_at"] = datetime.fromisoformat(paper_data["published_at"])
                    except:
                        pass

                formatted_results.append(paper_data)

            return formatted_results

        except Exception as e:
            print(f"Error in semantic search: {e}")
            return []

    async def hybrid_search(
        self,
        query: str,
        limit: int = 20,
        semantic_weight: float = 0.7,
        keyword_weight: float = 0.3,
        score_threshold: float = 0.3
    ) -> List[Dict[str, Any]]:
        """
        Perform hybrid search combining semantic and keyword search.

        Args:
            query: Search query
            limit: Maximum results
            semantic_weight: Weight for semantic similarity (0-1)
            keyword_weight: Weight for keyword matching (0-1)
            score_threshold: Minimum combined score

        Returns:
            List of search results with hybrid scores
        """
        # Get semantic search results
        semantic_results = await self.semantic_search(
            query=query,
            limit=limit * 2,  # Get more results for reranking
            score_threshold=0.0
        )

        # Perform simple keyword matching for reranking
        query_lower = query.lower()
        query_terms = set(query_lower.split())

        # Calculate hybrid scores
        for result in semantic_results:
            semantic_score = result.get("similarity_score", 0)

            # Simple keyword score based on term overlap
            title_lower = result.get("title", "").lower()
            abstract_lower = result.get("abstract", "").lower()
            text_terms = set(title_lower.split()) | set(abstract_lower.split())

            keyword_overlap = len(query_terms & text_terms) / max(len(query_terms), 1)
            keyword_score = keyword_overlap

            # Combine scores
            hybrid_score = (semantic_weight * semantic_score +
                          keyword_weight * keyword_score)

            result["hybrid_score"] = hybrid_score
            result["semantic_score"] = semantic_score
            result["keyword_score"] = keyword_score

        # Filter by threshold and sort by hybrid score
        filtered_results = [
            r for r in semantic_results
            if r.get("hybrid_score", 0) >= score_threshold
        ]
        filtered_results.sort(key=lambda x: x.get("hybrid_score", 0), reverse=True)

        return filtered_results[:limit]

    def get_collection_info(self) -> Dict[str, Any]:
        """Get information about the collection."""
        try:
            collection_info = self.client.get_collection(self.collection_name)
            return {
                "collection_name": self.collection_name,
                "vectors_count": collection_info.points_count,
                "embedding_dim": self.embedding_dim,
                "model_name": self.model_name,
                "status": "ready"
            }
        except Exception as e:
            return {
                "collection_name": self.collection_name,
                "error": str(e),
                "status": "error"
            }

    async def clear_collection(self) -> bool:
        """Clear all vectors from the collection."""
        try:
            self.client.delete_collection(self.collection_name)
            self._initialize_collection()
            return True
        except Exception as e:
            print(f"Error clearing collection: {e}")
            return False


# Global vector store instance
_vector_store: Optional[VectorStore] = None


def get_vector_store() -> VectorStore:
    """Get or create global vector store instance."""
    global _vector_store
    if _vector_store is None:
        _vector_store = VectorStore()
    return _vector_store
