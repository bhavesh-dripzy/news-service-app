package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/rs/zerolog/log"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(addr, password string, db int) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		PoolSize: 10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Info().Msg("Redis connection established")
	return &RedisCache{client: client}, nil
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get key %s: %w", key, err)
	}
	return val, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	var data []byte
	var err error

	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		data, err = json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
	}

	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *RedisCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	var data []byte
	var err error

	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		data, err = json.Marshal(value)
		if err != nil {
			return false, fmt.Errorf("failed to marshal value: %w", err)
		}
	}

	return c.client.SetNX(ctx, key, data, ttl).Result()
}

func (c *RedisCache) Del(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}
	return result > 0, nil
}

func (c *RedisCache) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	return c.client.ZAdd(ctx, key, members...).Err()
}

// SAdd adds members to a set
func (c *RedisCache) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return c.client.SAdd(ctx, key, members...).Err()
}

// SMembers returns all members of a set
func (c *RedisCache) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.client.SMembers(ctx, key).Result()
}

// ZRangeByScore returns members with scores in the given range
func (c *RedisCache) ZRangeByScore(ctx context.Context, key string, min, max float64, limit int64) ([]string, error) {
	query := &redis.ZRangeBy{
		Min:    fmt.Sprintf("%f", min),
		Max:    fmt.Sprintf("%f", max),
		Offset: 0,
		Count:  limit,
	}
	return c.client.ZRangeByScore(ctx, key, query).Result()
}

func (c *RedisCache) ZRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	return c.client.ZRangeWithScores(ctx, key, start, stop).Result()
}

func (c *RedisCache) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	return c.client.ZRevRangeWithScores(ctx, key, start, stop).Result()
}

func (c *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

func (c *RedisCache) GeoAdd(ctx context.Context, key string, longitude, latitude float64, member interface{}) error {
	return c.client.GeoAdd(ctx, key, &redis.GeoLocation{
		Longitude: longitude,
		Latitude:  latitude,
		Name:      fmt.Sprintf("%v", member),
	}).Err()
}

func (c *RedisCache) GeoRadius(ctx context.Context, key string, longitude, latitude float64, query *redis.GeoRadiusQuery) ([]redis.GeoLocation, error) {
	return c.client.GeoRadius(ctx, key, longitude, latitude, query).Result()
}

// Cache stampede protection
func (c *RedisCache) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) ([]byte, error) {
	// Try to get from cache first
	if data, err := c.Get(ctx, key); err == nil {
		return data, nil
	}

	// Create a lock key
	lockKey := fmt.Sprintf("lock:%s", key)
	
	// Try to acquire lock
	acquired, err := c.SetNX(ctx, lockKey, "1", 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !acquired {
		// Wait for the other process to finish
		for i := 0; i < 50; i++ { // Wait up to 5 seconds
			time.Sleep(100 * time.Millisecond)
			if data, err := c.Get(ctx, key); err == nil {
				return data, nil
			}
		}
		return nil, fmt.Errorf("timeout waiting for cache update")
	}

	// We have the lock, generate the value
	defer c.Del(ctx, lockKey)

	value, err := fn()
	if err != nil {
		return nil, fmt.Errorf("failed to generate value: %w", err)
	}

	// Store in cache
	if err := c.Set(ctx, key, value, ttl); err != nil {
		return nil, fmt.Errorf("failed to store value in cache: %w", err)
	}

	// Return the generated value
	switch v := value.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		return json.Marshal(value)
	}
}

var ErrKeyNotFound = fmt.Errorf("key not found")

