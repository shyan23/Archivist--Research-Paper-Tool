package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// CachedAnalysis represents a cached analysis result
type CachedAnalysis struct {
	ContentHash  string    `json:"content_hash"`
	PaperTitle   string    `json:"paper_title"`
	LatexContent string    `json:"latex_content"`
	CachedAt     time.Time `json:"cached_at"`
	ModelUsed    string    `json:"model_used"`
}

// RedisCache handles Redis-based caching for paper analysis
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(addr, password string, db int, ttl time.Duration) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("✓ Connected to Redis at %s", addr)

	return &RedisCache{
		client: client,
		ttl:    ttl,
		prefix: "archivist:analysis:",
	}, nil
}

// Close closes the Redis connection
func (rc *RedisCache) Close() error {
	return rc.client.Close()
}

// Get retrieves a cached analysis result by content hash
func (rc *RedisCache) Get(ctx context.Context, contentHash string) (*CachedAnalysis, error) {
	key := rc.prefix + contentHash

	data, err := rc.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		// Cache miss - not an error
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get from cache: %w", err)
	}

	var cached CachedAnalysis
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached data: %w", err)
	}

	log.Printf("  🎯 Cache HIT for hash: %s (cached %.1f hours ago)",
		contentHash[:12], time.Since(cached.CachedAt).Hours())

	return &cached, nil
}

// Set stores an analysis result in the cache
func (rc *RedisCache) Set(ctx context.Context, contentHash string, analysis *CachedAnalysis) error {
	key := rc.prefix + contentHash

	analysis.CachedAt = time.Now()
	analysis.ContentHash = contentHash

	data, err := json.Marshal(analysis)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	if err := rc.client.Set(ctx, key, data, rc.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	ttlHours := rc.ttl.Hours()
	log.Printf("  💾 Cached analysis for hash: %s (TTL: %.0f hours)", contentHash[:12], ttlHours)

	return nil
}

// Clear removes all cached entries with the archivist prefix
func (rc *RedisCache) Clear(ctx context.Context) (int64, error) {
	iter := rc.client.Scan(ctx, 0, rc.prefix+"*", 0).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return 0, fmt.Errorf("failed to scan keys: %w", err)
	}

	if len(keys) == 0 {
		return 0, nil
	}

	deleted, err := rc.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to delete keys: %w", err)
	}

	return deleted, nil
}

// GetStats returns cache statistics
func (rc *RedisCache) GetStats(ctx context.Context) (int64, error) {
	iter := rc.client.Scan(ctx, 0, rc.prefix+"*", 0).Iterator()

	var count int64
	for iter.Next(ctx) {
		count++
	}

	if err := iter.Err(); err != nil {
		return 0, fmt.Errorf("failed to scan keys: %w", err)
	}

	return count, nil
}

// Exists checks if a cache entry exists for the given content hash
func (rc *RedisCache) Exists(ctx context.Context, contentHash string) (bool, error) {
	key := rc.prefix + contentHash

	result, err := rc.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return result > 0, nil
}

// Delete removes a specific cache entry by content hash
func (rc *RedisCache) Delete(ctx context.Context, contentHash string) error {
	key := rc.prefix + contentHash

	deleted, err := rc.client.Del(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to delete cache entry: %w", err)
	}

	if deleted == 0 {
		return fmt.Errorf("cache entry not found")
	}

	log.Printf("  🗑️  Deleted cache entry for hash: %s", contentHash[:12])
	return nil
}

// ListAll returns all cached entries with their metadata
func (rc *RedisCache) ListAll(ctx context.Context) ([]*CachedAnalysis, error) {
	iter := rc.client.Scan(ctx, 0, rc.prefix+"*", 0).Iterator()

	var entries []*CachedAnalysis
	for iter.Next(ctx) {
		key := iter.Val()

		data, err := rc.client.Get(ctx, key).Bytes()
		if err != nil {
			log.Printf("Warning: failed to get data for key %s: %v", key, err)
			continue
		}

		var cached CachedAnalysis
		if err := json.Unmarshal(data, &cached); err != nil {
			log.Printf("Warning: failed to unmarshal data for key %s: %v", key, err)
			continue
		}

		entries = append(entries, &cached)
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan keys: %w", err)
	}

	return entries, nil
}
