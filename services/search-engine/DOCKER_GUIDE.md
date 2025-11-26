# Docker Deployment Guide for Archivist Search Engine

This guide covers everything you need to know about running the search engine service with Docker.

## Prerequisites

- Docker installed (version 20.10+)
- Docker Compose installed (version 1.29+)
- Gemini API key (for embeddings and semantic search)

Check your Docker installation:
```bash
docker --version
docker-compose --version
```

## Quick Start

1. **Navigate to the service directory:**
   ```bash
   cd services/search-engine
   ```

2. **Create environment file:**
   ```bash
   cp .env.example .env
   ```

3. **Edit `.env` and add your Gemini API key:**
   ```bash
   nano .env  # or use your preferred editor
   ```

   Add:
   ```
   GEMINI_API_KEY=your_actual_api_key_here
   ```

4. **Start the service:**
   ```bash
   docker-compose up -d
   ```

5. **Verify it's running:**
   ```bash
   # Check container status
   docker-compose ps

   # Check health
   curl http://localhost:8000/health

   # View API docs
   open http://localhost:8000/docs
   ```

## Common Operations

### Starting the Service

```bash
# Start in background (detached mode)
docker-compose up -d

# Start with logs visible
docker-compose up

# Start and rebuild if needed
docker-compose up -d --build
```

### Viewing Logs

```bash
# Follow logs in real-time
docker-compose logs -f

# View last 100 lines
docker-compose logs --tail=100

# View logs for specific time
docker-compose logs --since 10m
```

### Stopping the Service

```bash
# Stop containers but keep data
docker-compose stop

# Stop and remove containers (keeps volumes)
docker-compose down

# Stop, remove containers AND delete vector store data
docker-compose down -v
```

### Restarting the Service

```bash
# Restart without rebuilding
docker-compose restart

# Restart with rebuild
docker-compose up -d --build
```

## Directory Structure & Data Persistence

```
services/search-engine/
├── Dockerfile               # Container build instructions
├── docker-compose.yml       # Service orchestration
├── .env                     # Environment variables (create from .env.example)
├── .env.example            # Template for environment variables
├── data/                   # Persistent data (created automatically)
│   └── qdrant/            # Vector store data
└── app/                    # Application code
```

### Data Persistence

The vector store data is persisted in `./data/qdrant/` on your host machine. This means:
- Data survives container restarts
- You can backup the `data/` directory
- Deleting the container won't delete your indexed papers

To clear the vector store:
```bash
# Option 1: Use API endpoint
curl -X DELETE http://localhost:8000/api/vector-store/clear

# Option 2: Delete data directory
docker-compose down
rm -rf data/qdrant
docker-compose up -d
```

## Configuration

### Environment Variables

Edit `.env` file to customize:

```bash
# Required
GEMINI_API_KEY=your_key_here

# Optional (with defaults)
QDRANT_PATH=/app/data/qdrant
LOG_LEVEL=info
```

### Port Configuration

To change the port, edit `docker-compose.yml`:

```yaml
services:
  search-engine:
    ports:
      - "9000:8000"  # Change 9000 to your desired port
```

Then restart:
```bash
docker-compose down
docker-compose up -d
```

## Troubleshooting

### Container Won't Start

**Check logs:**
```bash
docker-compose logs
```

**Common issues:**
- Missing `.env` file → Copy from `.env.example`
- Invalid Gemini API key → Check key in `.env`
- Port 8000 already in use → Change port in `docker-compose.yml`

### Service is Slow or Unresponsive

**Check resource usage:**
```bash
docker stats archivist-search-engine
```

**Restart the service:**
```bash
docker-compose restart
```

### Permission Issues with Data Directory

```bash
# Fix permissions
sudo chown -R $USER:$USER data/
```

### Container Keeps Restarting

**Check health status:**
```bash
docker inspect archivist-search-engine | grep -A 10 Health
```

**View detailed logs:**
```bash
docker-compose logs --tail=50
```

## Health Monitoring

The service includes built-in health checks:

```bash
# Check via API
curl http://localhost:8000/health

# Check Docker health status
docker ps --filter name=archivist-search-engine --format "{{.Status}}"

# View health check logs
docker inspect archivist-search-engine | jq '.[0].State.Health'
```

## Updating the Service

When you pull new code changes:

```bash
# Pull latest changes
git pull

# Rebuild and restart
cd services/search-engine
docker-compose down
docker-compose up -d --build

# Verify update
curl http://localhost:8000/health
```

## Production Deployment

### Security Recommendations

1. **Use secrets management:**
   ```bash
   # Don't commit .env file
   echo ".env" >> .gitignore

   # Use Docker secrets in production
   docker secret create gemini_key -
   ```

2. **Limit resource usage:**

   Add to `docker-compose.yml`:
   ```yaml
   services:
     search-engine:
       deploy:
         resources:
           limits:
             cpus: '2.0'
             memory: 2G
           reservations:
             memory: 512M
   ```

3. **Use reverse proxy:**
   ```bash
   # Example nginx config
   location /search/ {
       proxy_pass http://localhost:8000/;
       proxy_set_header Host $host;
       proxy_set_header X-Real-IP $remote_addr;
   }
   ```

### Monitoring

**Set up logging:**
```yaml
services:
  search-engine:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

**Monitor with Prometheus:**
```bash
# Add metrics endpoint (future enhancement)
curl http://localhost:8000/metrics
```

## Integration with Archivist Go Application

The Go application connects to the search engine via HTTP:

```go
import "github.com/yourusername/archivist/internal/search"

// Create client
client := search.NewClient("http://localhost:8000")

// Check if service is running
if !client.IsServiceRunning() {
    log.Fatal("Search engine not running. Start with: docker-compose up -d")
}

// Perform search
results, err := client.Search(&search.SearchQuery{
    Query:      "transformer architecture",
    MaxResults: 20,
})
```

## Backup and Restore

### Backup Vector Store

```bash
# Stop service
docker-compose stop

# Backup data
tar -czf search-engine-backup-$(date +%Y%m%d).tar.gz data/

# Restart service
docker-compose start
```

### Restore Vector Store

```bash
# Stop service
docker-compose stop

# Restore data
tar -xzf search-engine-backup-20250112.tar.gz

# Restart service
docker-compose start
```

## Development Mode

For local development with hot-reload:

1. **Uncomment volume mount in `docker-compose.yml`:**
   ```yaml
   volumes:
     - ./app:/app/app  # Uncomment this line
   ```

2. **Restart with rebuild:**
   ```bash
   docker-compose up -d --build
   ```

Now code changes will auto-reload without rebuilding the container.

## Getting Help

- View API documentation: http://localhost:8000/docs
- Check service status: `docker-compose ps`
- View logs: `docker-compose logs -f`
- Check health: `curl http://localhost:8000/health`

## Additional Resources

- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [FastAPI Documentation](https://fastapi.tiangolo.com/)
- [Archivist Main README](../../README.md)
