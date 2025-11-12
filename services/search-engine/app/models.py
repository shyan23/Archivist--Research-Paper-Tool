"""
Data models for the search engine microservice.
"""

from datetime import datetime
from typing import List, Optional, Literal
from pydantic import BaseModel, Field


class SearchMode(str):
    """Search mode enumeration."""
    KEYWORD = "keyword"
    SEMANTIC = "semantic"
    FUZZY = "fuzzy"
    HYBRID = "hybrid"


class SearchQuery(BaseModel):
    """Search query request model."""
    query: str = Field(..., description="Search query string", min_length=1)
    max_results: int = Field(default=20, description="Maximum number of results", ge=1, le=100)
    sources: Optional[List[str]] = Field(default=None, description="Filter by sources (currently only arXiv)")
    start_date: Optional[datetime] = Field(default=None, description="Filter papers published after this date")
    end_date: Optional[datetime] = Field(default=None, description="Filter papers published before this date")
    fuzzy_threshold: int = Field(default=70, description="Minimum fuzzy match score (0-100)", ge=0, le=100)
    # Legacy fields (kept for backwards compatibility)
    search_mode: str = Field(default="fuzzy", description="Search mode (deprecated, always uses fuzzy)")
    semantic_weight: float = Field(default=0.7, description="Deprecated", ge=0, le=1)


class SearchResult(BaseModel):
    """Single paper search result."""
    title: str = Field(..., description="Paper title")
    authors: List[str] = Field(default_factory=list, description="List of authors")
    abstract: str = Field(default="", description="Paper abstract")
    published_at: datetime = Field(..., description="Publication date")
    pdf_url: Optional[str] = Field(default="", description="Direct PDF download URL")
    source_url: Optional[str] = Field(default="", description="Source page URL")
    source: str = Field(..., description="Source name: arXiv, OpenReview, ACL, Semantic")
    venue: str = Field(default="", description="Conference or journal name")
    id: str = Field(..., description="Unique ID from source")
    categories: List[str] = Field(default_factory=list, description="Paper categories/keywords")
    relevance_score: Optional[float] = Field(default=None, description="Relevance score (0-1)")
    similarity_score: Optional[float] = Field(default=None, description="Semantic similarity score (0-1)")
    fuzzy_score: Optional[float] = Field(default=None, description="Fuzzy match score (0-100)")


class SearchResponse(BaseModel):
    """Search response with results."""
    query: str = Field(..., description="Original search query")
    total: int = Field(..., description="Total number of results")
    results: List[SearchResult] = Field(..., description="List of search results")
    sources_searched: List[str] = Field(..., description="List of sources that were searched")


class DownloadRequest(BaseModel):
    """PDF download request."""
    pdf_url: str = Field(..., description="PDF URL to download")
    filename: Optional[str] = Field(default=None, description="Optional custom filename")


class DownloadResponse(BaseModel):
    """PDF download response."""
    success: bool = Field(..., description="Whether download was successful")
    filename: str = Field(..., description="Downloaded filename")
    size_bytes: int = Field(..., description="File size in bytes")
    message: str = Field(default="", description="Success or error message")


class HealthResponse(BaseModel):
    """Health check response."""
    status: str = Field(..., description="Service status")
    version: str = Field(..., description="Service version")
    providers: List[str] = Field(..., description="Available search providers")
