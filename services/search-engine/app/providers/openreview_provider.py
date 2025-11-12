"""
OpenReview search provider for ICLR, NeurIPS, and other conferences.
"""

import aiohttp
import aiofiles
from typing import List, Dict, Any
from datetime import datetime
from .base import SearchProvider
from ..models import SearchQuery, SearchResult


class OpenReviewProvider(SearchProvider):
    """Provider for searching OpenReview.net papers (ICLR, NeurIPS, etc.)."""

    def __init__(self):
        self.base_url = "https://api2.openreview.net"
        self.venues = [
            "ICLR.cc/2025/Conference",
            "NeurIPS.cc/2025/Conference",
            "ICLR.cc/2024/Conference",
            "NeurIPS.cc/2024/Conference",
            "ICLR.cc/2023/Conference",
            "NeurIPS.cc/2023/Conference",
        ]

    def name(self) -> str:
        return "OpenReview"

    async def search(self, query: SearchQuery) -> List[SearchResult]:
        """
        Search OpenReview using their API.

        Args:
            query: SearchQuery object

        Returns:
            List of SearchResult objects
        """
        all_results = []
        seen_ids = set()

        try:
            async with aiohttp.ClientSession() as session:
                for venue in self.venues:
                    # Search by content (title and abstract)
                    results = await self._search_venue(session, venue, query)

                    # Deduplicate by ID
                    for result in results:
                        if result.id not in seen_ids:
                            seen_ids.add(result.id)
                            all_results.append(result)

                    # Stop if we have enough results
                    if len(all_results) >= query.max_results:
                        break

        except Exception as e:
            print(f"OpenReview search error: {e}")

        # Sort by date (newest first) and limit
        all_results.sort(key=lambda x: x.published_at, reverse=True)
        return all_results[:query.max_results]

    async def _search_venue(
        self,
        session: aiohttp.ClientSession,
        venue: str,
        query: SearchQuery
    ) -> List[SearchResult]:
        """Search a specific venue on OpenReview."""
        results = []

        try:
            # Get all accepted papers for this venue
            # For accepted papers, use the Decision invitation
            invitation = f"{venue}/-/Decision"

            # Try to get submissions
            url = f"{self.base_url}/notes"
            params = {
                "invitation": invitation,
                "limit": 100,  # Get more papers per venue
                "offset": 0
            }

            headers = {"User-Agent": "Archivist/1.0"}

            async with session.get(url, params=params, headers=headers) as response:
                if response.status != 200:
                    return results

                data = await response.json()
                notes = data.get("notes", [])

                # Filter by search query in title or abstract
                query_lower = query.query.lower()

                for note in notes:
                    content = note.get("content", {})

                    # Extract title and abstract
                    title = self._extract_value(content.get("title", {}))
                    abstract = self._extract_value(content.get("abstract", {}))

                    if not title:
                        continue

                    # Check if query matches title or abstract
                    if query_lower not in title.lower() and query_lower not in abstract.lower():
                        continue

                    # Convert to SearchResult
                    result = self._convert_note(note, venue)
                    if result:
                        # Apply date filters
                        if query.start_date and result.published_at < query.start_date:
                            continue
                        if query.end_date and result.published_at > query.end_date:
                            continue

                        results.append(result)

        except Exception as e:
            print(f"Error searching venue {venue}: {e}")

        return results

    def _convert_note(self, note: Dict[str, Any], venue: str) -> SearchResult:
        """Convert an OpenReview note to a SearchResult."""
        try:
            content = note.get("content", {})

            # Extract fields
            title = self._extract_value(content.get("title", {}))
            abstract = self._extract_value(content.get("abstract", {}))

            # Extract authors
            authors_value = content.get("authors", {}).get("value", [])
            if isinstance(authors_value, list):
                authors = [str(a).strip() for a in authors_value if a]
            else:
                authors = []

            # Parse date (milliseconds to datetime)
            pdate = note.get("pdate", 0) or note.get("cdate", 0)
            published_at = datetime.fromtimestamp(pdate / 1000.0)

            # Extract venue name
            venue_name = self._extract_value(content.get("venue", {}))
            if not venue_name:
                venue_name = self._parse_venue_name(venue)

            # Build URLs
            note_id = note.get("id", "")
            forum_id = note.get("forum", note_id)
            source_url = f"https://openreview.net/forum?id={forum_id}"
            pdf_url = f"https://openreview.net/pdf?id={note_id}"

            # Extract keywords
            keywords_value = content.get("keywords", {}).get("value", [])
            if isinstance(keywords_value, list):
                categories = [str(k) for k in keywords_value if k]
            else:
                categories = []

            return SearchResult(
                title=title,
                authors=authors,
                abstract=abstract,
                published_at=published_at,
                pdf_url=pdf_url,
                source_url=source_url,
                source="OpenReview",
                venue=venue_name,
                id=note_id,
                categories=categories
            )

        except Exception as e:
            print(f"Error converting note: {e}")
            return None

    def _extract_value(self, field: Dict[str, Any]) -> str:
        """Extract string value from OpenReview field."""
        if not field:
            return ""

        value = field.get("value", "")
        if isinstance(value, str):
            return value.strip()
        elif isinstance(value, list) and len(value) > 0:
            return str(value[0]).strip()

        return ""

    def _parse_venue_name(self, invitation: str) -> str:
        """Parse venue name from invitation string."""
        # Example: "ICLR.cc/2025/Conference" -> "ICLR 2025"
        parts = invitation.split("/")
        if len(parts) >= 2:
            venue = parts[0].split(".")[0]
            year = parts[1]
            return f"{venue} {year}"
        return invitation

    async def download_pdf(self, url: str, output_path: str) -> bool:
        """
        Download PDF from OpenReview.

        Args:
            url: PDF URL
            output_path: Path to save the PDF

        Returns:
            True if successful, False otherwise
        """
        try:
            headers = {"User-Agent": "Archivist/1.0"}

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
