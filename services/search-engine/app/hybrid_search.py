"""
Hybrid search orchestrator - arXiv search with fuzzy matching.
Simplified version focusing on arXiv API with intelligent fuzzy matching.
"""

from typing import List, Dict, Any
from .models import SearchQuery, SearchResult
from .providers import ArxivProvider
from .fuzzy_search import get_fuzzy_matcher
import asyncio


class HybridSearchOrchestrator:
    """
    Orchestrates hybrid search combining:
    - arXiv keyword search
    - Fuzzy matching for typo tolerance
    - Abbreviation expansion
    """

    def __init__(self):
        """Initialize search orchestrator."""
        self.arxiv_provider = ArxivProvider()
        self.fuzzy_matcher = get_fuzzy_matcher()

    async def search(self, query: SearchQuery) -> List[SearchResult]:
        """
        Perform search using arXiv with fuzzy matching.

        Args:
            query: SearchQuery object

        Returns:
            List of SearchResult objects ranked by relevance
        """
        # Expand abbreviations in query
        expanded_queries = self.fuzzy_matcher.expand_query_terms(query.query)

        # Search arXiv with original and expanded queries
        all_results = []
        seen_ids = set()

        # Search with original query
        original_results = await self.arxiv_provider.search(query)
        for result in original_results:
            if result.id not in seen_ids:
                seen_ids.add(result.id)
                all_results.append(result)

        # Search with expanded queries if different from original
        for expanded_query in expanded_queries[1:]:  # Skip first (original)
            if len(all_results) >= query.max_results:
                break

            expanded_search = SearchQuery(
                query=expanded_query,
                max_results=query.max_results - len(all_results),
                sources=query.sources,
                start_date=query.start_date,
                end_date=query.end_date
            )

            expanded_results = await self.arxiv_provider.search(expanded_search)
            for result in expanded_results:
                if result.id not in seen_ids:
                    seen_ids.add(result.id)
                    all_results.append(result)

        # Apply fuzzy matching to rank results
        papers_dict = [
            {
                "title": r.title,
                "abstract": r.abstract,
                "authors": r.authors,
                "result": r
            }
            for r in all_results
        ]

        # Calculate fuzzy scores
        fuzzy_results = self.fuzzy_matcher.fuzzy_search_papers(
            query=query.query,
            papers=papers_dict,
            search_fields=["title", "abstract"],
            limit=query.max_results
        )

        # Extract results with fuzzy scores
        final_results = []
        for paper_dict, fuzzy_score in fuzzy_results:
            result = paper_dict["result"]
            result.fuzzy_score = float(fuzzy_score)
            result.relevance_score = fuzzy_score / 100.0  # Normalize to 0-1
            final_results.append(result)

        return final_results[:query.max_results]


# Global orchestrator instance
_orchestrator: HybridSearchOrchestrator = None


def get_search_orchestrator() -> HybridSearchOrchestrator:
    """Get or create global search orchestrator instance."""
    global _orchestrator
    if _orchestrator is None:
        _orchestrator = HybridSearchOrchestrator()
    return _orchestrator
