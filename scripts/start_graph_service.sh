#!/bin/bash
cd /home/shyan/Desktop/Code/Archivist
GEMINI_KEY=$(cat .env | grep GEMINI_API_KEY | cut -d'=' -f2)

docker run -d --name archivist-graph-service \
  --network archivist_archivist-network \
  -p 8081:8081 \
  -e KAFKA_BOOTSTRAP_SERVERS=kafka:9092 \
  -e NEO4J_URI=bolt://neo4j:7687 \
  -e NEO4J_USER=neo4j \
  -e NEO4J_PASSWORD=password \
  -e REDIS_URL=redis://host.docker.internal:6379 \
  -e GEMINI_API_KEY=$GEMINI_KEY \
  --add-host host.docker.internal:host-gateway \
  --restart unless-stopped \
  archivist-graph-service

echo "Graph service started!"
sleep 10
docker logs archivist-graph-service --tail 30
``