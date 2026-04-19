package cache

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client
var ctx = context.Background()

// InitRedis connects to Upstash Redis or any Redis instance
func InitRedis(url string) error {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return err
	}

	RedisClient = redis.NewClient(opt)

	// Test connection
	_, err = RedisClient.Ping(ctx).Result()
	if err != nil {
		return err
	}

	log.Println("⚡ Redis (Upstash) connected successfully")
	return nil
}

// SetCache stores a value with an expiration
func SetCache(key string, value interface{}, expiration time.Duration) error {
	return RedisClient.Set(ctx, key, value, expiration).Err()
}

// GetCache retrieves a value
func GetCache(key string) (string, error) {
	return RedisClient.Get(ctx, key).Result()
}

// DeleteCache removes a key
func DeleteCache(key string) error {
	return RedisClient.Del(ctx, key).Err()
}
