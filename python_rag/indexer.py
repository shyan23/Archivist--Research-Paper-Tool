"""
Document indexing system for RAG
Handles indexing of research papers into vector store
"""
import logging
import hashlib
from pathlib import Path
from typing import List, Optional, Dict, Any
import json

try:
    from .chunker import TextChunker, Chunk, create_chunks_from_latex
    from .embeddings import EmbeddingProvider
    from .vector_store import VectorStore, VectorDocument
    from .pdf_utils import extract_text_from_pdf, extract_title_from_pdf
except ImportError:
    from chunker import TextChunker, Chunk, create_chunks_from_latex
    from embeddings import EmbeddingProvider
    from vector_store import VectorStore, VectorDocument
    from pdf_utils import extract_text_from_pdf, extract_title_from_pdf

logger = logging.getLogger(__name__)


class DocumentIndexer:
    """Indexes documents into vector store with embeddings"""

    def __init__(
        self,
        chunker: TextChunker,
        embedding_provider: EmbeddingProvider,
        vector_store: VectorStore,
        metadata_dir: str = ".metadata"
    ):
        """
        Initialize document indexer

        Args:
            chunker: Text chunker
            embedding_provider: Embedding provider
            vector_store: Vector store
            metadata_dir: Directory to store indexing metadata
        """
        self.chunker = chunker
        self.embedding_provider = embedding_provider
        self.vector_store = vector_store
        self.metadata_dir = Path(metadata_dir)
        self.metadata_dir.mkdir(parents=True, exist_ok=True)

        self.index_metadata_file = self.metadata_dir / "indexed_papers.json"
        self.indexed_papers = self._load_index_metadata()

    def index_paper(
        self,
        paper_title: str,
        latex_content: str,
        pdf_path: Optional[str] = None,
        force_reindex: bool = False
    ) -> int:
        """
        Index a research paper

        Args:
            paper_title: Title of the paper (used as source identifier)
            latex_content: LaTeX content of the paper
            pdf_path: Optional path to PDF file
            force_reindex: Whether to force reindexing even if already indexed

        Returns:
            Number of chunks indexed
        """
        logger.info(f"📇 Indexing paper: {paper_title}")

        # Check if already indexed
        if not force_reindex and self.is_indexed(paper_title):
            num_chunks = self.indexed_papers[paper_title]['num_chunks']
            logger.info(f"  ⏭️  Paper already indexed ({num_chunks} chunks), skipping")
            return num_chunks

        # If reindexing, delete old chunks
        if force_reindex and self.is_indexed(paper_title):
            logger.info("  🗑️  Removing old index...")
            self.vector_store.delete_by_source(paper_title)

        # Step 1: Chunk the LaTeX content
        logger.info("  ✂️  Chunking LaTeX content...")
        chunks = self.chunker.chunk_latex(latex_content, paper_title)

        if not chunks:
            logger.warning("  ⚠️  No chunks generated!")
            return 0

        logger.info(f"  ✓ Created {len(chunks)} chunks")

        # Step 2: Generate embeddings
        logger.info(f"  🧮 Generating embeddings...")
        chunk_texts = [chunk.text for chunk in chunks]

        embeddings = self.embedding_provider.embed_batch(
            chunk_texts,
            batch_size=32,
            show_progress=False
        )

        logger.info(f"  ✓ Generated {len(embeddings)} embeddings")

        # Step 3: Create vector documents
        vector_docs = []
        for i, (chunk, embedding) in enumerate(zip(chunks, embeddings)):
            doc_id = self._generate_doc_id(paper_title, i)

            metadata = {
                'source': paper_title,
                'section': chunk.section,
                'chunk_index': str(i),
                **chunk.metadata
            }

            if pdf_path:
                metadata['pdf_path'] = pdf_path

            vector_doc = VectorDocument(
                id=doc_id,
                text=chunk.text,
                embedding=embedding,
                metadata=metadata
            )
            vector_docs.append(vector_doc)

        # Step 4: Add to vector store
        logger.info("  💾 Storing vectors in database...")
        self.vector_store.add_documents(vector_docs)

        # Step 5: Update metadata
        self.indexed_papers[paper_title] = {
            'num_chunks': len(chunks),
            'indexed_at': self._current_timestamp(),
            'pdf_path': pdf_path or '',
            'embedding_dim': self.embedding_provider.dimension
        }
        self._save_index_metadata()

        logger.info(f"  ✅ Successfully indexed: {paper_title} ({len(chunks)} chunks)")

        return len(chunks)

    def index_paper_from_latex_file(
        self,
        latex_path: str,
        paper_title: Optional[str] = None,
        pdf_path: Optional[str] = None,
        force_reindex: bool = False
    ) -> int:
        """
        Index a paper from a LaTeX file

        Args:
            latex_path: Path to .tex file
            paper_title: Optional paper title (extracted from filename if not provided)
            pdf_path: Optional path to corresponding PDF
            force_reindex: Whether to force reindexing

        Returns:
            Number of chunks indexed
        """
        latex_path = Path(latex_path)

        if not latex_path.exists():
            raise FileNotFoundError(f"LaTeX file not found: {latex_path}")

        # Read LaTeX content
        with open(latex_path, 'r', encoding='utf-8') as f:
            latex_content = f.read()

        # Extract title if not provided
        if not paper_title:
            paper_title = latex_path.stem

        return self.index_paper(paper_title, latex_content, pdf_path, force_reindex)

    def index_paper_from_pdf(
        self,
        pdf_path: str,
        paper_title: Optional[str] = None,
        force_reindex: bool = False
    ) -> int:
        """
        Index a paper from a PDF file

        Args:
            pdf_path: Path to .pdf file
            paper_title: Optional paper title (extracted from filename/metadata if not provided)
            force_reindex: Whether to force reindexing

        Returns:
            Number of chunks indexed
        """
        pdf_path = Path(pdf_path)

        if not pdf_path.exists():
            raise FileNotFoundError(f"PDF file not found: {pdf_path}")

        # Extract text from PDF
        text_content = extract_text_from_pdf(str(pdf_path))

        if not text_content:
            logger.warning(f"No text extracted from PDF: {pdf_path}")
            return 0

        # Extract title if not provided
        if not paper_title:
            # Try to get from PDF metadata first
            paper_title = extract_title_from_pdf(str(pdf_path))
            if not paper_title:
                # Fall back to filename
                paper_title = pdf_path.stem

        return self.index_paper(paper_title, text_content, str(pdf_path), force_reindex)

    def index_directory(
        self,
        directory: str,
        pattern: str = "*.tex",
        force_reindex: bool = False
    ) -> Dict[str, int]:
        """
        Index all LaTeX or PDF files in a directory

        Args:
            directory: Directory containing .tex or .pdf files
            pattern: File pattern to match (e.g., "*.tex", "*.pdf")
            force_reindex: Whether to force reindexing

        Returns:
            Dictionary mapping paper titles to number of chunks indexed
        """
        directory = Path(directory)

        if not directory.exists():
            raise FileNotFoundError(f"Directory not found: {directory}")

        files = list(directory.glob(pattern))

        if not files:
            logger.warning(f"No files matching '{pattern}' found in {directory}")
            return {}

        logger.info(f"📚 Indexing {len(files)} papers from {directory}")

        results = {}

        for file_path in files:
            try:
                # Handle PDF files
                if file_path.suffix.lower() == '.pdf':
                    num_chunks = self.index_paper_from_pdf(
                        str(file_path),
                        force_reindex=force_reindex
                    )
                    # Use extracted title or filename
                    title = extract_title_from_pdf(str(file_path)) or file_path.stem
                    results[title] = num_chunks
                # Handle LaTeX files
                elif file_path.suffix.lower() == '.tex':
                    num_chunks = self.index_paper_from_latex_file(
                        str(file_path),
                        force_reindex=force_reindex
                    )
                    results[file_path.stem] = num_chunks
                else:
                    logger.warning(f"  ⚠️ Unsupported file type: {file_path.name}")
                    continue

            except Exception as e:
                logger.error(f"  ❌ Failed to index {file_path.name}: {e}")
                results[file_path.stem] = 0

        logger.info(f"✅ Batch indexing complete: {len(results)} papers processed")
        return results

    def is_indexed(self, paper_title: str) -> bool:
        """Check if a paper is already indexed"""
        return paper_title in self.indexed_papers

    def get_indexed_papers(self) -> List[str]:
        """Get list of all indexed papers"""
        return list(self.indexed_papers.keys())

    def get_index_info(self, paper_title: str) -> Optional[Dict[str, Any]]:
        """Get indexing info for a paper"""
        return self.indexed_papers.get(paper_title)

    def delete_paper_index(self, paper_title: str) -> int:
        """
        Delete index for a specific paper

        Args:
            paper_title: Paper to delete

        Returns:
            Number of chunks deleted
        """
        if not self.is_indexed(paper_title):
            logger.warning(f"Paper not indexed: {paper_title}")
            return 0

        # Delete from vector store
        deleted = self.vector_store.delete_by_source(paper_title)

        # Remove from metadata
        del self.indexed_papers[paper_title]
        self._save_index_metadata()

        logger.info(f"✓ Deleted index for: {paper_title} ({deleted} chunks)")
        return deleted

    def _generate_doc_id(self, paper_title: str, chunk_index: int) -> str:
        """Generate unique document ID"""
        hash_input = f"{paper_title}_{chunk_index}".encode('utf-8')
        hash_hex = hashlib.md5(hash_input).hexdigest()
        return f"{hash_hex}_chunk_{chunk_index}"

    def _load_index_metadata(self) -> Dict[str, Any]:
        """Load index metadata from disk"""
        if not self.index_metadata_file.exists():
            return {}

        try:
            with open(self.index_metadata_file, 'r') as f:
                return json.load(f)
        except Exception as e:
            logger.warning(f"Failed to load index metadata: {e}")
            return {}

    def _save_index_metadata(self):
        """Save index metadata to disk"""
        try:
            with open(self.index_metadata_file, 'w') as f:
                json.dump(self.indexed_papers, f, indent=2)
        except Exception as e:
            logger.error(f"Failed to save index metadata: {e}")

    @staticmethod
    def _current_timestamp() -> str:
        """Get current timestamp as ISO string"""
        from datetime import datetime
        return datetime.now().isoformat()
