"""
Metadata Extractor - Extracts paper metadata from LaTeX using Gemini LLM
"""

import os
import json
import logging
import re
from typing import List, Dict, Any, Optional
from dataclasses import dataclass, field
import google.generativeai as genai

logger = logging.getLogger(__name__)


@dataclass
class PaperMetadata:
    """Extracted paper metadata"""
    title: str
    authors: List[str] = field(default_factory=list)
    affiliations: List[str] = field(default_factory=list)
    year: int = 0
    abstract: str = ""
    venue: str = ""
    venue_short: str = ""
    methods: List[str] = field(default_factory=list)
    datasets: List[str] = field(default_factory=list)
    metrics: List[str] = field(default_factory=list)
    research_field: str = ""


class MetadataExtractor:
    """Extracts metadata from LaTeX content using Gemini"""

    def __init__(self, api_key: Optional[str] = None, model: str = "gemini-2.0-flash-exp"):
        self.api_key = api_key or os.getenv("GEMINI_API_KEY")

        if not self.api_key:
            raise ValueError("GEMINI_API_KEY not found in environment")

        genai.configure(api_key=self.api_key)
        self.model = genai.GenerativeModel(model)

        logger.info(f"✅ Metadata extractor initialized with model: {model}")

    async def extract_metadata(self, latex_content: str, paper_title: str) -> PaperMetadata:
        """Extract metadata from LaTeX content"""

        # Try simple extraction first (faster)
        metadata = self._simple_extraction(latex_content, paper_title)

        # If simple extraction fails or is incomplete, use LLM
        if not metadata.authors or not metadata.year:
            logger.info("  🤖 Using Gemini for metadata extraction...")
            llm_metadata = await self._llm_extraction(latex_content, paper_title)

            # Merge results (prefer LLM results)
            metadata = self._merge_metadata(metadata, llm_metadata)

        logger.info(f"  ✅ Extracted metadata: {len(metadata.authors)} authors, {len(metadata.methods)} methods")

        return metadata

    def _simple_extraction(self, latex_content: str, paper_title: str) -> PaperMetadata:
        """Fast regex-based extraction"""
        metadata = PaperMetadata(title=paper_title)

        # Extract year
        year_match = re.search(r'\b(19|20)\d{2}\b', latex_content)
        if year_match:
            metadata.year = int(year_match.group())

        # Extract authors from \author{} command
        author_match = re.search(r'\\author\{([^}]+)\}', latex_content, re.IGNORECASE)
        if author_match:
            authors_text = author_match.group(1)
            # Simple split (may not be perfect)
            metadata.authors = [a.strip() for a in re.split(r'[,&]| and ', authors_text) if a.strip()]

        # Extract abstract
        abstract_match = re.search(r'\\begin\{abstract\}(.*?)\\end\{abstract\}', latex_content, re.DOTALL | re.IGNORECASE)
        if abstract_match:
            metadata.abstract = abstract_match.group(1).strip()[:500]

        return metadata

    async def _llm_extraction(self, latex_content: str, paper_title: str) -> PaperMetadata:
        """LLM-based extraction for complete metadata"""

        # Truncate LaTeX content to first 10,000 characters (context window limit)
        truncated_content = latex_content[:10000]

        prompt = f"""Extract metadata from this LaTeX research paper. Return ONLY valid JSON.

LATEX CONTENT:
{truncated_content}

Extract the following and return as JSON:
{{
  "title": "{paper_title}",
  "authors": ["author1", "author2", ...],
  "affiliations": ["affiliation1", "affiliation2", ...],
  "year": 2024,
  "abstract": "paper abstract...",
  "venue": "conference or journal name",
  "venue_short": "short name (e.g., NeurIPS, CVPR)",
  "methods": ["method1", "method2", ...],
  "datasets": ["dataset1", "dataset2", ...],
  "metrics": ["metric1", "metric2", ...],
  "research_field": "AI/ML/CV/NLP/etc"
}}

IMPORTANT:
- authors: Extract ALL author names
- affiliations: Match affiliations to authors (same order)
- methods: Key algorithms, architectures, techniques used
- datasets: Benchmark datasets mentioned
- metrics: Evaluation metrics (accuracy, BLEU, F1, etc.)
- If a field is not found, use empty string/array

Return ONLY the JSON object, no markdown formatting."""

        try:
            response = self.model.generate_content(prompt)
            response_text = response.text.strip()

            # Clean markdown formatting
            response_text = response_text.replace("```json", "").replace("```", "").strip()

            # Parse JSON
            data = json.loads(response_text)

            metadata = PaperMetadata(
                title=paper_title,
                authors=data.get("authors", []),
                affiliations=data.get("affiliations", []),
                year=data.get("year", 0),
                abstract=data.get("abstract", ""),
                venue=data.get("venue", ""),
                venue_short=data.get("venue_short", ""),
                methods=data.get("methods", []),
                datasets=data.get("datasets", []),
                metrics=data.get("metrics", []),
                research_field=data.get("research_field", "")
            )

            return metadata

        except Exception as e:
            logger.error(f"  ❌ LLM extraction failed: {e}")
            return PaperMetadata(title=paper_title)

    def _merge_metadata(self, simple: PaperMetadata, llm: PaperMetadata) -> PaperMetadata:
        """Merge results preferring LLM data"""
        return PaperMetadata(
            title=llm.title or simple.title,
            authors=llm.authors if llm.authors else simple.authors,
            affiliations=llm.affiliations if llm.affiliations else simple.affiliations,
            year=llm.year if llm.year else simple.year,
            abstract=llm.abstract if llm.abstract else simple.abstract,
            venue=llm.venue if llm.venue else simple.venue,
            venue_short=llm.venue_short if llm.venue_short else simple.venue_short,
            methods=llm.methods if llm.methods else simple.methods,
            datasets=llm.datasets if llm.datasets else simple.datasets,
            metrics=llm.metrics if llm.metrics else simple.metrics,
            research_field=llm.research_field if llm.research_field else simple.research_field
        )

    async def extract_citations(self, latex_content: str, paper_title: str) -> List[Dict[str, Any]]:
        """Extract citations from LaTeX"""

        # Try regex first for \cite{} commands
        cite_pattern = re.findall(r'\\cite\{([^}]+)\}', latex_content)

        if not cite_pattern:
            return []

        # Use LLM to extract full citation details
        prompt = f"""Extract citation details from this LaTeX paper.

LATEX CONTENT:
{latex_content[:10000]}

Found citation keys: {cite_pattern}

For each citation, extract:
- title: cited paper title
- authors: author names
- year: publication year
- importance: "high", "medium", or "low" based on how frequently/prominently cited

Return as JSON array:
[
  {{"title": "Paper Title", "authors": ["Author1"], "year": 2020, "importance": "high", "context": "brief context"}},
  ...
]

Return ONLY the JSON array."""

        try:
            response = self.model.generate_content(prompt)
            response_text = response.text.strip().replace("```json", "").replace("```", "").strip()

            citations = json.loads(response_text)

            logger.info(f"  ✅ Extracted {len(citations)} citations")

            return citations

        except Exception as e:
            logger.error(f"  ❌ Citation extraction failed: {e}")
            return []
