#!/bin/bash

# Docker testing script for Archivist

set -e

echo "🐳 Testing Docker deployment..."
echo ""

# Build
echo "1️⃣  Building Docker image..."
docker-compose build

# Check dependencies
echo ""
echo "2️⃣  Checking dependencies in container..."
docker-compose run --rm archivist check

# Test with sample PDF
echo ""
echo "3️⃣  Processing sample PDF..."
docker-compose run --rm archivist process lib/csit140108.pdf --force

# List processed papers
echo ""
echo "4️⃣  Listing processed papers..."
docker-compose run --rm archivist list

# Check status
echo ""
echo "5️⃣  Checking status..."
docker-compose run --rm archivist status lib/csit140108.pdf

echo ""
echo "✅ Docker tests complete!"
