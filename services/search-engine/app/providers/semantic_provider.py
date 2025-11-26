"""
Semantic search provider using vector embeddings and Qdrant.
"""

from typing import List
from datetime import datetime
from .base import SearchProvider
from ..models import SearchQuery, SearchResult
from ..vector_store import get_vector_store
from ..fuzzy_search import get_fuzzy_matcher


class SemanticProvider(SearchProvider):
    """Provider for semantic search using vector embeddings."""

    def __init__(self):
        """Initialize semantic provider."""
        self.vector_store = get_vector_store()
        self.fuzzy_matcher = get_fuzzy_matcher()

    def name(self) -> str:
        return "Semantic"

    async def search(self, query: SearchQuery) -> List[SearchResult]:
        """
        Perform semantic search using vector similarity.

        Args:
            query: SearchQuery object

        Returns:
            List of SearchResult objects
        """
        try:
            # Expand query if needed (handle abbreviations)
            expanded_queries = self.fuzzy_matcher.expand_query_terms(query.query)

            # Perform semantic search on the main query
            results = await self.vector_store.semantic_search(
                query=query.query,
                limit=query.max_results,
                score_threshold=0.3  # Lower threshold for semantic search
            )

            # Convert to SearchResult objects
            search_results = []
            for result_data in results:
                # Apply date filters if specified
                published_at = result_data.get("published_at")
                if isinstance(published_at, str):
                    try:
                        published_at = datetime.fromisoformat(published_at)
                    except:
                        published_at = datetime.now()

                if query.start_date and published_at < query.start_date:
                    continue
                if query.end_date and published_at > query.end_date:
                    continue

                search_result = SearchResult(
                    title=result_data.get("title", ""),
                    authors=result_data.get("authors", []),
                    abstract=result_data.get("abstract", ""),
                    published_at=published_at,
                    pdf_url=result_data.get("pdf_url", ""),
                    source_url=result_data.get("source_url", ""),
                    source="Semantic",
                    venue=result_data.get("venue", ""),
                    id=result_data.get("paper_id", result_data.get("id", "")),
                    categories=result_data.get("categories", []),
                    similarity_score=result_data.get("similarity_score", 0.0),
                    relevance_score=result_data.get("similarity_score", 0.0)
                )
                search_results.append(search_result)

            return search_results

        except Exception as e:
            print(f"Semantic search error: {e}")
            return []

    async def download_pdf(self, url: str, output_path: str) -> bool:
        """
        Semantic provider doesn't handle downloads directly.
        Delegates to the original source provider.
        """
        print("Semantic provider does not handle PDF downloads")
        return False

    async def hybrid_search(
        self,
        query: SearchQuery,
        keyword_results: List[SearchResult]
    ) -> List[SearchResult]:
        """
        Perform hybrid search combining semantic and keyword results.

        Args:
            query: SearchQuery object
            keyword_results: Results from keyword search

        Returns:
            Combined and reranked results
        """
        try:
            # Get semantic results
            semantic_results = await self.search(query)

            # Merge results by paper ID
            merged_results = {}

            # Add keyword results
            for result in keyword_results:
                paper_id = result.id
                merged_results[paper_id] = {
                    "result": result,
                    "keyword_score": 1.0,  # Full score for keyword match
                    "semantic_score": 0.0
                }

            # Add/update with semantic results
            for result in semantic_results:
                paper_id = result.id
                if paper_id in merged_results:
                    # Paper found in both searches - update semantic score
                    merged_results[paper_id]["semantic_score"] = result.similarity_score or 0.0
                else:
                    # Paper only in semantic search
                    merged_results[paper_id] = {
                        "result": result,
                        "keyword_score": 0.0,
                        "semantic_score": result.similarity_score or 0.0
                    }

            # Calculate hybrid scores
            semantic_weight = query.semantic_weight
            keyword_weight = 1.0 - semantic_weight

            final_results = []
            for paper_id, data in merged_results.items():
                result = data["result"]
                hybrid_score = (
                    keyword_weight * data["keyword_score"] +
                    semantic_weight * data["semantic_score"]
                )

                # Update relevance score
                result.relevance_score = hybrid_score
                result.similarity_score = data["semantic_score"]

                final_results.append(result)

            # Sort by relevance score
            final_results.sort(key=lambda x: x.relevance_score or 0, reverse=True)

            return final_results[:query.max_results]

        except Exception as e:
            print(f"Hybrid search error: {e}")
            return keyword_results  # Fallback to keyword results
