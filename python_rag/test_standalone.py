#!/usr/bin/env python3
"""
Standalone test script for Python RAG system with PDF support
Tests the chatbot with papers from /lib directory
"""
import os
import sys
from pathlib import Path
from dotenv import load_dotenv

load_dotenv()



# Set API key (modify this with your actual key)
if not os.getenv("GEMINI_API_KEY"):
    print("WARNING: GEMINI_API_KEY not set!")
    print("Set it with: export GEMINI_API_KEY=your_api_key")
    print("Or modify this script to set it directly\n")
    # Uncomment and add your key here:
    # os.environ["GEMINI_API_KEY"] = "your_key_here"

# Path to papers library
LIB_PATH = Path("/home/shyan/Desktop/Code/Archivist/lib")

# Example papers from your lib directory
EXAMPLE_PAPERS = [
    "NIPS-2017-attention-is-all-you-need-Paper.pdf",
    "2209.03561v2.pdf",  # Some research paper
    "Focus.pdf",
]


def index_sample_papers():
    """Index a few sample papers for testing"""
    from config import RAGConfig
    from embeddings import create_embedding_provider
    from vector_store import create_vector_store
    from chunker import TextChunker
    from indexer import DocumentIndexer

    print("\n" + "=" * 70)
    print("INDEXING SAMPLE PAPERS")
    print("=" * 70 + "\n")

    # Initialize config
    config = RAGConfig.from_env()

    # Initialize components
    print("Initializing RAG components...")
    embedding_provider = create_embedding_provider(
        provider=config.embedding.provider,
        model_name=config.embedding.model_name,
        api_key=config.embedding.gemini_api_key
    )

    vector_store = create_vector_store(
        provider=config.vector_store.provider,
        persist_directory=config.vector_store.persist_directory,
        collection_name=config.vector_store.collection_name,
        dimension=embedding_provider.dimension
    )

    chunker = TextChunker(
        chunk_size=config.chunking.chunk_size,
        chunk_overlap=config.chunking.chunk_overlap
    )

    indexer = DocumentIndexer(chunker, embedding_provider, vector_store)

    # Index sample papers
    print(f"\nIndexing papers from: {LIB_PATH}\n")

    for paper_file in EXAMPLE_PAPERS:
        pdf_path = LIB_PATH / paper_file

        if not pdf_path.exists():
            print(f"⚠️  Skipping {paper_file} (not found)")
            continue

        try:
            num_chunks = indexer.index_paper_from_pdf(str(pdf_path))
            print(f"✅ Indexed: {paper_file} ({num_chunks} chunks)")
        except Exception as e:
            print(f"❌ Failed to index {paper_file}: {e}")

    # Show summary
    print("\n" + "=" * 70)
    print(f"INDEXED PAPERS: {len(indexer.get_indexed_papers())}")
    print("=" * 70)
    for paper in indexer.get_indexed_papers():
        info = indexer.get_index_info(paper)
        print(f"  - {paper}: {info['num_chunks']} chunks")
    print()


def test_chat_cli():
    """Test chat via CLI"""
    from cli import RAGChatCLI
    from config import RAGConfig

    print("\n" + "=" * 70)
    print("STARTING INTERACTIVE CHAT")
    print("=" * 70 + "\n")

    config = RAGConfig.from_env()
    cli = RAGChatCLI(config)

    # Start interactive chat
    cli.chat_interactive()


def test_retrieval():
    """Test retrieval without chat"""
    from config import RAGConfig
    from embeddings import create_embedding_provider
    from vector_store import create_vector_store
    from retriever import Retriever

    print("\n" + "=" * 70)
    print("TESTING RETRIEVAL")
    print("=" * 70 + "\n")

    config = RAGConfig.from_env()

    embedding_provider = create_embedding_provider(
        provider=config.embedding.provider,
        model_name=config.embedding.model_name,
        api_key=config.embedding.gemini_api_key
    )

    vector_store = create_vector_store(
        provider=config.vector_store.provider,
        persist_directory=config.vector_store.persist_directory,
        collection_name=config.vector_store.collection_name,
        dimension=embedding_provider.dimension
    )

    retriever = Retriever(vector_store, embedding_provider)

    # Test query
    query = "What is the attention mechanism?"
    print(f"Query: {query}\n")

    context = retriever.retrieve(query, top_k=3)

    print(f"Retrieved {context.total_chunks} chunks:\n")
    print(f"Sources: {', '.join(context.sources)}")
    print(f"\nContext preview (first 500 chars):\n{context.context_text[:500]}...\n")


def print_usage():
    """Print usage instructions"""
    print("\n" + "=" * 70)
    print("ARCHIVIST RAG - STANDALONE TEST")
    print("=" * 70)
    print("\nUsage:")
    print("  python test_standalone.py index    - Index sample papers")
    print("  python test_standalone.py retrieve - Test retrieval only")
    print("  python test_standalone.py chat     - Start interactive chat")
    print("  python test_standalone.py all      - Run all tests")
    print("\nFor FastAPI testing:")
    print("  1. Start server: python -m python_rag.cli server")
    print("  2. Open browser: http://localhost:8000/docs")
    print("  3. Use POST /index/pdf to index papers from /lib")
    print("  4. Use POST /chat/session to create a chat session")
    print("  5. Use POST /chat/message to chat with the papers")
    print("\nExample papers in /lib:")
    for paper in EXAMPLE_PAPERS:
        print(f"  - {paper}")
    print()


def main():
    """Main entry point"""
    if len(sys.argv) < 2:
        print_usage()
        return

    command = sys.argv[1].lower()

    if command == "index":
        index_sample_papers()

    elif command == "retrieve":
        test_retrieval()

    elif command == "chat":
        test_chat_cli()

    elif command == "all":
        index_sample_papers()
        test_retrieval()
        print("\nSkipping interactive chat in 'all' mode.")
        print("Run 'python test_standalone.py chat' for interactive chat.\n")

    else:
        print(f"Unknown command: {command}")
        print_usage()


if __name__ == "__main__":
    main()
