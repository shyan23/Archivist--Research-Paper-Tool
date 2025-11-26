"""
Fuzzy string matching utilities for improved search tolerance.
Handles typos, abbreviations, and partial matches.
"""

from typing import List, Tuple, Dict, Any
from rapidfuzz import fuzz, process
import re


class FuzzyMatcher:
    """Fuzzy string matching for research paper search."""

    def __init__(
        self,
        match_threshold: int = 70,
        partial_threshold: int = 80
    ):
        """
        Initialize fuzzy matcher.

        Args:
            match_threshold: Minimum score for full string matching (0-100)
            partial_threshold: Minimum score for partial matching (0-100)
        """
        self.match_threshold = match_threshold
        self.partial_threshold = partial_threshold

    def fuzzy_match(
        self,
        query: str,
        text: str,
        method: str = "token_set_ratio"
    ) -> int:
        """
        Calculate fuzzy match score between query and text.

        Args:
            query: Search query
            text: Text to match against
            method: Matching method - 'token_set_ratio', 'partial_ratio', 'ratio'

        Returns:
            Match score (0-100)
        """
        if not query or not text:
            return 0

        query = query.lower().strip()
        text = text.lower().strip()

        if method == "token_set_ratio":
            return fuzz.token_set_ratio(query, text)
        elif method == "partial_ratio":
            return fuzz.partial_ratio(query, text)
        elif method == "ratio":
            return fuzz.ratio(query, text)
        else:
            return fuzz.token_set_ratio(query, text)

    def fuzzy_search_papers(
        self,
        query: str,
        papers: List[Dict[str, Any]],
        search_fields: List[str] = ["title", "abstract"],
        limit: int = 20
    ) -> List[Tuple[Dict[str, Any], int]]:
        """
        Fuzzy search through a list of papers.

        Args:
            query: Search query
            papers: List of paper dictionaries
            search_fields: Fields to search in
            limit: Maximum results

        Returns:
            List of (paper, score) tuples sorted by score
        """
        results = []

        for paper in papers:
            # Calculate max score across all search fields
            max_score = 0

            for field in search_fields:
                if field in paper:
                    text = str(paper[field])
                    score = self.fuzzy_match(query, text)
                    max_score = max(max_score, score)

            # Include if above threshold
            if max_score >= self.match_threshold:
                results.append((paper, max_score))

        # Sort by score descending
        results.sort(key=lambda x: x[1], reverse=True)

        return results[:limit]

    def extract_best_matches(
        self,
        query: str,
        choices: List[str],
        limit: int = 10,
        score_cutoff: int = None
    ) -> List[Tuple[str, int]]:
        """
        Extract best matching strings from a list of choices.

        Args:
            query: Search query
            choices: List of strings to search
            limit: Maximum results
            score_cutoff: Minimum score to include

        Returns:
            List of (choice, score) tuples
        """
        if score_cutoff is None:
            score_cutoff = self.match_threshold

        results = process.extract(
            query,
            choices,
            scorer=fuzz.token_set_ratio,
            limit=limit,
            score_cutoff=score_cutoff
        )

        return [(choice, score) for choice, score, _ in results]

    def calculate_paper_relevance(
        self,
        query: str,
        paper: Dict[str, Any],
        weights: Dict[str, float] = None
    ) -> float:
        """
        Calculate overall relevance score for a paper using weighted fuzzy matching.

        Args:
            query: Search query
            paper: Paper dictionary
            weights: Field weights (default: title=0.5, abstract=0.3, authors=0.2)

        Returns:
            Weighted relevance score (0-100)
        """
        if weights is None:
            weights = {
                "title": 0.5,
                "abstract": 0.3,
                "authors": 0.2
            }

        total_score = 0.0
        total_weight = 0.0

        # Title matching
        if "title" in paper and "title" in weights:
            title_score = self.fuzzy_match(query, paper["title"])
            total_score += title_score * weights["title"]
            total_weight += weights["title"]

        # Abstract matching
        if "abstract" in paper and "abstract" in weights:
            abstract_score = self.fuzzy_match(query, paper["abstract"])
            total_score += abstract_score * weights["abstract"]
            total_weight += weights["abstract"]

        # Authors matching (check each author)
        if "authors" in paper and "authors" in weights:
            authors = paper["authors"]
            if isinstance(authors, list):
                authors_text = " ".join(authors)
                authors_score = self.fuzzy_match(query, authors_text)
                total_score += authors_score * weights["authors"]
                total_weight += weights["authors"]

        # Normalize score
        if total_weight > 0:
            return total_score / total_weight
        return 0.0

    def expand_query_terms(self, query: str) -> List[str]:
        """
        Expand query terms to handle common abbreviations and variations.

        Args:
            query: Search query

        Returns:
            List of expanded query terms
        """
        # Common ML/AI abbreviations
        expansions = {
            "nn": ["neural network", "neural networks"],
            "cnn": ["convolutional neural network", "cnn"],
            "rnn": ["recurrent neural network", "rnn"],
            "lstm": ["long short-term memory", "lstm"],
            "gru": ["gated recurrent unit", "gru"],
            "gan": ["generative adversarial network", "gan"],
            "vae": ["variational autoencoder", "vae"],
            "bert": ["bidirectional encoder representations from transformers", "bert"],
            "gpt": ["generative pre-trained transformer", "gpt"],
            "nlp": ["natural language processing", "nlp"],
            "cv": ["computer vision", "cv"],
            "ml": ["machine learning", "ml"],
            "ai": ["artificial intelligence", "ai"],
            "rl": ["reinforcement learning", "rl"],
            "dl": ["deep learning", "dl"],
        }

        query_lower = query.lower()
        expanded_terms = [query]

        # Check for abbreviations
        for abbrev, expansions_list in expansions.items():
            if re.search(r'\b' + abbrev + r'\b', query_lower):
                # Add expanded versions
                for expansion in expansions_list:
                    expanded_query = re.sub(
                        r'\b' + abbrev + r'\b',
                        expansion,
                        query_lower
                    )
                    if expanded_query not in expanded_terms:
                        expanded_terms.append(expanded_query)

        return expanded_terms

    def is_typo_match(
        self,
        word1: str,
        word2: str,
        max_distance: int = 2
    ) -> bool:
        """
        Check if two words are likely typos of each other.

        Args:
            word1: First word
            word2: Second word
            max_distance: Maximum edit distance to consider as typo

        Returns:
            True if likely a typo match
        """
        if not word1 or not word2:
            return False

        # Use Levenshtein distance
        score = fuzz.ratio(word1.lower(), word2.lower())

        # High ratio indicates likely typo
        return score >= 85


# Global fuzzy matcher instance
_fuzzy_matcher: FuzzyMatcher = None


def get_fuzzy_matcher() -> FuzzyMatcher:
    """Get or create global fuzzy matcher instance."""
    global _fuzzy_matcher
    if _fuzzy_matcher is None:
        _fuzzy_matcher = FuzzyMatcher()
    return _fuzzy_matcher
