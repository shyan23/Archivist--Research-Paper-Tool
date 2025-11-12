"""
Base search provider interface.
"""

from abc import ABC, abstractmethod
from typing import List
from ..models import SearchQuery, SearchResult


class SearchProvider(ABC):
    """Abstract base class for all search providers."""

    @abstractmethod
    def name(self) -> str:
        """Return the provider name."""
        pass

    @abstractmethod
    async def search(self, query: SearchQuery) -> List[SearchResult]:
        """
        Perform a search and return results.

        Args:
            query: SearchQuery object with search parameters

        Returns:
            List of SearchResult objects
        """
        pass

    @abstractmethod
    async def download_pdf(self, url: str, output_path: str) -> bool:
        """
        Download a PDF from the given URL.

        Args:
            url: PDF URL
            output_path: Path to save the PDF

        Returns:
            True if download successful, False otherwise
        """
        pass

    def _sanitize_filename(self, filename: str) -> str:
        """Sanitize a filename by removing invalid characters."""
        invalid_chars = ['/', '\\', ':', '*', '?', '"', '<', '>', '|', '\n', '\r', '\t']
        for char in invalid_chars:
            filename = filename.replace(char, '_')

        filename = filename.strip().strip('.')

        # Limit length
        if len(filename) > 200:
            filename = filename[:200]

        return filename
