#!/usr/bin/env python3
"""
CLI for Python RAG system
Standalone interface for testing and using the RAG chatbot
"""
import os
import sys
import logging
import argparse
from pathlib import Path
from typing import Optional

from .config import RAGConfig
from .embeddings import create_embedding_provider
from .vector_store import create_vector_store
from .chunker import TextChunker
from .indexer import DocumentIndexer
from .retriever import Retriever
from .chat_engine import create_chat_engine

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(levelname)s: %(message)s'
)
logger = logging.getLogger(__name__)


class RAGChatCLI:
    """Command-line interface for RAG chat"""

    def __init__(self, config: RAGConfig):
        """Initialize RAG system"""
        self.config = config

        logger.info("🚀 Initializing Archivist RAG system...")

        # Initialize components
        logger.info(f"  📦 Loading embedding model: {config.embedding.model_name}")
        self.embedding_provider = create_embedding_provider(
            provider=config.embedding.provider,
            model_name=config.embedding.model_name,
            api_key=config.embedding.gemini_api_key or config.embedding.openai_api_key
        )

        logger.info(f"  💾 Loading vector store: {config.vector_store.provider}")
        self.vector_store = create_vector_store(
            provider=config.vector_store.provider,
            persist_directory=config.vector_store.persist_directory,
            collection_name=config.vector_store.collection_name,
            dimension=self.embedding_provider.dimension
        )

        # Create chunker
        self.chunker = TextChunker(
            chunk_size=config.chunking.chunk_size,
            chunk_overlap=config.chunking.chunk_overlap,
            respect_sections=config.chunking.respect_sections
        )

        # Create indexer
        self.indexer = DocumentIndexer(
            chunker=self.chunker,
            embedding_provider=self.embedding_provider,
            vector_store=self.vector_store
        )

        # Create retriever
        self.retriever = Retriever(
            vector_store=self.vector_store,
            embedding_provider=self.embedding_provider,
            top_k=config.retriever.top_k,
            score_threshold=config.retriever.score_threshold,
            max_context_length=config.retriever.max_context_length
        )

        # Create chat engine
        self.chat_engine = create_chat_engine(
            retriever=self.retriever,
            gemini_api_key=config.chat.gemini_api_key,
            model_name=config.chat.gemini_model,
            temperature=config.chat.temperature
        )

        logger.info("✅ RAG system ready!\n")

    def index_paper(self, file_path: str, force: bool = False):
        """Index a paper from LaTeX or PDF file"""
        file_path = Path(file_path)

        if not file_path.exists():
            logger.error(f"File not found: {file_path}")
            return

        # Handle PDF files
        if file_path.suffix.lower() == '.pdf':
            num_chunks = self.indexer.index_paper_from_pdf(
                str(file_path),
                force_reindex=force
            )
        # Handle LaTeX files
        elif file_path.suffix.lower() == '.tex':
            num_chunks = self.indexer.index_paper_from_latex_file(
                str(file_path),
                force_reindex=force
            )
        else:
            logger.error(f"Unsupported file type: {file_path.suffix}")
            logger.error("Supported types: .pdf, .tex")
            return

        print(f"\n✅ Indexed: {file_path.stem} ({num_chunks} chunks)\n")

    def index_directory(self, directory: str, pattern: str = "*.pdf", force: bool = False):
        """Index all PDF or LaTeX files in a directory"""
        results = self.indexer.index_directory(directory, pattern=pattern, force_reindex=force)

        print(f"\n✅ Indexed {len(results)} papers:\n")
        for paper, chunks in results.items():
            print(f"  - {paper}: {chunks} chunks")
        print()

    def list_papers(self):
        """List indexed papers"""
        papers = self.indexer.get_indexed_papers()

        if not papers:
            print("\n📭 No papers indexed yet.\n")
            return

        print(f"\n📚 Indexed Papers ({len(papers)}):\n")
        for i, paper in enumerate(papers, 1):
            info = self.indexer.get_index_info(paper)
            chunks = info.get('num_chunks', 0)
            print(f"  {i}. {paper} ({chunks} chunks)")
        print()

    def chat_interactive(self, paper_titles: Optional[list] = None):
        """Start interactive chat session"""
        paper_titles = paper_titles or []

        # Create session
        session = self.chat_engine.create_session(paper_titles)

        print("\n" + "=" * 70)
        print("💬 Archivist RAG Chat")
        print("=" * 70)

        if paper_titles:
            print(f"\n📖 Chatting about: {', '.join(paper_titles)}")
        else:
            print("\n📖 Chatting about all indexed papers")

        print("\nTips:")
        print("  - Ask specific questions about methodologies, results, etc.")
        print("  - Type 'exit' or 'quit' to end the session")
        print("  - Type 'papers' to see which papers are being discussed")
        print()

        while True:
            try:
                # Get user input
                user_input = input("\n💬 You: ").strip()

                if not user_input:
                    continue

                if user_input.lower() in ['exit', 'quit', 'q']:
                    print("\n👋 Goodbye!\n")
                    break

                if user_input.lower() == 'papers':
                    if paper_titles:
                        print(f"\n📖 Current papers: {', '.join(paper_titles)}\n")
                    else:
                        print("\n📖 Searching across all indexed papers\n")
                    continue

                # Get response
                response = self.chat_engine.chat(session.session_id, user_input)

                # Display response
                print(f"\n🤖 Archivist: {response.content}")

                if response.citations:
                    print(f"\n📚 Sources: {', '.join(response.citations)}")

            except KeyboardInterrupt:
                print("\n\n👋 Goodbye!\n")
                break
            except Exception as e:
                logger.error(f"Error: {e}")
                print(f"\n❌ Error: {e}\n")


def main():
    """Main CLI entry point"""
    parser = argparse.ArgumentParser(
        description="Archivist Python RAG System - Chat with research papers"
    )

    subparsers = parser.add_subparsers(dest='command', help='Commands')

    # Index command
    index_parser = subparsers.add_parser('index', help='Index papers')
    index_parser.add_argument('path', help='PDF/LaTeX file or directory to index')
    index_parser.add_argument('--force', action='store_true', help='Force reindexing')
    index_parser.add_argument('--pattern', default='*.pdf', help='File pattern for directory indexing (default: *.pdf)')

    # List command
    subparsers.add_parser('list', help='List indexed papers')

    # Chat command
    chat_parser = subparsers.add_parser('chat', help='Start chat session')
    chat_parser.add_argument(
        '--papers',
        nargs='+',
        help='Specific papers to chat about (optional)'
    )

    # Server command
    server_parser = subparsers.add_parser('server', help='Start API server')
    server_parser.add_argument('--port', type=int, default=8000, help='Server port')
    server_parser.add_argument('--host', default='0.0.0.0', help='Server host')

    args = parser.parse_args()

    # Load configuration
    config = RAGConfig.from_env()

    # Check for API key
    if not config.embedding.gemini_api_key and not config.embedding.openai_api_key:
        if not os.getenv('GEMINI_API_KEY'):
            logger.error("❌ GEMINI_API_KEY environment variable not set")
            logger.error("   Set it with: export GEMINI_API_KEY=your_api_key")
            sys.exit(1)
        config.embedding.gemini_api_key = os.getenv('GEMINI_API_KEY')
        config.chat.gemini_api_key = os.getenv('GEMINI_API_KEY')

    if not args.command:
        parser.print_help()
        sys.exit(0)

    # Execute command
    if args.command == 'server':
        # Start FastAPI server
        import uvicorn
        from .api_server import app

        logger.info(f"🚀 Starting API server on {args.host}:{args.port}")
        uvicorn.run(app, host=args.host, port=args.port, log_level="info")

    else:
        # Initialize CLI
        cli = RAGChatCLI(config)

        if args.command == 'index':
            path = Path(args.path)
            if path.is_file():
                cli.index_paper(str(path), force=args.force)
            elif path.is_dir():
                cli.index_directory(str(path), pattern=args.pattern, force=args.force)
            else:
                logger.error(f"Path not found: {path}")
                sys.exit(1)

        elif args.command == 'list':
            cli.list_papers()

        elif args.command == 'chat':
            cli.chat_interactive(args.papers)


if __name__ == '__main__':
    main()
