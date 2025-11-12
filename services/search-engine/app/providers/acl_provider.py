"""
ACL Anthology search provider for EMNLP, ACL, and other NLP conferences.
"""

import aiohttp
import aiofiles
from typing import List, Dict, Any
from datetime import datetime
from bs4 import BeautifulSoup
from .base import SearchProvider
from ..models import SearchQuery, SearchResult


class ACLProvider(SearchProvider):
    """Provider for searching ACL Anthology papers."""

    def __init__(self):
        self.base_url = "https://aclanthology.org"
        self.search_url = f"{self.base_url}/search/"

    def name(self) -> str:
        return "ACL"

    async def search(self, query: SearchQuery) -> List[SearchResult]:
        """
        Search ACL Anthology.

        Args:
            query: SearchQuery object

        Returns:
            List of SearchResult objects
        """
        results = []

        try:
            async with aiohttp.ClientSession() as session:
                # ACL Anthology search parameters
                params = {
                    "q": query.query,
                    "f": "title|abstract",  # Search in title and abstract
                }

                headers = {
                    "User-Agent": "Mozilla/5.0 (compatible; Archivist/1.0)"
                }

                async with session.get(
                    self.search_url,
                    params=params,
                    headers=headers
                ) as response:
                    if response.status != 200:
                        print(f"ACL search failed with status {response.status}")
                        return results

                    html = await response.text()
                    soup = BeautifulSoup(html, 'lxml')

                    # Parse search results
                    paper_items = soup.find_all("p", class_="d-sm-flex align-items-stretch")

                    for item in paper_items[:query.max_results]:
                        result = await self._parse_paper_item(item, session, query)
                        if result:
                            results.append(result)

        except Exception as e:
            print(f"ACL search error: {e}")

        return results

    async def _parse_paper_item(
        self,
        item,
        session: aiohttp.ClientSession,
        query: SearchQuery
    ) -> SearchResult:
        """Parse a single paper item from search results."""
        try:
            # Extract title and link
            title_tag = item.find("strong", class_="align-middle")
            if not title_tag:
                return None

            title_link = title_tag.find("a")
            if not title_link:
                return None

            title = title_link.get_text(strip=True)
            paper_url = self.base_url + title_link.get("href", "")

            # Extract paper ID from URL (e.g., /2025.emnlp-main.123/)
            paper_id = title_link.get("href", "").strip("/").split("/")[-1]

            # Extract authors
            authors = []
            author_span = item.find("span", class_="d-block")
            if author_span:
                author_links = author_span.find_all("a")
                authors = [a.get_text(strip=True) for a in author_links]

            # Extract venue info
            venue = ""
            venue_link = item.find("a", class_="badge badge-primary align-middle mr-1")
            if venue_link:
                venue = venue_link.get_text(strip=True)

            # Extract year from venue or paper ID
            year = self._extract_year(venue, paper_id)
            published_at = datetime(year, 1, 1) if year else datetime.now()

            # Apply date filters
            if query.start_date and published_at < query.start_date:
                return None
            if query.end_date and published_at > query.end_date:
                return None

            # Fetch paper details to get abstract
            abstract = await self._fetch_abstract(session, paper_url)

            # Build PDF URL
            pdf_url = f"{self.base_url}/{paper_id}.pdf"

            return SearchResult(
                title=title,
                authors=authors,
                abstract=abstract,
                published_at=published_at,
                pdf_url=pdf_url,
                source_url=paper_url,
                source="ACL",
                venue=venue,
                id=paper_id,
                categories=[venue] if venue else []
            )

        except Exception as e:
            print(f"Error parsing ACL paper item: {e}")
            return None

    async def _fetch_abstract(self, session: aiohttp.ClientSession, url: str) -> str:
        """Fetch abstract from paper page."""
        try:
            headers = {
                "User-Agent": "Mozilla/5.0 (compatible; Archivist/1.0)"
            }

            async with session.get(url, headers=headers) as response:
                if response.status != 200:
                    return ""

                html = await response.text()
                soup = BeautifulSoup(html, 'lxml')

                # Find abstract
                abstract_card = soup.find("div", class_="card-body acl-abstract")
                if abstract_card:
                    abstract_span = abstract_card.find("span", class_="d-block")
                    if abstract_span:
                        return abstract_span.get_text(strip=True)

        except Exception as e:
            print(f"Error fetching abstract: {e}")

        return ""

    def _extract_year(self, venue: str, paper_id: str) -> int:
        """Extract year from venue string or paper ID."""
        try:
            # Try to extract from paper ID (e.g., "2025.emnlp-main.123")
            if "." in paper_id:
                year_str = paper_id.split(".")[0]
                return int(year_str)

            # Try to extract from venue string
            import re
            year_match = re.search(r'20\d{2}', venue)
            if year_match:
                return int(year_match.group())

        except Exception:
            pass

        return datetime.now().year

    async def download_pdf(self, url: str, output_path: str) -> bool:
        """
        Download PDF from ACL Anthology.

        Args:
            url: PDF URL
            output_path: Path to save the PDF

        Returns:
            True if successful, False otherwise
        """
        try:
            headers = {
                "User-Agent": "Mozilla/5.0 (compatible; Archivist/1.0)"
            }

            async with aiohttp.ClientSession() as session:
                async with session.get(url, headers=headers) as response:
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
