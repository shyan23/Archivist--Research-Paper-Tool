"""
Graph Microservice - FastAPI server for concurrent Neo4j graph building
Handles paper metadata extraction and graph construction without blocking paper processing
"""

from fastapi import FastAPI, HTTPException, BackgroundTasks
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import List, Optional, Dict, Any
import asyncio
import logging
from datetime import datetime

from .graph_builder import GraphBuilder
from .metadata_extractor import MetadataExtractor
from .worker_queue import WorkerQueue
from .kafka_consumer import GraphKafkaConsumer
import os

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Initialize FastAPI app
app = FastAPI(
    title="Archivist Graph Service",
    description="Microservice for building knowledge graphs from research papers",
    version="1.0.0"
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Global instances (initialized on startup)
graph_builder: Optional[GraphBuilder] = None
metadata_extractor: Optional[MetadataExtractor] = None
worker_queue: Optional[WorkerQueue] = None
kafka_consumer: Optional[GraphKafkaConsumer] = None
semantic_search: Optional = None  # Import at runtime
citation_analyzer: Optional = None  # Import at runtime


# ============================================================================
# Request/Response Models
# ============================================================================

class PaperRequest(BaseModel):
    """Request to add a paper to the graph"""
    paper_title: str
    latex_content: str
    pdf_path: str
    processed_at: Optional[str] = None
    priority: int = 0

class PaperResponse(BaseModel):
    """Response after submitting paper"""
    status: str
    message: str
    job_id: str
    queue_position: int

class GraphStatsResponse(BaseModel):
    """Graph statistics"""
    paper_count: int
    author_count: int
    citation_count: int
    method_count: int
    dataset_count: int
    venue_count: int
    institution_count: int
    last_updated: str

class JobStatusResponse(BaseModel):
    """Job processing status"""
    job_id: str
    status: str  # pending, processing, completed, failed
    progress: float
    message: Optional[str] = None
    created_at: str
    completed_at: Optional[str] = None

class BatchProcessRequest(BaseModel):
    """Request to process multiple papers"""
    papers: List[PaperRequest]

class QueryRequest(BaseModel):
    """Graph query request"""
    query_type: str  # "citations", "similar", "author_papers", "method_papers"
    parameters: Dict[str, Any]


# ============================================================================
# Startup/Shutdown Events
# ============================================================================

@app.on_event("startup")
async def startup_event():
    """Initialize services on startup"""
    global graph_builder, metadata_extractor, worker_queue, kafka_consumer, semantic_search, citation_analyzer

    logger.info("🚀 Starting Graph Microservice...")

    try:
        # Get configuration from environment
        neo4j_uri = os.getenv("NEO4J_URI", "bolt://localhost:7687")
        neo4j_user = os.getenv("NEO4J_USER", "neo4j")
        neo4j_password = os.getenv("NEO4J_PASSWORD", "password")
        kafka_bootstrap = os.getenv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092")

        # Initialize graph builder
        logger.info(f"Connecting to Neo4j at {neo4j_uri}...")
        graph_builder = GraphBuilder(
            uri=neo4j_uri,
            username=neo4j_user,
            password=neo4j_password,
            database="neo4j"
        )
        await graph_builder.connect()
        await graph_builder.initialize_schema()
        logger.info("✅ Neo4j connected and schema initialized")

        # Initialize metadata extractor
        logger.info("Initializing metadata extractor...")
        metadata_extractor = MetadataExtractor()
        logger.info("✅ Metadata extractor ready")

        # Initialize worker queue
        logger.info("Starting worker queue (4 workers)...")
        worker_queue = WorkerQueue(
            graph_builder=graph_builder,
            metadata_extractor=metadata_extractor,
            num_workers=4
        )
        await worker_queue.start()
        logger.info("✅ Worker queue started")

        # Initialize Kafka consumer
        logger.info(f"Starting Kafka consumer ({kafka_bootstrap})...")
        kafka_consumer = GraphKafkaConsumer(
            bootstrap_servers=kafka_bootstrap,
            topic="paper.processed",
            group_id="graph-builder",
            worker_queue=worker_queue
        )
        await kafka_consumer.start()
        logger.info("✅ Kafka consumer started")

        # Initialize semantic search engine
        logger.info("Initializing semantic search engine...")
        from .semantic_search import SemanticSearchEngine
        qdrant_url = os.getenv("QDRANT_URL", "http://localhost:6333")
        semantic_search = SemanticSearchEngine(
            qdrant_url=qdrant_url,
            collection_name="papers"
        )
        logger.info("✅ Semantic search ready")

        # Initialize citation analyzer
        logger.info("Initializing citation impact analyzer...")
        from .citation_analysis import CitationImpactAnalyzer
        citation_analyzer = CitationImpactAnalyzer(graph_builder)
        logger.info("✅ Citation analyzer ready")

        logger.info("🎉 Graph Microservice ready!")

    except Exception as e:
        logger.error(f"❌ Startup failed: {e}")
        raise

@app.on_event("shutdown")
async def shutdown_event():
    """Cleanup on shutdown"""
    logger.info("🛑 Shutting down Graph Microservice...")

    if kafka_consumer:
        await kafka_consumer.stop()
        logger.info("✅ Kafka consumer stopped")

    if worker_queue:
        await worker_queue.shutdown()
        logger.info("✅ Worker queue shut down")

    if graph_builder:
        await graph_builder.close()
        logger.info("✅ Neo4j connection closed")

    logger.info("👋 Goodbye!")


# ============================================================================
# API Endpoints
# ============================================================================

@app.get("/")
async def root():
    """Health check endpoint"""
    return {
        "service": "Archivist Graph Service",
        "status": "running",
        "version": "1.0.0",
        "timestamp": datetime.now().isoformat()
    }

@app.get("/health")
async def health_check():
    """Detailed health check"""
    neo4j_status = "connected" if graph_builder and graph_builder.is_connected() else "disconnected"
    queue_status = "running" if worker_queue and worker_queue.is_running else "stopped"

    return {
        "status": "healthy" if neo4j_status == "connected" else "degraded",
        "neo4j": neo4j_status,
        "worker_queue": queue_status,
        "queue_size": worker_queue.queue_size() if worker_queue else 0,
        "timestamp": datetime.now().isoformat()
    }

@app.post("/api/graph/add-paper", response_model=PaperResponse)
async def add_paper(request: PaperRequest):
    """
    Add a paper to the knowledge graph (non-blocking)

    This endpoint queues the paper for background processing and returns immediately.
    The actual graph building happens asynchronously.
    """
    if not worker_queue:
        raise HTTPException(status_code=503, detail="Worker queue not initialized")

    try:
        # Submit job to worker queue
        job_id = await worker_queue.submit_job(
            paper_title=request.paper_title,
            latex_content=request.latex_content,
            pdf_path=request.pdf_path,
            processed_at=request.processed_at,
            priority=request.priority
        )

        queue_position = worker_queue.queue_size()

        logger.info(f"📥 Queued paper: {request.paper_title} (job_id: {job_id})")

        return PaperResponse(
            status="queued",
            message=f"Paper queued for graph building",
            job_id=job_id,
            queue_position=queue_position
        )

    except Exception as e:
        logger.error(f"❌ Failed to queue paper: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/api/graph/batch-process")
async def batch_process(request: BatchProcessRequest):
    """Process multiple papers in batch"""
    if not worker_queue:
        raise HTTPException(status_code=503, detail="Worker queue not initialized")

    job_ids = []

    for paper in request.papers:
        try:
            job_id = await worker_queue.submit_job(
                paper_title=paper.paper_title,
                latex_content=paper.latex_content,
                pdf_path=paper.pdf_path,
                processed_at=paper.processed_at,
                priority=paper.priority
            )
            job_ids.append(job_id)
        except Exception as e:
            logger.error(f"Failed to queue {paper.paper_title}: {e}")

    return {
        "status": "queued",
        "total_papers": len(request.papers),
        "queued": len(job_ids),
        "failed": len(request.papers) - len(job_ids),
        "job_ids": job_ids
    }

@app.get("/api/graph/job/{job_id}", response_model=JobStatusResponse)
async def get_job_status(job_id: str):
    """Get status of a specific job"""
    if not worker_queue:
        raise HTTPException(status_code=503, detail="Worker queue not initialized")

    status = await worker_queue.get_job_status(job_id)

    if not status:
        raise HTTPException(status_code=404, detail=f"Job {job_id} not found")

    return JobStatusResponse(**status)

@app.get("/api/graph/stats", response_model=GraphStatsResponse)
async def get_graph_stats():
    """Get knowledge graph statistics"""
    if not graph_builder:
        raise HTTPException(status_code=503, detail="Graph builder not initialized")

    try:
        stats = await graph_builder.get_stats()

        return GraphStatsResponse(
            paper_count=stats.get("paper_count", 0),
            author_count=stats.get("author_count", 0),
            citation_count=stats.get("citation_count", 0),
            method_count=stats.get("method_count", 0),
            dataset_count=stats.get("dataset_count", 0),
            venue_count=stats.get("venue_count", 0),
            institution_count=stats.get("institution_count", 0),
            last_updated=datetime.now().isoformat()
        )

    except Exception as e:
        logger.error(f"Failed to get stats: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/api/graph/queue-stats")
async def get_queue_stats():
    """Get worker queue statistics"""
    if not worker_queue:
        raise HTTPException(status_code=503, detail="Worker queue not initialized")

    return {
        "queue_size": worker_queue.queue_size(),
        "processed_count": worker_queue.processed_count,
        "failed_count": worker_queue.failed_count,
        "active_workers": worker_queue.num_workers,
        "is_running": worker_queue.is_running
    }

@app.post("/api/graph/query")
async def query_graph(request: QueryRequest):
    """Execute custom graph queries"""
    if not graph_builder:
        raise HTTPException(status_code=503, detail="Graph builder not initialized")

    try:
        result = await graph_builder.execute_query(
            query_type=request.query_type,
            parameters=request.parameters
        )

        return {
            "status": "success",
            "query_type": request.query_type,
            "results": result
        }

    except Exception as e:
        logger.error(f"Query failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/api/graph/paper/{paper_title}")
async def get_paper_details(paper_title: str):
    """Get detailed information about a paper from the graph"""
    if not graph_builder:
        raise HTTPException(status_code=503, detail="Graph builder not initialized")

    try:
        paper_info = await graph_builder.get_paper_details(paper_title)

        if not paper_info:
            raise HTTPException(status_code=404, detail=f"Paper '{paper_title}' not found in graph")

        return paper_info

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Failed to get paper details: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.delete("/api/graph/paper/{paper_title}")
async def delete_paper(paper_title: str):
    """Remove a paper from the graph"""
    if not graph_builder:
        raise HTTPException(status_code=503, detail="Graph builder not initialized")

    try:
        success = await graph_builder.delete_paper(paper_title)

        if success:
            return {"status": "success", "message": f"Paper '{paper_title}' deleted"}
        else:
            raise HTTPException(status_code=404, detail=f"Paper '{paper_title}' not found")

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Failed to delete paper: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/api/graph/rebuild")
async def rebuild_graph():
    """Rebuild entire graph (warning: expensive operation)"""
    if not graph_builder:
        raise HTTPException(status_code=503, detail="Graph builder not initialized")

    try:
        await graph_builder.clear_all()

        return {
            "status": "success",
            "message": "Graph cleared. Submit papers to rebuild."
        }

    except Exception as e:
        logger.error(f"Failed to rebuild graph: {e}")
        raise HTTPException(status_code=500, detail=str(e))


# ============================================================================
# NEW: Semantic Search & Recommendations
# ============================================================================

@app.post("/api/graph/search/semantic")
async def semantic_search_papers(query: str, top_k: int = 10, threshold: float = 0.7):
    """Semantic search for papers using natural language queries"""
    if not semantic_search:
        raise HTTPException(status_code=503, detail="Semantic search not initialized")

    try:
        results = await semantic_search.search_similar_papers(query, top_k, threshold)

        return {
            "status": "success",
            "query": query,
            "results": results,
            "count": len(results)
        }

    except Exception as e:
        logger.error(f"Semantic search failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/api/graph/recommend/{paper_title}")
async def recommend_similar_papers(paper_title: str, top_k: int = 10):
    """Get paper recommendations based on similarity"""
    if not semantic_search:
        raise HTTPException(status_code=503, detail="Semantic search not initialized")

    try:
        similar = await semantic_search.find_similar_to_paper(
            paper_title=paper_title,
            top_k=top_k,
            score_threshold=0.85
        )

        return {
            "status": "success",
            "source_paper": paper_title,
            "recommendations": similar,
            "count": len(similar)
        }

    except Exception as e:
        logger.error(f"Recommendation failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


# ============================================================================
# NEW: Citation Impact Analysis
# ============================================================================

@app.get("/api/graph/citations/h-index/{author_name}")
async def get_author_h_index(author_name: str):
    """Calculate H-index for an author"""
    if not citation_analyzer:
        raise HTTPException(status_code=503, detail="Citation analyzer not initialized")

    try:
        h_index_data = await citation_analyzer.calculate_h_index(author_name)

        return {
            "status": "success",
            "author": author_name,
            **h_index_data
        }

    except Exception as e:
        logger.error(f"H-index calculation failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/api/graph/citations/timeline/{paper_title}")
async def get_citation_timeline(paper_title: str):
    """Get citation count over time for a paper"""
    if not citation_analyzer:
        raise HTTPException(status_code=503, detail="Citation analyzer not initialized")

    try:
        timeline = await citation_analyzer.get_citation_timeline(paper_title)

        return {
            "status": "success",
            **timeline
        }

    except Exception as e:
        logger.error(f"Citation timeline failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/api/graph/citations/influential/{paper_title}")
async def get_influential_citations(paper_title: str, top_k: int = 10):
    """Get most influential citations for a paper"""
    if not citation_analyzer:
        raise HTTPException(status_code=503, detail="Citation analyzer not initialized")

    try:
        influential = await citation_analyzer.get_influential_citations(paper_title, top_k)

        return {
            "status": "success",
            "paper": paper_title,
            "influential_citations": influential,
            "count": len(influential)
        }

    except Exception as e:
        logger.error(f"Influential citations failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/api/graph/citations/contexts/{paper_title}")
async def get_citation_contexts(paper_title: str):
    """Get why papers cite this one (categorized by theme)"""
    if not citation_analyzer:
        raise HTTPException(status_code=503, detail="Citation analyzer not initialized")

    try:
        contexts = await citation_analyzer.extract_citation_contexts(paper_title)

        total_contexts = sum(len(v) for v in contexts.values())

        return {
            "status": "success",
            "paper": paper_title,
            "total_contexts": total_contexts,
            "contexts": contexts
        }

    except Exception as e:
        logger.error(f"Citation contexts failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/api/graph/citations/analysis/{paper_title}")
async def get_complete_citation_analysis(paper_title: str):
    """Get comprehensive citation analysis (all metrics combined)"""
    if not citation_analyzer:
        raise HTTPException(status_code=503, detail="Citation analyzer not initialized")

    try:
        analysis = await citation_analyzer.get_complete_citation_analysis(paper_title)

        return {
            "status": "success",
            **analysis
        }

    except Exception as e:
        logger.error(f"Complete citation analysis failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


# ============================================================================
# NEW: Advanced Graph Queries
# ============================================================================

@app.get("/api/graph/path")
async def find_connection_path(paper1: str, paper2: str, max_hops: int = 5):
    """Find shortest path between two papers in citation network"""
    if not graph_builder:
        raise HTTPException(status_code=503, detail="Graph builder not initialized")

    try:
        path = await graph_builder.find_path_between_papers(paper1, paper2, max_hops)

        if not path:
            return {
                "status": "not_found",
                "message": f"No path found between '{paper1}' and '{paper2}' within {max_hops} hops",
                "path": None
            }

        return {
            "status": "success",
            "paper1": paper1,
            "paper2": paper2,
            "path": path,
            "hops": len(path) - 1
        }

    except Exception as e:
        logger.error(f"Path finding failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/api/graph/trending/{year}")
async def get_trending_methods(year: int, top_k: int = 10):
    """Get trending methods/concepts for a given year"""
    if not graph_builder:
        raise HTTPException(status_code=503, detail="Graph builder not initialized")

    try:
        trending = await graph_builder.get_trending_concepts(year, top_k)

        return {
            "status": "success",
            "year": year,
            "trending_methods": trending,
            "count": len(trending)
        }

    except Exception as e:
        logger.error(f"Trending methods failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


# ============================================================================
# Main Entry Point
# ============================================================================

if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "app.main:app",
        host="0.0.0.0",
        port=8081,
        reload=True,
        log_level="info"
    )
