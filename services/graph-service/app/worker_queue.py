"""
Worker Queue - Manages concurrent background processing of graph building jobs
"""

import asyncio
import uuid
import logging
from typing import Optional, Dict, Any
from datetime import datetime
from dataclasses import dataclass, asdict
from enum import Enum

logger = logging.getLogger(__name__)


class JobStatus(str, Enum):
    PENDING = "pending"
    PROCESSING = "processing"
    COMPLETED = "completed"
    FAILED = "failed"


@dataclass
class GraphJob:
    """Represents a graph building job"""
    job_id: str
    paper_title: str
    latex_content: str
    pdf_path: str
    processed_at: Optional[str]
    priority: int
    status: JobStatus
    created_at: str
    started_at: Optional[str] = None
    completed_at: Optional[str] = None
    error: Optional[str] = None
    progress: float = 0.0


class WorkerQueue:
    """Manages background workers for graph building"""

    def __init__(self, graph_builder, metadata_extractor, num_workers: int = 4):
        self.graph_builder = graph_builder
        self.metadata_extractor = metadata_extractor
        self.num_workers = num_workers

        # Job queue (priority queue)
        self.job_queue = asyncio.PriorityQueue()

        # Job tracking
        self.jobs: Dict[str, GraphJob] = {}
        self.processed_count = 0
        self.failed_count = 0

        # Worker control
        self.workers = []
        self.is_running = False
        self._shutdown_event = asyncio.Event()

    async def start(self):
        """Start worker tasks"""
        self.is_running = True

        for i in range(self.num_workers):
            worker = asyncio.create_task(self._worker(i))
            self.workers.append(worker)

        logger.info(f"✅ Started {self.num_workers} graph workers")

    async def _worker(self, worker_id: int):
        """Worker coroutine that processes jobs"""
        logger.info(f"[Worker {worker_id}] Started")

        while self.is_running:
            try:
                # Get job from queue with timeout
                try:
                    priority, job = await asyncio.wait_for(
                        self.job_queue.get(),
                        timeout=1.0
                    )
                except asyncio.TimeoutError:
                    continue

                logger.info(f"[Worker {worker_id}] Processing: {job.paper_title}")

                # Update job status
                job.status = JobStatus.PROCESSING
                job.started_at = datetime.now().isoformat()

                # Process the job
                try:
                    await self._process_job(job, worker_id)

                    job.status = JobStatus.COMPLETED
                    job.progress = 100.0
                    job.completed_at = datetime.now().isoformat()

                    self.processed_count += 1
                    logger.info(f"[Worker {worker_id}] ✅ Completed: {job.paper_title}")

                except Exception as e:
                    job.status = JobStatus.FAILED
                    job.error = str(e)
                    job.completed_at = datetime.now().isoformat()

                    self.failed_count += 1
                    logger.error(f"[Worker {worker_id}] ❌ Failed: {job.paper_title} - {e}")

                finally:
                    self.job_queue.task_done()

            except Exception as e:
                logger.error(f"[Worker {worker_id}] Unexpected error: {e}")

        logger.info(f"[Worker {worker_id}] Stopped")

    async def _process_job(self, job: GraphJob, worker_id: int):
        """Process a single graph building job"""

        # Step 1: Extract metadata (20% progress)
        logger.info(f"[Worker {worker_id}]   📊 Extracting metadata...")
        metadata = await self.metadata_extractor.extract_metadata(
            job.latex_content,
            job.paper_title
        )
        job.progress = 20.0

        # Step 2: Add paper node (40% progress)
        logger.info(f"[Worker {worker_id}]   📝 Adding paper node...")
        await self.graph_builder.add_paper_node(
            title=job.paper_title,
            pdf_path=job.pdf_path,
            metadata=metadata
        )
        job.progress = 40.0

        # Step 3: Add authors and relationships (60% progress)
        logger.info(f"[Worker {worker_id}]   👤 Adding {len(metadata.authors)} authors...")
        await self.graph_builder.add_authors(
            paper_title=job.paper_title,
            authors=metadata.authors,
            affiliations=metadata.affiliations
        )
        job.progress = 60.0

        # Step 4: Add methods, datasets, venues (80% progress)
        logger.info(f"[Worker {worker_id}]   🔧 Adding methods and datasets...")
        await self.graph_builder.add_methods(job.paper_title, metadata.methods)
        await self.graph_builder.add_datasets(job.paper_title, metadata.datasets)

        if metadata.venue:
            await self.graph_builder.add_venue(job.paper_title, metadata.venue, metadata.year)

        job.progress = 80.0

        # Step 5: Extract and add citations (100% progress)
        logger.info(f"[Worker {worker_id}]   🔗 Extracting citations...")
        citations = await self.metadata_extractor.extract_citations(
            job.latex_content,
            job.paper_title
        )

        if citations:
            await self.graph_builder.add_citations(job.paper_title, citations)

        job.progress = 100.0

        logger.info(f"[Worker {worker_id}]   ✅ Graph building complete for: {job.paper_title}")

    async def submit_job(
        self,
        paper_title: str,
        latex_content: str,
        pdf_path: str,
        processed_at: Optional[str] = None,
        priority: int = 0
    ) -> str:
        """Submit a job to the queue (higher priority = processed first)"""

        job_id = str(uuid.uuid4())

        job = GraphJob(
            job_id=job_id,
            paper_title=paper_title,
            latex_content=latex_content,
            pdf_path=pdf_path,
            processed_at=processed_at or datetime.now().isoformat(),
            priority=priority,
            status=JobStatus.PENDING,
            created_at=datetime.now().isoformat()
        )

        # Store job
        self.jobs[job_id] = job

        # Add to priority queue (lower number = higher priority)
        await self.job_queue.put((-priority, job))

        logger.info(f"📥 Job queued: {job_id} - {paper_title} (priority: {priority})")

        return job_id

    async def get_job_status(self, job_id: str) -> Optional[Dict[str, Any]]:
        """Get status of a job"""
        job = self.jobs.get(job_id)

        if not job:
            return None

        return asdict(job)

    def queue_size(self) -> int:
        """Get current queue size"""
        return self.job_queue.qsize()

    async def shutdown(self):
        """Gracefully shutdown workers"""
        logger.info("🛑 Shutting down worker queue...")

        self.is_running = False

        # Wait for all workers to finish
        await asyncio.gather(*self.workers, return_exceptions=True)

        # Wait for queue to be empty
        await self.job_queue.join()

        logger.info(f"✅ Worker queue shut down. Processed: {self.processed_count}, Failed: {self.failed_count}")
