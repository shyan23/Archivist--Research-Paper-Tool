#!/bin/bash
# Test graph building with one paper

echo "Testing graph building with paper: lib/2025.emnlp-main.28.pdf"
echo ""

# Process with force flag to trigger reprocessing (will publish to Kafka)
echo "y
n
y" | ./archivist process lib/2025.emnlp-main.28.pdf --force -m fast

echo ""
echo "============================================"
echo "Test complete! Check:"
echo "1. Neo4j Browser: http://localhost:7474"
echo "   Username: neo4j"
echo "   Password: password"
echo ""
echo "2. Run this query to see nodes:"
echo "   MATCH (n) RETURN n LIMIT 25"
echo "============================================"
