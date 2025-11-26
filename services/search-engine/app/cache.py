"""
Redis cache manager for the Archivist Search Engine.

Caches search results to reduce API calls and improve response times.
"""

import os
import json
import hashlib
from typing import List, Optional
from datetime import timedelta
import redis.asyncio as aioredis
from redis.asyncio import Redis
from .models import SearchResult


class RedisCache:
    """Redis-based cache for search results."""

    def __init__(self, redis_url: Optional[str] = None, ttl_hours: int = 24):
        """
        Initialize Redis cache.

        Args:
            redis_url: Redis connection URL (default: redis://localhost:6379)
            ttl_hours: Time-to-live for cached entries in hours (default: 24)
        """
        self.redis_url = redis_url or os.getenv("REDIS_URL", "redis://localhost:6379")
        self.ttl = timedelta(hours=ttl_hours)
        self._client: Optional[Redis] = None

    async def connect(self):
        """Establish connection to Redis."""
        if self._client is None:
            self._client = await aioredis.from_url(
                self.redis_url,
                encoding="utf-8",
                decode_responses=True
            )

    async def disconnect(self):
        """Close Redis connection."""
        if self._client:
            await self._client.close()
            self._client = None

    def _generate_cache_key(self, query: str, sources: List[str], max_results: int) -> str:
        """
        Generate a unique cache key for a search query.

        Args:
            query: Search query string
            sources: List of sources to search
            max_results: Maximum number of results

        Returns:
            MD5 hash of the query parameters
        """
        # Create a deterministic string from query parameters
        cache_data = {
            "query": query.lower().strip(),
            "sources": sorted(sources),
            "max_results": max_results
        }
        cache_string = json.dumps(cache_data, sort_keys=True)

        # Generate MD5 hash
        hash_object = hashlib.md5(cache_string.encode())
        return f"search:{hash_object.hexdigest()}"

    async def get_cached_results(
        self,
        query: str,
        sources: List[str],
        max_results: int
    ) -> Optional[List[SearchResult]]:
        """
        Retrieve cached search results.

        Args:
            query: Search query string
            sources: List of sources to search
            max_results: Maximum number of results

        Returns:
            List of SearchResult objects if cached, None otherwise
        """
        if not self._client:
            await self.connect()

        cache_key = self._generate_cache_key(query, sources, max_results)

        try:
            cached_data = await self._client.get(cache_key)

            if cached_data:
                # Deserialize cached results
                results_data = json.loads(cached_data)
                return [SearchResult(**result) for result in results_data]

            return None

        except Exception as e:
            print(f"Cache retrieval error: {e}")
            return None

    async def cache_results(
        self,
        query: str,
        sources: List[str],
        max_results: int,
        results: List[SearchResult]
    ) -> bool:
        """
        Cache search results.

        Args:
            query: Search query string
            sources: List of sources to search
            max_results: Maximum number of results
            results: List of SearchResult objects to cache

        Returns:
            True if successfully cached, False otherwise
        """
        if not self._client:
            await self.connect()

        cache_key = self._generate_cache_key(query, sources, max_results)

        try:
            # Serialize results to JSON
            results_data = [result.model_dump() for result in results]
            cached_data = json.dumps(results_data, default=str)

            # Store in Redis with TTL
            await self._client.setex(
                cache_key,
                int(self.ttl.total_seconds()),
                cached_data
            )

            return True

        except Exception as e:
            print(f"Cache storage error: {e}")
            return False

    async def invalidate_cache(self, query: str, sources: List[str], max_results: int) -> bool:
        """
        Invalidate cached results for a specific query.

        Args:
            query: Search query string
            sources: List of sources to search
            max_results: Maximum number of results

        Returns:
            True if successfully invalidated, False otherwise
        """
        if not self._client:
            await self.connect()

        cache_key = self._generate_cache_key(query, sources, max_results)

        try:
            await self._client.delete(cache_key)
            return True
        except Exception as e:
            print(f"Cache invalidation error: {e}")
            return False

    async def clear_all_cache(self) -> bool:
        """
        Clear all cached search results.

        Returns:
            True if successfully cleared, False otherwise
        """
        if not self._client:
            await self.connect()

        try:
            # Find all search cache keys
            keys = []
            async for key in self._client.scan_iter(match="search:*"):
                keys.append(key)

            if keys:
                await self._client.delete(*keys)

            return True

        except Exception as e:
            print(f"Cache clear error: {e}")
            return False

    async def get_cache_stats(self) -> dict:
        """
        Get cache statistics.

        Returns:
            Dictionary with cache statistics
        """
        if not self._client:
            await self.connect()

        try:
            # Count cache keys
            cached_queries = 0
            async for _ in self._client.scan_iter(match="search:*"):
                cached_queries += 1

            # Get Redis info
            info = await self._client.info("memory")

            return {
                "cached_queries": cached_queries,
                "memory_used_bytes": info.get("used_memory", 0),
                "memory_used_human": info.get("used_memory_human", "0B"),
                "ttl_hours": self.ttl.total_seconds() / 3600,
                "redis_url": self.redis_url.split("@")[-1] if "@" in self.redis_url else self.redis_url
            }

        except Exception as e:
            print(f"Cache stats error: {e}")
            return {
                "error": str(e),
                "cached_queries": 0
            }


# Global cache instance
_cache_instance: Optional[RedisCache] = None


def get_cache() -> RedisCache:
    """Get or create the global cache instance."""
    global _cache_instance
    if _cache_instance is None:
        _cache_instance = RedisCache()
    return _cache_instance
