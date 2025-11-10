"""
Intelligent text chunking with LaTeX and section awareness
"""
import logging
import re
from typing import List, Dict, Tuple
from dataclasses import dataclass

logger = logging.getLogger(__name__)


@dataclass
class Chunk:
    """Text chunk with metadata"""
    text: str
    chunk_index: int
    source: str
    section: str = ""
    start_offset: int = 0
    end_offset: int = 0
    metadata: Dict[str, str] = None

    def __post_init__(self):
        if self.metadata is None:
            self.metadata = {}


class TextChunker:
    """Intelligent text chunker with section awareness"""

    def __init__(
        self,
        chunk_size: int = 1000,
        chunk_overlap: int = 200,
        respect_sections: bool = True
    ):
        """
        Initialize chunker

        Args:
            chunk_size: Target chunk size in characters
            chunk_overlap: Overlap between chunks in characters
            respect_sections: Whether to respect section boundaries
        """
        self.chunk_size = chunk_size
        self.chunk_overlap = chunk_overlap
        self.respect_sections = respect_sections

    def chunk_text(self, text: str, source: str = "unknown") -> List[Chunk]:
        """
        Chunk plain text with smart sentence boundaries

        Args:
            text: Text to chunk
            source: Source identifier (paper title, file path)

        Returns:
            List of Chunk objects
        """
        if not text or not text.strip():
            return []

        # Clean text
        text = self._clean_text(text)

        # Split into sentences
        sentences = self._split_into_sentences(text)

        if not sentences:
            return []

        chunks = []
        current_chunk = []
        current_length = 0
        chunk_index = 0

        for sentence in sentences:
            sentence_len = len(sentence)

            # Check if adding this sentence would exceed chunk size
            if current_length + sentence_len > self.chunk_size and current_chunk:
                # Create chunk
                chunk_text = " ".join(current_chunk)
                chunks.append(Chunk(
                    text=chunk_text,
                    chunk_index=chunk_index,
                    source=source,
                    start_offset=0,
                    end_offset=len(chunk_text)
                ))
                chunk_index += 1

                # Calculate overlap sentences
                overlap_sentences = self._calculate_overlap_sentences(
                    current_chunk,
                    self.chunk_overlap
                )

                # Start new chunk with overlap
                current_chunk = overlap_sentences
                current_length = sum(len(s) for s in overlap_sentences)

            # Add current sentence
            current_chunk.append(sentence)
            current_length += sentence_len + 1  # +1 for space

        # Add final chunk
        if current_chunk:
            chunk_text = " ".join(current_chunk)
            chunks.append(Chunk(
                text=chunk_text,
                chunk_index=chunk_index,
                source=source,
                start_offset=0,
                end_offset=len(chunk_text)
            ))

        logger.info(f"✓ Created {len(chunks)} chunks from text ({len(text)} chars)")
        return chunks

    def chunk_latex(self, latex_text: str, source: str = "unknown") -> List[Chunk]:
        """
        Chunk LaTeX text with section awareness

        Args:
            latex_text: LaTeX content
            source: Source identifier

        Returns:
            List of Chunk objects with section metadata
        """
        if self.respect_sections:
            sections = self._extract_latex_sections(latex_text)

            if sections:
                all_chunks = []
                global_index = 0

                for section_name, section_text in sections.items():
                    # Chunk each section separately
                    section_chunks = self.chunk_text(section_text, source)

                    # Add section metadata
                    for chunk in section_chunks:
                        chunk.section = section_name
                        chunk.chunk_index = global_index
                        chunk.metadata['section'] = section_name
                        all_chunks.append(chunk)
                        global_index += 1

                logger.info(f"✓ Created {len(all_chunks)} chunks from {len(sections)} sections")
                return all_chunks

        # Fallback to regular chunking
        return self.chunk_text(latex_text, source)

    def _clean_text(self, text: str) -> str:
        """Clean text by removing excessive whitespace"""
        # Remove excessive newlines
        text = re.sub(r'\n{3,}', '\n\n', text)

        # Remove excessive spaces
        text = re.sub(r'[ \t]+', ' ', text)

        # Remove common LaTeX commands
        text = re.sub(r'\\(textbf|textit|emph|texttt)\{([^}]+)\}', r'\2', text)
        text = re.sub(r'\\(cite|ref|label)\{[^}]+\}', '', text)

        return text.strip()

    def _split_into_sentences(self, text: str) -> List[str]:
        """Split text into sentences using regex"""
        # Pattern for sentence boundaries
        sentence_pattern = r'(?<=[.!?])\s+(?=[A-Z])'

        sentences = re.split(sentence_pattern, text)

        # Clean and filter
        sentences = [s.strip() for s in sentences if s.strip()]

        return sentences

    def _calculate_overlap_sentences(self, sentences: List[str], overlap_size: int) -> List[str]:
        """Calculate which sentences to include in overlap"""
        if not sentences:
            return []

        overlap_sentences = []
        current_size = 0

        # Work backwards from the end
        for sentence in reversed(sentences):
            sentence_len = len(sentence)
            if current_size + sentence_len > overlap_size:
                break
            overlap_sentences.insert(0, sentence)
            current_size += sentence_len + 1

        return overlap_sentences

    def _extract_latex_sections(self, latex_text: str) -> Dict[str, str]:
        """
        Extract sections from LaTeX document

        Returns:
            Dictionary mapping section names to section content
        """
        sections = {}

        # Patterns for different section levels
        patterns = [
            (r'\\section\{([^}]+)\}', 'section'),
            (r'\\subsection\{([^}]+)\}', 'subsection'),
            (r'\\subsubsection\{([^}]+)\}', 'subsubsection'),
        ]

        # Find all section markers
        section_markers = []

        for pattern, level in patterns:
            for match in re.finditer(pattern, latex_text):
                section_markers.append((
                    match.start(),
                    match.end(),
                    match.group(1),  # Section title
                    level
                ))

        if not section_markers:
            return sections

        # Sort by position
        section_markers.sort(key=lambda x: x[0])

        # Extract content between sections
        for i, (start, end, title, level) in enumerate(section_markers):
            # Find content start (after section command)
            content_start = end

            # Find content end (next section or end of document)
            if i < len(section_markers) - 1:
                content_end = section_markers[i + 1][0]
            else:
                content_end = len(latex_text)

            # Extract and clean content
            content = latex_text[content_start:content_end]
            content = self._clean_text(content)

            if content.strip():
                # Create section key
                section_key = f"{title} ({level})"
                sections[section_key] = content

        return sections


def create_chunks_from_pdf(
    pdf_path: str,
    chunk_size: int = 1000,
    chunk_overlap: int = 200
) -> List[Chunk]:
    """
    Extract text from PDF and create chunks

    Args:
        pdf_path: Path to PDF file
        chunk_size: Chunk size in characters
        chunk_overlap: Overlap in characters

    Returns:
        List of chunks
    """
    try:
        import fitz  # PyMuPDF
    except ImportError:
        raise ImportError("PyMuPDF not installed. Run: pip install PyMuPDF")

    # Extract text from PDF
    doc = fitz.open(pdf_path)
    full_text = ""

    for page in doc:
        full_text += page.get_text()

    doc.close()

    # Create chunks
    chunker = TextChunker(chunk_size, chunk_overlap)
    source = pdf_path.split('/')[-1].replace('.pdf', '')

    return chunker.chunk_text(full_text, source)


def create_chunks_from_latex(
    latex_path: str,
    chunk_size: int = 1000,
    chunk_overlap: int = 200,
    respect_sections: bool = True
) -> List[Chunk]:
    """
    Read LaTeX file and create chunks

    Args:
        latex_path: Path to .tex file
        chunk_size: Chunk size in characters
        chunk_overlap: Overlap in characters
        respect_sections: Whether to respect section boundaries

    Returns:
        List of chunks
    """
    with open(latex_path, 'r', encoding='utf-8') as f:
        latex_content = f.read()

    chunker = TextChunker(chunk_size, chunk_overlap, respect_sections)
    source = latex_path.split('/')[-1].replace('.tex', '')

    return chunker.chunk_latex(latex_content, source)
