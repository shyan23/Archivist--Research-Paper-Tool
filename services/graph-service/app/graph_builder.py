"""
Graph Builder - Neo4j operations for knowledge graph construction
"""

import logging
from typing import List, Dict, Any, Optional
from neo4j import AsyncGraphDatabase, AsyncDriver
from dataclasses import dataclass

logger = logging.getLogger(__name__)


@dataclass
class PaperMetadata:
    """Extracted paper metadata"""
    title: str
    authors: List[str]
    affiliations: List[str]
    year: int
    abstract: str
    venue: str
    venue_short: str
    methods: List[str]
    datasets: List[str]
    metrics: List[str]
    research_field: str


class GraphBuilder:
    """Handles all Neo4j graph operations"""

    def __init__(self, uri: str, username: str, password: str, database: str = "neo4j"):
        self.uri = uri
        self.username = username
        self.password = password
        self.database = database
        self.driver: Optional[AsyncDriver] = None

    async def __aenter__(self):
        """Async context manager entry"""
        await self.connect()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit"""
        await self.close()

    async def connect(self):
        """Connect to Neo4j"""
        self.driver = AsyncGraphDatabase.driver(
            self.uri,
            auth=(self.username, self.password)
        )
        logger.info(f"Connected to Neo4j at {self.uri}")

    async def close(self):
        """Close Neo4j connection"""
        if self.driver:
            await self.driver.close()
            logger.info("Neo4j connection closed")

    def is_connected(self) -> bool:
        """Check if connected to Neo4j"""
        return self.driver is not None

    async def initialize_schema(self):
        """Initialize Neo4j schema with constraints and indexes"""
        queries = [
            # Constraints (ensure uniqueness)
            "CREATE CONSTRAINT paper_title_unique IF NOT EXISTS FOR (p:Paper) REQUIRE p.title IS UNIQUE",
            "CREATE CONSTRAINT author_name_unique IF NOT EXISTS FOR (a:Author) REQUIRE a.name IS UNIQUE",
            "CREATE CONSTRAINT institution_name_unique IF NOT EXISTS FOR (i:Institution) REQUIRE i.name IS UNIQUE",
            "CREATE CONSTRAINT method_name_unique IF NOT EXISTS FOR (m:Method) REQUIRE m.name IS UNIQUE",
            "CREATE CONSTRAINT dataset_name_unique IF NOT EXISTS FOR (d:Dataset) REQUIRE d.name IS UNIQUE",
            "CREATE CONSTRAINT venue_name_unique IF NOT EXISTS FOR (v:Venue) REQUIRE v.name IS UNIQUE",

            # Indexes (for fast lookups)
            "CREATE INDEX paper_year_idx IF NOT EXISTS FOR (p:Paper) ON (p.year)",
            "CREATE INDEX author_field_idx IF NOT EXISTS FOR (a:Author) ON (a.field)",
            "CREATE INDEX method_type_idx IF NOT EXISTS FOR (m:Method) ON (m.type)",
        ]

        async with self.driver.session(database=self.database) as session:
            for query in queries:
                try:
                    await session.run(query)
                except Exception as e:
                    logger.warning(f"Schema query failed (may already exist): {e}")

        logger.info("✅ Neo4j schema initialized")

    async def add_paper_node(self, title: str, pdf_path: str, metadata: PaperMetadata):
        """Add a paper node to the graph"""
        query = """
        MERGE (p:Paper {title: $title})
        SET p.pdf_path = $pdf_path,
            p.year = $year,
            p.abstract = $abstract,
            p.authors = $authors,
            p.methodologies = $methods,
            p.datasets = $datasets,
            p.metrics = $metrics,
            p.processed_at = datetime()
        RETURN p.title
        """

        async with self.driver.session(database=self.database) as session:
            result = await session.run(query, {
                "title": title,
                "pdf_path": pdf_path,
                "year": metadata.year,
                "abstract": metadata.abstract[:500],  # Truncate long abstracts
                "authors": metadata.authors,
                "methods": metadata.methods,
                "datasets": metadata.datasets,
                "metrics": metadata.metrics
            })
            await result.consume()

        logger.info(f"  ✅ Added paper node: {title}")

    async def add_authors(self, paper_title: str, authors: List[str], affiliations: List[str]):
        """Add author nodes and link to paper"""
        if not authors:
            return

        for i, author_name in enumerate(authors):
            affiliation = affiliations[i] if i < len(affiliations) else ""

            # Create author node
            author_query = """
            MERGE (a:Author {name: $name})
            SET a.affiliation = COALESCE(a.affiliation, $affiliation)
            RETURN a.name
            """

            # Link paper to author
            link_query = """
            MATCH (p:Paper {title: $paper_title})
            MERGE (a:Author {name: $author_name})
            MERGE (p)-[r:WRITTEN_BY {position: $position, is_corresponding: $is_corresponding}]->(a)
            RETURN r
            """

            async with self.driver.session(database=self.database) as session:
                # Add author
                await session.run(author_query, {
                    "name": author_name,
                    "affiliation": affiliation
                })

                # Link to paper
                await session.run(link_query, {
                    "paper_title": paper_title,
                    "author_name": author_name,
                    "position": i + 1,
                    "is_corresponding": (i == 0)  # First author is corresponding
                })

            # Add institution if affiliation exists
            if affiliation:
                await self._add_institution(author_name, affiliation)

        logger.info(f"  ✅ Added {len(authors)} authors")

    async def _add_institution(self, author_name: str, institution_name: str):
        """Add institution node and link to author"""
        query = """
        MERGE (i:Institution {name: $institution_name})
        WITH i
        MATCH (a:Author {name: $author_name})
        MERGE (a)-[r:AFFILIATED_WITH]->(i)
        RETURN r
        """

        async with self.driver.session(database=self.database) as session:
            await session.run(query, {
                "author_name": author_name,
                "institution_name": institution_name
            })

    async def add_methods(self, paper_title: str, methods: List[str]):
        """Add method nodes and link to paper"""
        if not methods:
            return

        for method_name in methods:
            query = """
            MERGE (m:Method {name: $method_name})
            WITH m
            MATCH (p:Paper {title: $paper_title})
            MERGE (p)-[r:USES_METHOD {is_main_method: true}]->(m)
            RETURN r
            """

            async with self.driver.session(database=self.database) as session:
                await session.run(query, {
                    "paper_title": paper_title,
                    "method_name": method_name
                })

        logger.info(f"  ✅ Added {len(methods)} methods")

    async def add_datasets(self, paper_title: str, datasets: List[str]):
        """Add dataset nodes and link to paper"""
        if not datasets:
            return

        for dataset_name in datasets:
            query = """
            MERGE (d:Dataset {name: $dataset_name})
            WITH d
            MATCH (p:Paper {title: $paper_title})
            MERGE (p)-[r:USES_DATASET {purpose: 'evaluation'}]->(d)
            RETURN r
            """

            async with self.driver.session(database=self.database) as session:
                await session.run(query, {
                    "paper_title": paper_title,
                    "dataset_name": dataset_name
                })

        logger.info(f"  ✅ Added {len(datasets)} datasets")

    async def add_venue(self, paper_title: str, venue_name: str, year: int):
        """Add venue node and link to paper"""
        query = """
        MERGE (v:Venue {name: $venue_name})
        WITH v
        MATCH (p:Paper {title: $paper_title})
        MERGE (p)-[r:PUBLISHED_IN {year: $year}]->(v)
        RETURN r
        """

        async with self.driver.session(database=self.database) as session:
            await session.run(query, {
                "paper_title": paper_title,
                "venue_name": venue_name,
                "year": year
            })

        logger.info(f"  ✅ Added venue: {venue_name}")

    async def add_citations(self, source_paper: str, citations: List[Dict[str, Any]]):
        """Add citation relationships"""
        if not citations:
            return

        added = 0

        for citation in citations:
            cited_title = citation.get("title", "")

            if not cited_title:
                continue

            # Check if cited paper exists in graph
            exists = await self.paper_exists(cited_title)

            if not exists:
                continue

            # Add citation relationship
            query = """
            MATCH (source:Paper {title: $source_paper})
            MATCH (target:Paper {title: $cited_title})
            MERGE (source)-[r:CITES {
                importance: $importance,
                context: $context
            }]->(target)
            RETURN r
            """

            async with self.driver.session(database=self.database) as session:
                await session.run(query, {
                    "source_paper": source_paper,
                    "cited_title": cited_title,
                    "importance": citation.get("importance", "medium"),
                    "context": citation.get("context", "")[:200]  # Truncate
                })

            added += 1

        logger.info(f"  ✅ Added {added} citations")

    async def paper_exists(self, title: str) -> bool:
        """Check if a paper exists in the graph"""
        query = "MATCH (p:Paper {title: $title}) RETURN count(p) as count"

        async with self.driver.session(database=self.database) as session:
            result = await session.run(query, {"title": title})
            record = await result.single()
            return record["count"] > 0 if record else False

    async def get_stats(self) -> Dict[str, int]:
        """Get graph statistics"""
        queries = {
            "paper_count": "MATCH (p:Paper) RETURN count(p) as count",
            "author_count": "MATCH (a:Author) RETURN count(a) as count",
            "citation_count": "MATCH ()-[r:CITES]->() RETURN count(r) as count",
            "method_count": "MATCH (m:Method) RETURN count(m) as count",
            "dataset_count": "MATCH (d:Dataset) RETURN count(d) as count",
            "venue_count": "MATCH (v:Venue) RETURN count(v) as count",
            "institution_count": "MATCH (i:Institution) RETURN count(i) as count"
        }

        stats = {}

        async with self.driver.session(database=self.database) as session:
            for key, query in queries.items():
                result = await session.run(query)
                record = await result.single()
                stats[key] = record["count"] if record else 0

        return stats

    async def get_paper_details(self, title: str) -> Optional[Dict[str, Any]]:
        """Get detailed information about a paper"""
        query = """
        MATCH (p:Paper {title: $title})
        OPTIONAL MATCH (p)-[:WRITTEN_BY]->(a:Author)
        OPTIONAL MATCH (p)-[:USES_METHOD]->(m:Method)
        OPTIONAL MATCH (p)-[:USES_DATASET]->(d:Dataset)
        OPTIONAL MATCH (p)-[:CITES]->(cited:Paper)
        RETURN p,
               collect(DISTINCT a.name) as authors,
               collect(DISTINCT m.name) as methods,
               collect(DISTINCT d.name) as datasets,
               collect(DISTINCT cited.title) as citations
        """

        async with self.driver.session(database=self.database) as session:
            result = await session.run(query, {"title": title})
            record = await result.single()

            if not record:
                return None

            paper = record["p"]

            return {
                "title": paper["title"],
                "year": paper.get("year"),
                "abstract": paper.get("abstract"),
                "pdf_path": paper.get("pdf_path"),
                "authors": record["authors"],
                "methods": record["methods"],
                "datasets": record["datasets"],
                "citations": record["citations"]
            }

    async def delete_paper(self, title: str) -> bool:
        """Delete a paper and all its relationships"""
        query = """
        MATCH (p:Paper {title: $title})
        DETACH DELETE p
        RETURN count(p) as deleted
        """

        async with self.driver.session(database=self.database) as session:
            result = await session.run(query, {"title": title})
            record = await result.single()
            return record["deleted"] > 0 if record else False

    async def clear_all(self):
        """Clear entire graph (use with caution!)"""
        query = "MATCH (n) DETACH DELETE n"

        async with self.driver.session(database=self.database) as session:
            await session.run(query)

        logger.warning("⚠️  Graph cleared!")

    async def execute_query(self, query_type: str, parameters: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Execute predefined graph queries"""
        queries = {
            "most_cited": """
                MATCH (p:Paper)<-[r:CITES]-()
                RETURN p.title as title, count(r) as citations
                ORDER BY citations DESC
                LIMIT $limit
            """,
            "author_papers": """
                MATCH (a:Author {name: $author_name})<-[:WRITTEN_BY]-(p:Paper)
                RETURN p.title as title, p.year as year
                ORDER BY p.year DESC
            """,
            "method_papers": """
                MATCH (m:Method {name: $method_name})<-[:USES_METHOD]-(p:Paper)
                RETURN p.title as title, p.year as year
                ORDER BY p.year DESC
            """,
            "collaboration_network": """
                MATCH (a1:Author {name: $author_name})<-[:WRITTEN_BY]-(p:Paper)-[:WRITTEN_BY]->(a2:Author)
                WHERE a1 <> a2
                RETURN DISTINCT a2.name as collaborator, count(p) as joint_papers
                ORDER BY joint_papers DESC
            """
        }

        query = queries.get(query_type)

        if not query:
            raise ValueError(f"Unknown query type: {query_type}")

        async with self.driver.session(database=self.database) as session:
            result = await session.run(query, parameters)
            records = await result.values()

            return [dict(record) for record in records]
