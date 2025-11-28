# Archivist Product Strategy: Your Second Brain for Research

## The Vision: Beyond Storage—Towards Insight

Today, Archivist is a tool for organizing documents. Tomorrow, it will be an extension of the researcher's mind—a second brain that doesn't just store information but connects ideas, surfaces insights, and accelerates discovery.

Our guiding principle will be the relentless pursuit of simplicity and power. The user should feel that the system anticipates their needs. The technology should be invisible, leaving only a fluid, intuitive experience.

## The Foundation: Nailing the Basics

My investigation revealed that the core ingestion pipeline relies on an AI model to parse basic metadata like titles and authors from PDFs. This is fundamentally fragile. It is slow, non-deterministic, and dependent on a network connection. It is the opposite of "it just works."

**Phase 1 is to rebuild this foundation.**

1.  **Deterministic Ingestion:** We will replace the LLM-based PDF parsing with robust, local, and instantaneous libraries (e.g., `unidoc` or similar). Text and basic metadata will be extracted with 100% reliability.
2.  **Intelligent Enrichment:** After reliable extraction, we will use academic APIs (CrossRef, ArXiv, Semantic Scholar) to fetch canonical metadata. This guarantees accuracy.
3.  **Repurpose AI for Magic, Not Mechanics:** The Gemini models will no longer be used for basic parsing. They will be applied *after* ingestion to perform tasks that feel like magic:
    *   Deep-semantic summarization.
    *   Identifying key concepts and methodologies.
    *   Generating potential research questions.

## Pillar I: Effortless Collection

The act of adding a paper should be as simple as a thought.

*   **Universal Inbox:** Drag-and-drop papers, folders, or even BibTeX files directly into Archivist.
*   **Connector Ecosystem:** Direct, two-way sync with Zotero, Mendeley, and Google Scholar.
*   **Web Clipper:** A browser extension to capture papers, articles, and websites with a single click.

## Pillar II: Discovery Through Conversation & Exploration

The search bar is dead. Discovery is not about finding what you know; it's about uncovering what you don't.

*   **The Graph is the Interface:** The primary view will be a dynamic, visual knowledge graph. Users will fly through their library, seeing connections and clusters intuitively. The work in `tui/graph.go` is the seed of this, but it will become the entire experience.
*   **Conversational Query:** The user will simply ask questions in natural language.
    *   *"Show me the foundational papers for protein folding."*
    *   *"What are the main counter-arguments to this paper?"*
    *   *"Which authors in my library are collaborating on new work?"*
*   **The Daily Briefing:** Proactive notifications powered by the Kafka pipeline (`internal/graph/kafka_producer.go`). *"A new preprint was published that cites two of your most-read papers. Here is a summary."*

## Pillar III: The Insight Engine

This is where Archivist becomes an active research partner.

*   **Automated Synthesis:** Select multiple papers and ask Archivist to "Compare the methodologies" or "Synthesize the key findings."
*   **Hypothesis Generation:** Archivist will analyze the "white space" in your knowledge graph to suggest unexplored connections. *"You have a deep collection on 'Graph Neural Networks' and 'Causal Inference.' The intersection is a hot research area. Here are three potential research questions."*
*   **Personalized Summaries:** Summaries are generated not in isolation, but in the context of your existing library, highlighting points of connection or contradiction with what you already know.

## The Path Forward

This is not an incremental update. It is a strategic pivot. We will focus first on fortifying the ingestion pipeline, as it is the bedrock of the entire experience. Only with a solid foundation can we build the revolutionary product that researchers deserve.
