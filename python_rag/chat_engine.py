"""
Chat engine with RAG support
Handles conversations about research papers using retrieval augmented generation
"""
import logging
import time
import json
from typing import List, Optional, Dict, Any
from dataclasses import dataclass, asdict
from pathlib import Path

from .retriever import Retriever, RetrievedContext

logger = logging.getLogger(__name__)


@dataclass
class Message:
    """Chat message"""
    role: str  # "user" or "assistant"
    content: str
    timestamp: float
    citations: List[str] = None

    def __post_init__(self):
        if self.citations is None:
            self.citations = []


@dataclass
class ChatSession:
    """Chat session with history"""
    session_id: str
    paper_titles: List[str]
    messages: List[Message]
    created_at: float
    last_updated: float


class ChatEngine:
    """Chat engine with RAG capabilities"""

    def __init__(
        self,
        retriever: Retriever,
        llm_provider: str = "gemini",
        gemini_api_key: Optional[str] = None,
        model_name: str = "models/gemini-2.0-flash-exp",
        temperature: float = 0.7,
        max_tokens: int = 8000
    ):
        """
        Initialize chat engine

        Args:
            retriever: RAG retriever
            llm_provider: LLM provider ("gemini", "openai")
            gemini_api_key: Gemini API key
            model_name: Model name
            temperature: Sampling temperature
            max_tokens: Maximum tokens to generate
        """
        self.retriever = retriever
        self.llm_provider = llm_provider.lower()
        self.model_name = model_name
        self.temperature = temperature
        self.max_tokens = max_tokens

        # Initialize LLM client
        if self.llm_provider == "gemini":
            if not gemini_api_key:
                raise ValueError("Gemini API key required")

            import google.generativeai as genai
            genai.configure(api_key=gemini_api_key)
            self.model = genai.GenerativeModel(model_name)
            logger.info(f"✓ Gemini chat engine initialized: {model_name}")

        elif self.llm_provider == "openai":
            if not gemini_api_key:  # Reuse for OpenAI key
                raise ValueError("OpenAI API key required")

            from openai import OpenAI
            self.client = OpenAI(api_key=gemini_api_key)
            logger.info(f"✓ OpenAI chat engine initialized: {model_name}")

        else:
            raise ValueError(f"Unsupported LLM provider: {llm_provider}")

        # Session storage
        self.sessions: Dict[str, ChatSession] = {}

    def create_session(self, paper_titles: List[str]) -> ChatSession:
        """
        Create a new chat session

        Args:
            paper_titles: Papers to chat about

        Returns:
            ChatSession
        """
        session_id = f"session_{int(time.time() * 1000)}"
        now = time.time()

        session = ChatSession(
            session_id=session_id,
            paper_titles=paper_titles,
            messages=[],
            created_at=now,
            last_updated=now
        )

        self.sessions[session_id] = session

        logger.info(f"✓ Created chat session {session_id} with {len(paper_titles)} papers")

        return session

    def chat(
        self,
        session_id: str,
        user_message: str,
        stream: bool = False
    ) -> Message:
        """
        Process a chat message with RAG

        Args:
            session_id: Session ID
            user_message: User's message
            stream: Whether to stream response (not implemented)

        Returns:
            Assistant message with response and citations
        """
        if session_id not in self.sessions:
            raise ValueError(f"Session not found: {session_id}")

        session = self.sessions[session_id]

        logger.info(f"💬 User: {self._truncate(user_message, 60)}")

        # Add user message to history
        user_msg = Message(
            role="user",
            content=user_message,
            timestamp=time.time()
        )
        session.messages.append(user_msg)

        # Retrieve relevant context
        logger.info("  🔍 Retrieving relevant context...")

        if not session.paper_titles:
            # No specific papers, search all
            context = self.retriever.retrieve(user_message)
        elif len(session.paper_titles) == 1:
            # Single paper
            context = self.retriever.retrieve_from_paper(
                user_message,
                session.paper_titles[0]
            )
        else:
            # Multiple papers
            context = self.retriever.retrieve_multi_paper(
                user_message,
                session.paper_titles
            )

        logger.info(f"  ✓ Retrieved {context.total_chunks} chunks from {len(context.sources)} sources")

        # Build prompt
        prompt = self._build_prompt(session, user_message, context)

        # Generate response
        logger.info("  🤖 Generating response...")
        response_text = self._generate_response(prompt)

        # Extract citations
        citations = self._extract_citations(context)

        # Create assistant message
        assistant_msg = Message(
            role="assistant",
            content=response_text,
            timestamp=time.time(),
            citations=citations
        )

        # Add to session
        session.messages.append(assistant_msg)
        session.last_updated = time.time()

        logger.info(f"  ✓ Response generated ({len(response_text)} chars)")

        return assistant_msg

    def get_session(self, session_id: str) -> Optional[ChatSession]:
        """Get a chat session"""
        return self.sessions.get(session_id)

    def get_all_sessions(self) -> List[ChatSession]:
        """Get all active sessions"""
        return list(self.sessions.values())

    def delete_session(self, session_id: str) -> bool:
        """Delete a session"""
        if session_id in self.sessions:
            del self.sessions[session_id]
            logger.info(f"✓ Deleted session {session_id}")
            return True
        return False

    def export_session_to_latex(self, session_id: str) -> str:
        """
        Export chat session to LaTeX format

        Args:
            session_id: Session ID

        Returns:
            LaTeX formatted conversation
        """
        if session_id not in self.sessions:
            raise ValueError(f"Session not found: {session_id}")

        session = self.sessions[session_id]

        latex = "\\section{Q\\&A Session}\n\n"

        if session.paper_titles:
            latex += "\\subsection{Papers Discussed}\n"
            latex += "\\begin{itemize}\n"
            for title in session.paper_titles:
                latex += f"  \\item {self._escape_latex(title)}\n"
            latex += "\\end{itemize}\n\n"

        latex += "\\subsection{Conversation}\n\n"

        qa_number = 1
        for msg in session.messages:
            if msg.role == "user":
                latex += f"\\textbf{{Question {qa_number}:}} {self._escape_latex(msg.content)}\n\n"
            else:
                latex += f"\\textbf{{Answer:}} {self._escape_latex(msg.content)}\n\n"

                if msg.citations:
                    latex += "\\textit{Sources:} "
                    latex += ", ".join(self._escape_latex(c) for c in msg.citations)
                    latex += "\n\n"

                qa_number += 1

        return latex

    def _build_prompt(
        self,
        session: ChatSession,
        user_message: str,
        context: RetrievedContext
    ) -> str:
        """Build RAG prompt with context and history"""
        prompt_parts = []

        # System instruction
        prompt_parts.append(
            "You are a helpful AI research assistant for CS students studying AI/ML, "
            "Computer Vision, and Networking papers.\n\n"
        )

        # Papers being discussed
        if session.paper_titles:
            prompt_parts.append("You are discussing the following papers:\n")
            for title in session.paper_titles:
                prompt_parts.append(f"- {title}\n")
            prompt_parts.append("\n")

        # Retrieved context
        prompt_parts.append("RELEVANT CONTEXT FROM PAPERS:\n")
        prompt_parts.append("---\n")
        prompt_parts.append(context.context_text)
        prompt_parts.append("---\n\n")

        # Conversation history (last 3 exchanges)
        if len(session.messages) > 1:
            prompt_parts.append("CONVERSATION HISTORY:\n")

            # Get last 6 messages (3 Q&A pairs)
            history_messages = session.messages[max(0, len(session.messages) - 7):-1]

            for msg in history_messages:
                if msg.role == "user":
                    prompt_parts.append(f"User: {msg.content}\n")
                else:
                    prompt_parts.append(f"Assistant: {msg.content}\n")

            prompt_parts.append("\n")

        # Current question
        prompt_parts.append("CURRENT QUESTION:\n")
        prompt_parts.append(f"{user_message}\n\n")

        # Instructions
        prompt_parts.append("INSTRUCTIONS:\n")
        prompt_parts.append("- Answer the question using the provided context from the papers.\n")
        prompt_parts.append("- Be clear, concise, and student-friendly.\n")
        prompt_parts.append("- Cite specific sections when referencing information.\n")
        prompt_parts.append("- If the context doesn't contain enough information, say so.\n")
        prompt_parts.append("- Use technical terms but explain them when first introduced.\n")
        prompt_parts.append("- If comparing multiple papers, clearly distinguish between them.\n\n")

        prompt_parts.append("ANSWER:\n")

        return "".join(prompt_parts)

    def _generate_response(self, prompt: str) -> str:
        """Generate LLM response"""
        if self.llm_provider == "gemini":
            response = self.model.generate_content(
                prompt,
                generation_config={
                    'temperature': self.temperature,
                    'max_output_tokens': self.max_tokens,
                }
            )
            return response.text

        elif self.llm_provider == "openai":
            response = self.client.chat.completions.create(
                model=self.model_name,
                messages=[{"role": "user", "content": prompt}],
                temperature=self.temperature,
                max_tokens=self.max_tokens
            )
            return response.choices[0].message.content

        else:
            raise NotImplementedError(f"LLM provider not implemented: {self.llm_provider}")

    def _extract_citations(self, context: RetrievedContext) -> List[str]:
        """Extract citation strings from retrieved context"""
        citations = []
        seen = set()

        for chunk in context.chunks:
            doc = chunk.document
            source = doc.metadata.get('source', 'unknown')
            section = doc.metadata.get('section', '')

            if section:
                citation = f"{source} (Section: {section})"
            else:
                citation = source

            if citation not in seen:
                seen.add(citation)
                citations.append(citation)

        return citations

    @staticmethod
    def _truncate(text: str, max_len: int) -> str:
        """Truncate text"""
        if len(text) <= max_len:
            return text
        return text[:max_len] + "..."

    @staticmethod
    def _escape_latex(text: str) -> str:
        """Escape LaTeX special characters"""
        replacements = {
            '\\': '\\textbackslash{}',
            '&': '\\&',
            '%': '\\%',
            '$': '\\$',
            '#': '\\#',
            '_': '\\_',
            '{': '\\{',
            '}': '\\}',
            '~': '\\textasciitilde{}',
            '^': '\\textasciicircum{}',
        }

        for char, replacement in replacements.items():
            text = text.replace(char, replacement)

        return text


def create_chat_engine(
    retriever: Retriever,
    gemini_api_key: Optional[str] = None,
    model_name: str = "models/gemini-2.0-flash-exp",
    temperature: float = 0.7
) -> ChatEngine:
    """
    Factory function to create chat engine

    Args:
        retriever: RAG retriever
        gemini_api_key: Gemini API key
        model_name: Model name
        temperature: Sampling temperature

    Returns:
        ChatEngine instance
    """
    return ChatEngine(
        retriever=retriever,
        llm_provider="gemini",
        gemini_api_key=gemini_api_key,
        model_name=model_name,
        temperature=temperature
    )
