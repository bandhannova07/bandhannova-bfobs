package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bandhannova/api-hunter/internal/cache"
	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/gofiber/fiber/v2"
)

// ═══════════════════════════════════════════════
//  HMAC Session Token System
// ═══════════════════════════════════════════════

const (
	AdminTokenTTL         = 24 * time.Hour
	LoginRateLimit        = 5          // per minute
	AdminRateLimit        = 500        // per minute (Increased for smooth dashboard experience)
	AutoBanThreshold      = 20         // total failed logins
	AutoBanDuration       = 1 * time.Hour
	RateLimitWindowSeconds = 60
)

// GenerateAdminToken creates an HMAC-SHA256 signed session token
func GenerateAdminToken(masterKey string) string {
	expiry := time.Now().Add(AdminTokenTTL).Unix()
	payload := fmt.Sprintf("bfobs-admin:%d", expiry)

	mac := hmac.New(sha256.New, []byte(masterKey))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("%s.%s", payload, signature)
}

// ValidateAdminToken verifies an HMAC-SHA256 signed session token
func ValidateAdminToken(token string, masterKey string) bool {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return false
	}

	payload := parts[0]
	signature := parts[1]

	// Verify signature
	mac := hmac.New(sha256.New, []byte(masterKey))
	mac.Write([]byte(payload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return false
	}

	// Check expiry
	colonIdx := strings.LastIndex(payload, ":")
	if colonIdx == -1 {
		return false
	}
	expiryStr := payload[colonIdx+1:]
	expiry, err := strconv.ParseInt(expiryStr, 10, 64)
	if err != nil {
		return false
	}

	return time.Now().Unix() < expiry
}

// ═══════════════════════════════════════════════
//  Rate Limiter (Redis-based, Per-IP)
// ═══════════════════════════════════════════════

func getClientIP(c *fiber.Ctx) string {
	// Check common proxy headers
	if ip := c.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := c.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return c.IP()
}

func checkRateLimit(key string, limit int) bool {
	// Use Redis Incr for sliding window (simplified to fixed window for performance)
	count, err := cache.Incr("ratelimit:"+key, time.Duration(RateLimitWindowSeconds)*time.Second)
	if err != nil {
		return true // Fallback to allow if Redis is down
	}
	return int(count) <= limit
}

func recordLoginFailure(ip string) (remaining int, banned bool) {
	key := "failcount:" + ip
	count, _ := cache.Incr(key, 24*time.Hour) // Keep count for 24h

	if int(count) >= AutoBanThreshold {
		// Set ban in Redis
		_ = cache.Set("ban:"+ip, time.Now().Add(AutoBanDuration).Unix(), AutoBanDuration)
		return 0, true
	}

	return AutoBanThreshold - int(count), false
}

func resetLoginFailures(ip string) {
	_ = cache.Del("failcount:" + ip)
	_ = cache.Del("ban:" + ip)
}

func getBanRemaining(ip string) int64 {
	var banUntil int64
	exists, _ := cache.Get("ban:"+ip, &banUntil)
	if !exists {
		return 0
	}

	remaining := banUntil - time.Now().Unix()
	if remaining < 0 {
		_ = cache.Del("ban:" + ip)
		return 0
	}
	return remaining
}


// ═══════════════════════════════════════════════
//  Middleware Handlers
// ═══════════════════════════════════════════════

// AdminAuthRequired validates HMAC session token from X-Admin-Token header
func AdminAuthRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := getClientIP(c)

		// Check auto-ban
		if banRemaining := getBanRemaining(ip); banRemaining > 0 {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   true,
				"message": fmt.Sprintf("IP banned for %d seconds due to excessive failed attempts", banRemaining),
			})
		}

		// Rate limit admin routes
		if !checkRateLimit("admin:"+ip, AdminRateLimit) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   true,
				"message": "Rate limit exceeded. Maximum 30 requests per minute.",
			})
		}

		token := c.Get("X-Admin-Token")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   true,
				"message": "Admin authentication required",
			})
		}

		if !ValidateAdminToken(token, config.AppConfig.BandhanNovaMasterKey) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   true,
				"message": "Invalid or expired admin session",
			})
		}

		c.Locals("admin_ip", ip)
		return c.Next()
	}
}

// AdminLoginRateLimiter applies strict rate limiting to login endpoint
func AdminLoginRateLimiter() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := getClientIP(c)

		// Check auto-ban first
		if banRemaining := getBanRemaining(ip); banRemaining > 0 {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       true,
				"message":     "Too many failed attempts. IP temporarily banned.",
				"ban_seconds": banRemaining,
			})
		}

		// Rate limit login attempts
		if !checkRateLimit("login:"+ip, LoginRateLimit) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   true,
				"message": "Too many login attempts. Wait 60 seconds.",
			})
		}

		c.Locals("client_ip", ip)
		return c.Next()
	}
}

// RecordFailedLogin records a failed login and returns ban info
func RecordFailedLogin(ip string) (int, bool) {
	return recordLoginFailure(ip)
}

// ResetFailedLogins clears failure count for an IP on successful login
func ResetFailedLogins(ip string) {
	resetLoginFailures(ip)
}
