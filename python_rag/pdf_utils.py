"""
PDF text extraction utilities for RAG system
Allows indexing research papers directly from PDFs
"""
import logging
from pathlib import Path
from typing import Optional

logger = logging.getLogger(__name__)


def extract_text_from_pdf(pdf_path: str) -> str:
    """
    Extract text content from a PDF file

    Args:
        pdf_path: Path to the PDF file

    Returns:
        Extracted text content

    Raises:
        FileNotFoundError: If PDF file doesn't exist
        ImportError: If required PDF library not installed
    """
    pdf_path = Path(pdf_path)

    if not pdf_path.exists():
        raise FileNotFoundError(f"PDF file not found: {pdf_path}")

    try:
        import PyPDF2

        logger.info(f"Extracting text from PDF: {pdf_path.name}")

        text_content = []

        with open(pdf_path, 'rb') as f:
            pdf_reader = PyPDF2.PdfReader(f)
            num_pages = len(pdf_reader.pages)

            logger.debug(f"  Processing {num_pages} pages...")

            for page_num, page in enumerate(pdf_reader.pages):
                try:
                    page_text = page.extract_text()
                    if page_text.strip():
                        text_content.append(page_text)
                except Exception as e:
                    logger.warning(f"  Failed to extract text from page {page_num + 1}: {e}")
                    continue

        full_text = "\n\n".join(text_content)

        if not full_text.strip():
            logger.warning(f"  No text extracted from {pdf_path.name}")
            return ""

        logger.info(f"  Extracted {len(full_text)} characters from {num_pages} pages")
        return full_text

    except ImportError:
        logger.error("PyPDF2 not installed. Install with: pip install PyPDF2")
        raise ImportError(
            "PDF processing requires PyPDF2. Install with: pip install PyPDF2"
        )


def extract_title_from_pdf(pdf_path: str) -> Optional[str]:
    """
    Extract title from PDF metadata

    Args:
        pdf_path: Path to PDF file

    Returns:
        PDF title from metadata, or None if not available
    """
    try:
        import PyPDF2

        with open(pdf_path, 'rb') as f:
            pdf_reader = PyPDF2.PdfReader(f)

            if pdf_reader.metadata:
                title = pdf_reader.metadata.get('/Title')
                if title:
                    return str(title)

    except Exception as e:
        logger.debug(f"Could not extract PDF metadata: {e}")

    return None


def get_pdf_info(pdf_path: str) -> dict:
    """
    Get information about a PDF file

    Args:
        pdf_path: Path to PDF file

    Returns:
        Dictionary with PDF information (pages, title, etc.)
    """
    try:
        import PyPDF2

        with open(pdf_path, 'rb') as f:
            pdf_reader = PyPDF2.PdfReader(f)

            info = {
                'num_pages': len(pdf_reader.pages),
                'title': extract_title_from_pdf(pdf_path),
                'filename': Path(pdf_path).name,
            }

            if pdf_reader.metadata:
                info['author'] = pdf_reader.metadata.get('/Author', None)
                info['subject'] = pdf_reader.metadata.get('/Subject', None)
                info['creator'] = pdf_reader.metadata.get('/Creator', None)

            return info

    except Exception as e:
        logger.error(f"Failed to get PDF info: {e}")
        return {'filename': Path(pdf_path).name, 'error': str(e)}
