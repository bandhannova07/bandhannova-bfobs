package cache

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/redis/go-redis/v9"
)

var (
	RedisClient *redis.Client
	ctx         = context.Background()
)

// InitRedis initializes the Redis client
func InitRedis() {
	if config.AppConfig.RedisURL == "" {
		log.Println("⚠️ Redis URL not found in config. Caching disabled.")
		return
	}

	opt, err := redis.ParseURL(config.AppConfig.RedisURL)
	if err != nil {
		log.Printf("❌ Failed to parse Redis URL: %v", err)
		return
	}

	RedisClient = redis.NewClient(opt)

	// Test Connection
	status := RedisClient.Ping(ctx)
	if status.Err() != nil {
		log.Printf("❌ Redis Connection Failed: %v", status.Err())
		RedisClient = nil
		return
	}

	log.Println("💓 Redis Connection Established (Upstash / Serverless)")
}

// Set stores a value in Redis with TTL
func Set(key string, value interface{}, expiration time.Duration) error {
	if RedisClient == nil {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return RedisClient.Set(ctx, key, data, expiration).Err()
}

// Get retrieves a value from Redis and unmarshals it
func Get(key string, target interface{}) (bool, error) {
	if RedisClient == nil {
		return false, nil
	}

	val, err := RedisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		return false, err
	}

	err = json.Unmarshal([]byte(val), target)
	if err != nil {
		return false, err
	}

	return true, nil
}

// Del removes a key from Redis
func Del(key string) error {
	if RedisClient == nil {
		return nil
	}
	return RedisClient.Del(ctx, key).Err()
}

// Incr increments a key for rate limiting
func Incr(key string, expiration time.Duration) (int64, error) {
	if RedisClient == nil {
		return 0, nil
	}

	pipe := RedisClient.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, expiration)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return incr.Val(), nil
}
