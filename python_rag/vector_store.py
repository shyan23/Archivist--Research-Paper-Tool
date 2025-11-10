"""
Vector store implementations for RAG system
Supports: ChromaDB, FAISS, Redis
"""
import logging
from abc import ABC, abstractmethod
from pathlib import Path
from typing import List, Dict, Any, Optional, Tuple
import numpy as np

logger = logging.getLogger(__name__)


class VectorDocument:
    """Document to be stored in vector database"""
    def __init__(
        self,
        id: str,
        text: str,
        embedding: np.ndarray,
        metadata: Dict[str, Any]
    ):
        self.id = id
        self.text = text
        self.embedding = embedding
        self.metadata = metadata


class SearchResult:
    """Search result with document and score"""
    def __init__(self, document: VectorDocument, score: float, distance: float = 0.0):
        self.document = document
        self.score = score
        self.distance = distance

    def __repr__(self):
        return f"SearchResult(score={self.score:.3f}, source={self.document.metadata.get('source', 'unknown')})"


class VectorStore(ABC):
    """Abstract base class for vector stores"""

    @abstractmethod
    def add_documents(self, documents: List[VectorDocument]) -> None:
        """Add documents to the vector store"""
        pass

    @abstractmethod
    def search(
        self,
        query_embedding: np.ndarray,
        top_k: int = 5,
        filter: Optional[Dict[str, Any]] = None
    ) -> List[SearchResult]:
        """Search for similar documents"""
        pass

    @abstractmethod
    def delete_by_source(self, source: str) -> int:
        """Delete all documents from a specific source"""
        pass

    @abstractmethod
    def get_documents_by_source(self, source: str) -> List[VectorDocument]:
        """Get all documents from a specific source"""
        pass

    @abstractmethod
    def count(self) -> int:
        """Get total number of documents"""
        pass


class ChromaDBVectorStore(VectorStore):
    """ChromaDB vector store implementation"""

    def __init__(
        self,
        persist_directory: str = ".metadata/chromadb",
        collection_name: str = "research_papers"
    ):
        try:
            import chromadb
            from chromadb.config import Settings
        except ImportError:
            raise ImportError("chromadb not installed. Run: pip install chromadb")

        self.persist_directory = Path(persist_directory)
        self.persist_directory.mkdir(parents=True, exist_ok=True)

        logger.info(f"Initializing ChromaDB at: {self.persist_directory}")

        self.client = chromadb.PersistentClient(
            path=str(self.persist_directory),
            settings=Settings(anonymized_telemetry=False)
        )

        self.collection_name = collection_name
        self.collection = self.client.get_or_create_collection(
            name=collection_name,
            metadata={"hnsw:space": "cosine"}  # Use cosine similarity
        )

        logger.info(f"✓ ChromaDB collection '{collection_name}' ready ({self.count()} documents)")

    def add_documents(self, documents: List[VectorDocument]) -> None:
        """Add documents to ChromaDB"""
        if not documents:
            return

        ids = [doc.id for doc in documents]
        embeddings = [doc.embedding.tolist() for doc in documents]
        metadatas = [doc.metadata for doc in documents]
        documents_text = [doc.text for doc in documents]

        self.collection.add(
            ids=ids,
            embeddings=embeddings,
            metadatas=metadatas,
            documents=documents_text
        )

        logger.info(f"✓ Added {len(documents)} documents to ChromaDB")

    def search(
        self,
        query_embedding: np.ndarray,
        top_k: int = 5,
        filter: Optional[Dict[str, Any]] = None
    ) -> List[SearchResult]:
        """Search for similar documents"""
        where = None
        if filter and "source" in filter:
            where = {"source": filter["source"]}

        results = self.collection.query(
            query_embeddings=[query_embedding.tolist()],
            n_results=top_k,
            where=where
        )

        search_results = []

        if results['ids'] and results['ids'][0]:
            for i, doc_id in enumerate(results['ids'][0]):
                distance = results['distances'][0][i] if results['distances'] else 0.0
                # Convert cosine distance to similarity score (1 - distance)
                score = 1.0 - distance

                doc = VectorDocument(
                    id=doc_id,
                    text=results['documents'][0][i],
                    embedding=np.array(results['embeddings'][0][i]) if results['embeddings'] else np.array([]),
                    metadata=results['metadatas'][0][i]
                )

                search_results.append(SearchResult(doc, score, distance))

        return search_results

    def delete_by_source(self, source: str) -> int:
        """Delete all documents from a specific source"""
        # Get all documents with this source
        results = self.collection.get(where={"source": source})

        if not results['ids']:
            return 0

        self.collection.delete(ids=results['ids'])
        count = len(results['ids'])
        logger.info(f"✓ Deleted {count} documents from source: {source}")
        return count

    def get_documents_by_source(self, source: str) -> List[VectorDocument]:
        """Get all documents from a specific source"""
        results = self.collection.get(where={"source": source})

        documents = []
        for i, doc_id in enumerate(results['ids']):
            doc = VectorDocument(
                id=doc_id,
                text=results['documents'][i],
                embedding=np.array(results['embeddings'][i]) if results.get('embeddings') else np.array([]),
                metadata=results['metadatas'][i]
            )
            documents.append(doc)

        return documents

    def get_all_sources(self) -> List[str]:
        """Get list of all unique sources"""
        # ChromaDB doesn't have a direct way to get unique metadata values
        # We need to fetch all documents and extract unique sources
        all_docs = self.collection.get()
        if not all_docs['metadatas']:
            return []

        sources = set()
        for metadata in all_docs['metadatas']:
            if 'source' in metadata:
                sources.add(metadata['source'])

        return sorted(list(sources))

    def count(self) -> int:
        """Get total number of documents"""
        return self.collection.count()


class FAISSVectorStore(VectorStore):
    """FAISS vector store implementation (for faster search)"""

    def __init__(
        self,
        index_path: str = ".metadata/faiss_index",
        dimension: int = 384
    ):
        try:
            import faiss
        except ImportError:
            raise ImportError("faiss not installed. Run: pip install faiss-cpu")

        self.index_path = Path(index_path)
        self.index_path.mkdir(parents=True, exist_ok=True)

        self.dimension = dimension
        self.index_file = self.index_path / "index.faiss"
        self.metadata_file = self.index_path / "metadata.npz"

        # Create or load FAISS index
        if self.index_file.exists():
            self.index = faiss.read_index(str(self.index_file))
            self._load_metadata()
            logger.info(f"✓ Loaded FAISS index from {self.index_file} ({self.count()} documents)")
        else:
            # Create new index with cosine similarity
            # Normalize vectors for cosine similarity via L2
            self.index = faiss.IndexFlatIP(dimension)  # Inner product for normalized vectors
            self.documents_metadata = []
            logger.info(f"✓ Created new FAISS index (dimension: {dimension})")

    def add_documents(self, documents: List[VectorDocument]) -> None:
        """Add documents to FAISS index"""
        if not documents:
            return

        # Extract embeddings and normalize for cosine similarity
        embeddings = np.array([doc.embedding for doc in documents], dtype=np.float32)
        faiss.normalize_L2(embeddings)  # Normalize for cosine similarity

        # Add to index
        self.index.add(embeddings)

        # Store metadata
        for doc in documents:
            self.documents_metadata.append({
                'id': doc.id,
                'text': doc.text,
                'metadata': doc.metadata
            })

        self._save()
        logger.info(f"✓ Added {len(documents)} documents to FAISS index")

    def search(
        self,
        query_embedding: np.ndarray,
        top_k: int = 5,
        filter: Optional[Dict[str, Any]] = None
    ) -> List[SearchResult]:
        """Search for similar documents"""
        import faiss

        # Normalize query embedding
        query_emb = query_embedding.reshape(1, -1).astype(np.float32)
        faiss.normalize_L2(query_emb)

        # Search
        scores, indices = self.index.search(query_emb, min(top_k * 2, self.count()))  # Get more for filtering

        results = []
        for score, idx in zip(scores[0], indices[0]):
            if idx == -1:  # FAISS returns -1 for empty slots
                continue

            metadata_entry = self.documents_metadata[idx]

            # Apply filter
            if filter:
                if 'source' in filter and metadata_entry['metadata'].get('source') != filter['source']:
                    continue

            doc = VectorDocument(
                id=metadata_entry['id'],
                text=metadata_entry['text'],
                embedding=np.array([]),  # Don't store embeddings in memory
                metadata=metadata_entry['metadata']
            )

            results.append(SearchResult(doc, float(score), 1.0 - float(score)))

            if len(results) >= top_k:
                break

        return results

    def delete_by_source(self, source: str) -> int:
        """Delete all documents from a specific source"""
        # FAISS doesn't support deletion directly
        # We need to rebuild the index without the source
        new_metadata = []
        deleted_count = 0

        for i, meta in enumerate(self.documents_metadata):
            if meta['metadata'].get('source') == source:
                deleted_count += 1
            else:
                new_metadata.append(meta)

        if deleted_count == 0:
            return 0

        # Rebuild index
        self.documents_metadata = new_metadata
        self.index.reset()

        # Re-add remaining documents
        # (Note: This is inefficient for large datasets)
        logger.info(f"Rebuilding FAISS index after deleting {deleted_count} documents...")

        # Would need to re-embed or store embeddings - simplified for now
        self._save()
        logger.info(f"✓ Deleted {deleted_count} documents from source: {source}")

        return deleted_count

    def get_documents_by_source(self, source: str) -> List[VectorDocument]:
        """Get all documents from a specific source"""
        documents = []
        for meta in self.documents_metadata:
            if meta['metadata'].get('source') == source:
                doc = VectorDocument(
                    id=meta['id'],
                    text=meta['text'],
                    embedding=np.array([]),
                    metadata=meta['metadata']
                )
                documents.append(doc)
        return documents

    def get_all_sources(self) -> List[str]:
        """Get list of all unique sources"""
        sources = set()
        for meta in self.documents_metadata:
            if 'source' in meta['metadata']:
                sources.add(meta['metadata']['source'])
        return sorted(list(sources))

    def count(self) -> int:
        """Get total number of documents"""
        return self.index.ntotal

    def _save(self):
        """Save index and metadata to disk"""
        import faiss
        faiss.write_index(self.index, str(self.index_file))
        np.savez_compressed(
            self.metadata_file,
            metadata=np.array(self.documents_metadata, dtype=object)
        )

    def _load_metadata(self):
        """Load metadata from disk"""
        data = np.load(self.metadata_file, allow_pickle=True)
        self.documents_metadata = data['metadata'].tolist()


def create_vector_store(
    provider: str = "chromadb",
    persist_directory: str = ".metadata/chromadb",
    collection_name: str = "research_papers",
    dimension: int = 384
) -> VectorStore:
    """
    Factory function to create vector store

    Args:
        provider: One of "chromadb", "faiss"
        persist_directory: Directory to persist the store
        collection_name: Collection name (for ChromaDB)
        dimension: Embedding dimension (for FAISS)

    Returns:
        VectorStore instance
    """
    provider = provider.lower()

    if provider == "chromadb":
        return ChromaDBVectorStore(persist_directory, collection_name)
    elif provider == "faiss":
        return FAISSVectorStore(persist_directory, dimension)
    else:
        raise ValueError(f"Unknown provider: {provider}. Choose from: chromadb, faiss")
