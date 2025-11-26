"""
arXiv search provider using the official arxiv Python package.
"""

import arxiv
import aiofiles
import aiohttp
from typing import List
from datetime import datetime, timezone
from .base import SearchProvider
from ..models import SearchQuery, SearchResult


class ArxivProvider(SearchProvider):
    """Provider for searching arXiv.org papers."""

    def name(self) -> str:
        return "arXiv"

    async def search(self, query: SearchQuery) -> List[SearchResult]:
        """
        Search arXiv using the official arxiv Python package.

        Args:
            query: SearchQuery object

        Returns:
            List of SearchResult objects
        """
        try:
            # Create arXiv client
            client = arxiv.Client()

            # Build search query
            search = arxiv.Search(
                query=query.query,
                max_results=query.max_results,
                sort_by=arxiv.SortCriterion.SubmittedDate,
                sort_order=arxiv.SortOrder.Descending
            )

            # Execute search
            results = []
            for paper in client.results(search):
                # Apply date filters if specified
                published = paper.published
                if published.tzinfo is None:
                    published = published.replace(tzinfo=timezone.utc)

                # Ensure query dates have timezone for comparison
                if query.start_date:
                    start_date = query.start_date
                    if start_date.tzinfo is None:
                        start_date = start_date.replace(tzinfo=timezone.utc)
                    if published < start_date:
                        continue

                if query.end_date:
                    end_date = query.end_date
                    if end_date.tzinfo is None:
                        end_date = end_date.replace(tzinfo=timezone.utc)
                    if published > end_date:
                        continue

                # Extract authors
                authors = [author.name for author in paper.authors]

                # Extract categories
                categories = list(paper.categories)

                # Get PDF URL (construct if not provided)
                pdf_url = paper.pdf_url
                if not pdf_url:
                    # Construct PDF URL from entry_id
                    # Format: http://arxiv.org/abs/2409.15512 -> http://arxiv.org/pdf/2409.15512.pdf
                    pdf_url = paper.entry_id.replace('/abs/', '/pdf/') + '.pdf'

                # Ensure published_at has timezone info (required for Go client)
                published_at = paper.published
                if published_at.tzinfo is None:
                    # If naive datetime, assume UTC
                    published_at = published_at.replace(tzinfo=timezone.utc)

                # Create SearchResult
                result = SearchResult(
                    title=paper.title,
                    authors=authors,
                    abstract=paper.summary,
                    published_at=published_at,
                    pdf_url=pdf_url,
                    source_url=paper.entry_id,
                    source="arXiv",
                    venue="arXiv",
                    id=paper.get_short_id(),
                    categories=categories
                )
                results.append(result)

            return results

        except Exception as e:
            print(f"arXiv search error: {e}")
            return []

    async def download_pdf(self, url: str, output_path: str) -> bool:
        """
        Download PDF from arXiv.

        Args:
            url: PDF URL
            output_path: Path to save the PDF

        Returns:
            True if successful, False otherwise
        """
        try:
            async with aiohttp.ClientSession() as session:
                async with session.get(url) as response:
                    if response.status == 200:
                        async with aiofiles.open(output_path, 'wb') as f:
                            await f.write(await response.read())
                        return True
                    else:
                        print(f"Failed to download PDF: status {response.status}")
                        return False
        except Exception as e:
            print(f"Download error: {e}")
            return False
