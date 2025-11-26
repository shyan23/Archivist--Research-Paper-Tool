"""
Citation Impact Analysis - H-index, temporal trends, influential citations
"""

import logging
from typing import List, Dict, Any, Optional, Tuple
from datetime import datetime
from collections import defaultdict
import math

logger = logging.getLogger(__name__)


class CitationImpactAnalyzer:
    """Analyzes citation impact and metrics for papers"""

    def __init__(self, graph_builder):
        self.graph_builder = graph_builder
        logger.info("✅ Citation impact analyzer initialized")

    async def calculate_h_index(self, author_name: str) -> Dict[str, Any]:
        """
        Calculate H-index for an author

        H-index: An author has index h if h of their papers have at least h citations each,
        and the other papers have no more than h citations each.

        Returns:
            {
                "h_index": int,
                "total_papers": int,
                "total_citations": int,
                "highly_cited_papers": List[Dict]
            }
        """
        try:
            # Get author's papers with citation counts
            query = """
            MATCH (a:Author {name: $author_name})<-[:WRITTEN_BY]-(p:Paper)
            OPTIONAL MATCH (p)<-[r:CITES]-()
            RETURN p.title as title,
                   p.year as year,
                   count(r) as citations
            ORDER BY citations DESC
            """

            async with self.graph_builder.driver.session(database=self.graph_builder.database) as session:
                result = await session.run(query, {"author_name": author_name})
                records = await result.values()

            if not records:
                return {
                    "h_index": 0,
                    "total_papers": 0,
                    "total_citations": 0,
                    "highly_cited_papers": []
                }

            # Calculate H-index
            citation_counts = [record[2] for record in records]
            h_index = self._compute_h_index(citation_counts)

            # Get highly cited papers (those contributing to H-index)
            highly_cited = []
            for i, record in enumerate(records):
                if i < h_index:
                    highly_cited.append({
                        "title": record[0],
                        "year": record[1],
                        "citations": record[2]
                    })

            total_citations = sum(citation_counts)

            logger.info(f"  ✅ H-index for '{author_name}': {h_index}")

            return {
                "h_index": h_index,
                "total_papers": len(records),
                "total_citations": total_citations,
                "highly_cited_papers": highly_cited
            }

        except Exception as e:
            logger.error(f"  ❌ H-index calculation failed: {e}")
            return {
                "h_index": 0,
                "total_papers": 0,
                "total_citations": 0,
                "highly_cited_papers": []
            }

    def _compute_h_index(self, citation_counts: List[int]) -> int:
        """
        Compute H-index from sorted citation counts

        Args:
            citation_counts: List of citation counts (sorted descending)

        Returns:
            H-index value
        """
        h = 0
        for i, citations in enumerate(citation_counts, 1):
            if citations >= i:
                h = i
            else:
                break
        return h

    async def get_citation_timeline(self, paper_title: str) -> Dict[str, Any]:
        """
        Get citation count over time for a paper

        Returns:
            {
                "paper_title": str,
                "total_citations": int,
                "timeline": [{"year": 2020, "citations": 5}, ...],
                "growth_rate": float
            }
        """
        try:
            # Get citations with years
            query = """
            MATCH (source:Paper)-[r:CITES]->(target:Paper {title: $paper_title})
            RETURN source.year as citing_year,
                   count(r) as citation_count
            ORDER BY citing_year
            """

            async with self.graph_builder.driver.session(database=self.graph_builder.database) as session:
                result = await session.run(query, {"paper_title": paper_title})
                records = await result.values()

            # Build timeline
            timeline = []
            year_counts = defaultdict(int)

            for record in records:
                year = record[0]
                count = record[1]
                if year:
                    year_counts[year] += count

            # Sort by year
            for year in sorted(year_counts.keys()):
                timeline.append({
                    "year": year,
                    "citations": year_counts[year]
                })

            total_citations = sum(year_counts.values())

            # Calculate growth rate (citations per year)
            growth_rate = 0.0
            if len(timeline) > 1:
                years_span = timeline[-1]["year"] - timeline[0]["year"]
                if years_span > 0:
                    growth_rate = total_citations / years_span

            logger.info(f"  ✅ Citation timeline for '{paper_title}': {total_citations} total")

            return {
                "paper_title": paper_title,
                "total_citations": total_citations,
                "timeline": timeline,
                "growth_rate": round(growth_rate, 2)
            }

        except Exception as e:
            logger.error(f"  ❌ Citation timeline failed: {e}")
            return {
                "paper_title": paper_title,
                "total_citations": 0,
                "timeline": [],
                "growth_rate": 0.0
            }

    async def get_influential_citations(
        self,
        paper_title: str,
        top_k: int = 10
    ) -> List[Dict[str, Any]]:
        """
        Get most influential citations (papers that cite this one)

        Influence is measured by:
        1. Citation count of the citing paper
        2. Importance field in CITES relationship
        3. Recency of citation

        Returns:
            List of influential citing papers with scores
        """
        try:
            query = """
            MATCH (citing:Paper)-[r:CITES]->(target:Paper {title: $paper_title})
            OPTIONAL MATCH (citing)<-[c:CITES]-()
            WITH citing, r, count(c) as citing_paper_citations
            RETURN citing.title as title,
                   citing.year as year,
                   citing.authors as authors,
                   r.importance as importance,
                   r.context as context,
                   citing_paper_citations
            ORDER BY citing_paper_citations DESC
            LIMIT $top_k
            """

            async with self.graph_builder.driver.session(database=self.graph_builder.database) as session:
                result = await session.run(query, {
                    "paper_title": paper_title,
                    "top_k": top_k
                })
                records = await result.values()

            influential_citations = []

            for record in records:
                # Calculate influence score
                impact_score = self._calculate_influence_score(
                    citing_citations=record[5],
                    importance=record[3],
                    year=record[1]
                )

                influential_citations.append({
                    "title": record[0],
                    "year": record[1],
                    "authors": record[2] if record[2] else [],
                    "importance": record[3],
                    "context": record[4],
                    "citations": record[5],
                    "influence_score": round(impact_score, 2)
                })

            logger.info(f"  ✅ Found {len(influential_citations)} influential citations")

            return influential_citations

        except Exception as e:
            logger.error(f"  ❌ Influential citations failed: {e}")
            return []

    def _calculate_influence_score(
        self,
        citing_citations: int,
        importance: Optional[str],
        year: Optional[int]
    ) -> float:
        """
        Calculate influence score for a citing paper

        Factors:
        - Citation count of citing paper (higher = more influence)
        - Importance level (high/medium/low)
        - Recency (more recent = slightly more weight)

        Returns:
            Influence score (0-100)
        """
        # Base score from citation count (log scale to avoid extreme values)
        citation_score = math.log(citing_citations + 1) * 10

        # Importance multiplier
        importance_multiplier = {
            "high": 1.5,
            "medium": 1.0,
            "low": 0.7
        }.get(importance if importance else "medium", 1.0)

        # Recency bonus (papers from last 5 years get small boost)
        current_year = datetime.now().year
        recency_bonus = 0
        if year and (current_year - year) <= 5:
            recency_bonus = 5

        influence_score = (citation_score * importance_multiplier) + recency_bonus

        return min(influence_score, 100)  # Cap at 100

    async def extract_citation_contexts(
        self,
        paper_title: str
    ) -> Dict[str, List[str]]:
        """
        Extract why papers cite this one

        Groups citation contexts by theme

        Returns:
            {
                "methodology": ["context1", ...],
                "comparison": ["context2", ...],
                "background": ["context3", ...],
                "extension": ["context4", ...]
            }
        """
        try:
            query = """
            MATCH (citing:Paper)-[r:CITES]->(target:Paper {title: $paper_title})
            WHERE r.context IS NOT NULL AND r.context <> ''
            RETURN citing.title as citing_title,
                   r.context as context,
                   r.importance as importance
            """

            async with self.graph_builder.driver.session(database=self.graph_builder.database) as session:
                result = await session.run(query, {"paper_title": paper_title})
                records = await result.values()

            # Group contexts by theme (simple keyword-based classification)
            contexts = {
                "methodology": [],
                "comparison": [],
                "background": [],
                "extension": [],
                "other": []
            }

            for record in records:
                context = record[1].lower()

                # Classify context
                if any(word in context for word in ["method", "approach", "technique", "algorithm"]):
                    contexts["methodology"].append(record[1])
                elif any(word in context for word in ["compare", "versus", "outperform", "better"]):
                    contexts["comparison"].append(record[1])
                elif any(word in context for word in ["background", "prior", "previous", "seminal"]):
                    contexts["background"].append(record[1])
                elif any(word in context for word in ["extend", "improve", "build", "based on"]):
                    contexts["extension"].append(record[1])
                else:
                    contexts["other"].append(record[1])

            # Count total contexts
            total = sum(len(v) for v in contexts.values())

            logger.info(f"  ✅ Extracted {total} citation contexts for '{paper_title}'")

            return contexts

        except Exception as e:
            logger.error(f"  ❌ Citation context extraction failed: {e}")
            return {
                "methodology": [],
                "comparison": [],
                "background": [],
                "extension": [],
                "other": []
            }

    async def get_complete_citation_analysis(
        self,
        paper_title: str
    ) -> Dict[str, Any]:
        """
        Get comprehensive citation analysis for a paper

        Combines all metrics into one report

        Returns:
            Complete citation analysis with all metrics
        """
        try:
            # Run all analyses in parallel
            timeline = await self.get_citation_timeline(paper_title)
            influential = await self.get_influential_citations(paper_title, top_k=5)
            contexts = await self.extract_citation_contexts(paper_title)

            # Get basic paper info
            paper_info = await self.graph_builder.get_paper_details(paper_title)

            analysis = {
                "paper_title": paper_title,
                "paper_info": paper_info,
                "citation_metrics": {
                    "total_citations": timeline["total_citations"],
                    "growth_rate": timeline["growth_rate"],
                    "timeline": timeline["timeline"]
                },
                "influential_citations": influential,
                "citation_contexts": contexts
            }

            logger.info(f"  ✅ Complete citation analysis for '{paper_title}'")

            return analysis

        except Exception as e:
            logger.error(f"  ❌ Complete citation analysis failed: {e}")
            return {
                "paper_title": paper_title,
                "error": str(e)
            }
