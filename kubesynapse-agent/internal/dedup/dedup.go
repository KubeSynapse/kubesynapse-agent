// Package dedup provides alert deduplication to prevent alert storms.
// It supports both in-memory (local) and Redis (distributed) backends.
package dedup

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// Deduplicator defines the interface for alert deduplication.
type Deduplicator interface {
	ShouldAlert(ctx context.Context, namespace, podName string) bool
	Reset(ctx context.Context, namespace, podName string)
}

// InMemoryCache provides a local, thread-safe deduplication cache.
type InMemoryCache struct {
	mu       sync.RWMutex
	entries  map[string]time.Time
	cooldown time.Duration
}

// NewInMemory creates a new local dedup cache.
func NewInMemory(cooldownSeconds int) *InMemoryCache {
	c := &InMemoryCache{
		entries:  make(map[string]time.Time),
		cooldown: time.Duration(cooldownSeconds) * time.Second,
	}
	go c.cleanup()
	return c
}

func (c *InMemoryCache) ShouldAlert(_ context.Context, namespace, podName string) bool {
	key := cacheKey(namespace, podName)
	c.mu.Lock()
	defer c.mu.Unlock()

	if lastSeen, exists := c.entries[key]; exists {
		if time.Since(lastSeen) < c.cooldown {
			return false
		}
	}

	c.entries[key] = time.Now()
	return true
}

func (c *InMemoryCache) Reset(_ context.Context, namespace, podName string) {
	key := cacheKey(namespace, podName)
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

func (c *InMemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, ts := range c.entries {
			if now.Sub(ts) > c.cooldown {
				delete(c.entries, key)
			}
		}
		c.mu.Unlock()
	}
}

// RedisCache provides a distributed deduplication cache using Redis.
type RedisCache struct {
	client   *redis.Client
	cooldown time.Duration
}

// NewRedis creates a new Redis-backed dedup cache.
func NewRedis(redisURL, password string, useTLS bool, cooldownSeconds int) (*RedisCache, error) {
	opts := &redis.Options{
		Addr:     redisURL,
		Password: password,
	}

	if useTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &RedisCache{
		client:   client,
		cooldown: time.Duration(cooldownSeconds) * time.Second,
	}, nil
}

func (c *RedisCache) ShouldAlert(ctx context.Context, namespace, podName string) bool {
	key := fmt.Sprintf("dedup:%s", cacheKey(namespace, podName))

	// Attempt to set the key only if it doesn't exist (NX) with an expiration (EX)
	// This makes the check and the record-keeping atomic in Redis.
	ok, err := c.client.SetNX(ctx, key, time.Now().Unix(), c.cooldown).Result()
	if err != nil {
		log.Printf("[dedup] Redis error: %v. Falling back to allowing alert.", err)
		return true
	}

	return ok
}

func (c *RedisCache) Reset(ctx context.Context, namespace, podName string) {
	key := fmt.Sprintf("dedup:%s", cacheKey(namespace, podName))
	c.client.Del(ctx, key)
}

// FileCache provides a persistent deduplication cache using a local JSON file.
type FileCache struct {
	mu       sync.RWMutex
	entries  map[string]time.Time
	cooldown time.Duration
	path     string
}

// NewFile creates a new File-backed dedup cache.
func NewFile(path string, cooldownSeconds int) *FileCache {
	c := &FileCache{
		entries:  make(map[string]time.Time),
		cooldown: time.Duration(cooldownSeconds) * time.Second,
		path:     path,
	}

	c.load()
	go c.periodicSave()
	return c
}

func (c *FileCache) ShouldAlert(_ context.Context, namespace, podName string) bool {
	key := cacheKey(namespace, podName)
	c.mu.Lock()
	defer c.mu.Unlock()

	if lastSeen, exists := c.entries[key]; exists {
		if time.Since(lastSeen) < c.cooldown {
			return false
		}
	}

	c.entries[key] = time.Now()
	_ = c.save() // Best effort save on change
	return true
}

func (c *FileCache) Reset(_ context.Context, namespace, podName string) {
	key := cacheKey(namespace, podName)
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
	_ = c.save()
}

func (c *FileCache) load() {
	data, err := ioutil.ReadFile(c.path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[dedup] Failed to read persistence file: %v", err)
		}
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if err := json.Unmarshal(data, &c.entries); err != nil {
		log.Printf("[dedup] Failed to unmarshal persistence file: %v", err)
	} else {
		log.Printf("[dedup] Successfully loaded %d records from %s", len(c.entries), c.path)
	}
}

func (c *FileCache) save() error {
	c.mu.RLock()
	data, err := json.Marshal(c.entries)
	c.mu.RUnlock()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(c.path, data, 0644)
}

func (c *FileCache) periodicSave() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		if err := c.save(); err != nil {
			log.Printf("[dedup] Periodic save failed: %v", err)
		}
	}
}

func cacheKey(namespace, podName string) string {
	return fmt.Sprintf("%s/%s", namespace, podName)
}
