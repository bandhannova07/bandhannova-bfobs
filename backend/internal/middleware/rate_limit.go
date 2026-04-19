package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/bandhannova/api-hunter/internal/ex-db/cache"
	"github.com/gofiber/fiber/v2"
)

var ctx = context.Background()

// RedisRateLimiter limits requests per IP or User ID using Redis
func RedisRateLimiter(maxRequests int, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// If Redis is not initialized, skip rate limiting
		if cache.RedisClient == nil {
			return c.Next()
		}

		// Use IP address as the key
		key := fmt.Sprintf("rate_limit:%s", c.IP())
		
		// Increment the counter in Redis
		count, err := cache.RedisClient.Incr(ctx, key).Result()
		if err != nil {
			return c.Next()
		}

		// Set expiration if it's a new key
		if count == 1 {
			cache.RedisClient.Expire(ctx, key, window)
		}

		// Check if limit exceeded
		if int(count) > maxRequests {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   true,
				"message": "Too many requests. Please slow down.",
				"retry_after": window.Seconds(),
			})
		}

		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", maxRequests-int(count)))

		return c.Next()
	}
}
