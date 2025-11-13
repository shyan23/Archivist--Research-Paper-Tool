"""
FastAPI application for the Archivist Search Engine microservice.
"""

import os
import asyncio
from typing import List, Dict, Any
from fastapi import FastAPI, HTTPException, File, UploadFile, Body
from fastapi.responses import FileResponse
from fastapi.middleware.cors import CORSMiddleware

from . import __version__
from .models import (
    SearchQuery,
    SearchResponse,
    SearchResult,
    DownloadRequest,
    DownloadResponse,
    HealthResponse
)
from .providers import ArxivProvider, OpenReviewProvider, ACLProvider
from .hybrid_search import get_search_orchestrator
from .vector_store import get_vector_store
from .cache import get_cache

# Create FastAPI app
app = FastAPI(
    title="Archivist Search Engine",
    description="Modular search engine for academic papers from arXiv, OpenReview, and ACL Anthology",
    version=__version__,
    docs_url="/docs",
    redoc_url="/redoc"
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Allow all origins for now
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Initialize providers
providers = {
    "arxiv": ArxivProvider(),
    "openreview": OpenReviewProvider(),
    "acl": ACLProvider()
}


@app.get("/", response_model=HealthResponse)
async def root():
    """Root endpoint - health check."""
    return {
        "status": "running",
        "version": __version__,
        "providers": list(providers.keys())
    }


@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint."""
    return {
        "status": "healthy",
        "version": __version__,
        "providers": list(providers.keys())
    }


@app.post("/api/search", response_model=SearchResponse)
async def search_papers(query: SearchQuery):
    """
    Search for academic papers on arXiv with intelligent fuzzy matching.

    Args:
        query: SearchQuery object with search parameters

    Returns:
        SearchResponse with results ranked by relevance

    Features:
        - arXiv API search
        - Automatic abbreviation expansion (CNN, BERT, etc.)
        - Fuzzy string matching for typo tolerance
        - Relevance-based ranking
        - Redis caching to reduce API costs
    """
    try:
        # Get cache instance
        cache = get_cache()

        # Check if results are cached
        sources = query.sources if query.sources else ["arxiv"]
        cached_results = await cache.get_cached_results(
            query=query.query,
            sources=sources,
            max_results=query.max_results
        )

        if cached_results:
            print(f"Cache hit for query: {query.query}")
            return SearchResponse(
                query=query.query,
                total=len(cached_results),
                results=cached_results,
                sources_searched=["arXiv"],
                cached=True
            )

        # Cache miss - perform actual search
        print(f"Cache miss for query: {query.query}")

        # Get hybrid search orchestrator
        orchestrator = get_search_orchestrator()

        # Perform search
        results = await orchestrator.search(query)

        # Cache the results for future requests
        await cache.cache_results(
            query=query.query,
            sources=sources,
            max_results=query.max_results,
            results=results
        )

        return SearchResponse(
            query=query.query,
            total=len(results),
            results=results,
            sources_searched=["arXiv"],
            cached=False
        )

    except Exception as e:
        print(f"Search error: {e}")
        raise HTTPException(
            status_code=500,
            detail=f"Search failed: {str(e)}"
        )


@app.post("/api/download", response_model=DownloadResponse)
async def download_paper(request: DownloadRequest):
    """
    Download a paper PDF from a URL.

    Args:
        request: DownloadRequest with PDF URL and optional filename

    Returns:
        DownloadResponse with download status
    """
    try:
        # Determine provider based on URL
        provider = None
        if "arxiv.org" in request.pdf_url:
            provider = providers["arxiv"]
        elif "openreview.net" in request.pdf_url:
            provider = providers["openreview"]
        elif "aclanthology.org" in request.pdf_url:
            provider = providers["acl"]
        else:
            raise HTTPException(
                status_code=400,
                detail="Unknown PDF source. Supported: arXiv, OpenReview, ACL"
            )

        # Generate filename
        if request.filename:
            filename = request.filename
            if not filename.endswith(".pdf"):
                filename += ".pdf"
        else:
            # Extract filename from URL
            filename = request.pdf_url.split("/")[-1]
            if not filename.endswith(".pdf"):
                filename += ".pdf"

        # Sanitize filename
        filename = provider._sanitize_filename(filename.replace(".pdf", "")) + ".pdf"

        # Create temp directory if it doesn't exist
        temp_dir = "/tmp/archivist_downloads"
        os.makedirs(temp_dir, exist_ok=True)

        output_path = os.path.join(temp_dir, filename)

        # Download the PDF
        success = await provider.download_pdf(request.pdf_url, output_path)

        if not success:
            raise HTTPException(status_code=500, detail="Failed to download PDF")

        # Get file size
        file_size = os.path.getsize(output_path)

        return DownloadResponse(
            success=True,
            filename=filename,
            size_bytes=file_size,
            message=f"PDF downloaded successfully to {output_path}"
        )

    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Download error: {str(e)}")


@app.get("/api/providers")
async def list_providers():
    """List all available search providers."""
    return {
        "providers": [
            {
                "name": provider.name(),
                "key": key
            }
            for key, provider in providers.items()
        ]
    }


@app.get("/api/stats")
async def get_stats():
    """Get service statistics."""
    return {
        "total_providers": len(providers),
        "providers": list(providers.keys()),
        "version": __version__,
        "status": "operational"
    }


@app.post("/api/index/paper")
async def index_paper(paper: Dict[str, Any] = Body(...)):
    """
    Index a single paper in the vector store for semantic search.

    Args:
        paper: Paper dictionary with fields:
            - paper_id: Unique identifier
            - title: Paper title
            - abstract: Paper abstract
            - authors: List of authors
            - metadata: Additional metadata (source, venue, published_at, etc.)

    Returns:
        Status of indexing operation
    """
    try:
        vector_store = get_vector_store()

        success = await vector_store.index_paper(
            paper_id=paper.get("paper_id", paper.get("id", "")),
            title=paper.get("title", ""),
            abstract=paper.get("abstract", ""),
            authors=paper.get("authors", []),
            metadata=paper.get("metadata", {})
        )

        if success:
            return {
                "success": True,
                "message": f"Paper indexed successfully: {paper.get('title', 'Unknown')}"
            }
        else:
            raise HTTPException(
                status_code=500,
                detail="Failed to index paper"
            )

    except Exception as e:
        raise HTTPException(
            status_code=500,
            detail=f"Indexing error: {str(e)}"
        )


@app.post("/api/index/batch")
async def index_papers_batch(papers: List[Dict[str, Any]] = Body(...)):
    """
    Index multiple papers in batch for efficient processing.

    Args:
        papers: List of paper dictionaries

    Returns:
        Number of successfully indexed papers
    """
    try:
        vector_store = get_vector_store()

        indexed_count = await vector_store.index_papers_batch(papers)

        return {
            "success": True,
            "indexed_count": indexed_count,
            "total_submitted": len(papers),
            "message": f"Successfully indexed {indexed_count}/{len(papers)} papers"
        }

    except Exception as e:
        raise HTTPException(
            status_code=500,
            detail=f"Batch indexing error: {str(e)}"
        )


@app.post("/api/index/from-search")
async def index_from_search_results(query: SearchQuery):
    """
    Search papers and automatically index them in the vector store.

    Args:
        query: Search query (uses keyword search only)

    Returns:
        Number of papers indexed
    """
    try:
        # Force keyword search mode
        query.search_mode = "keyword"

        # Get search orchestrator
        orchestrator = get_search_orchestrator()

        # Perform keyword search
        results = await orchestrator.search(query)

        # Prepare papers for indexing
        papers = []
        for result in results:
            paper = {
                "paper_id": result.id,
                "title": result.title,
                "abstract": result.abstract,
                "authors": result.authors,
                "metadata": {
                    "source": result.source,
                    "venue": result.venue,
                    "published_at": result.published_at.isoformat() if result.published_at else None,
                    "pdf_url": result.pdf_url,
                    "source_url": result.source_url,
                    "categories": result.categories
                }
            }
            papers.append(paper)

        # Index papers
        vector_store = get_vector_store()
        indexed_count = await vector_store.index_papers_batch(papers)

        return {
            "success": True,
            "query": query.query,
            "found": len(results),
            "indexed": indexed_count,
            "message": f"Indexed {indexed_count} papers from search results"
        }

    except Exception as e:
        raise HTTPException(
            status_code=500,
            detail=f"Index from search error: {str(e)}"
        )


@app.get("/api/vector-store/info")
async def get_vector_store_info():
    """Get information about the vector store."""
    try:
        vector_store = get_vector_store()
        info = vector_store.get_collection_info()
        return info
    except Exception as e:
        raise HTTPException(
            status_code=500,
            detail=f"Error getting vector store info: {str(e)}"
        )


@app.delete("/api/vector-store/clear")
async def clear_vector_store():
    """Clear all vectors from the vector store."""
    try:
        vector_store = get_vector_store()
        success = await vector_store.clear_collection()

        if success:
            return {
                "success": True,
                "message": "Vector store cleared successfully"
            }
        else:
            raise HTTPException(
                status_code=500,
                detail="Failed to clear vector store"
            )

    except Exception as e:
        raise HTTPException(
            status_code=500,
            detail=f"Error clearing vector store: {str(e)}"
        )


@app.get("/api/cache/stats")
async def get_cache_stats():
    """Get cache statistics including memory usage and cached queries count."""
    try:
        cache = get_cache()
        stats = await cache.get_cache_stats()
        return stats
    except Exception as e:
        raise HTTPException(
            status_code=500,
            detail=f"Error getting cache stats: {str(e)}"
        )


@app.delete("/api/cache/clear")
async def clear_cache():
    """Clear all cached search results."""
    try:
        cache = get_cache()
        success = await cache.clear_all_cache()

        if success:
            return {
                "success": True,
                "message": "Cache cleared successfully"
            }
        else:
            raise HTTPException(
                status_code=500,
                detail="Failed to clear cache"
            )

    except Exception as e:
        raise HTTPException(
            status_code=500,
            detail=f"Error clearing cache: {str(e)}"
        )


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
