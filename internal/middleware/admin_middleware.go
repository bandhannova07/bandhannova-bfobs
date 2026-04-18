package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/gofiber/fiber/v2"
)

// ═══════════════════════════════════════════════
//  HMAC Session Token System
// ═══════════════════════════════════════════════

const (
	AdminTokenTTL         = 24 * time.Hour
	LoginRateLimit        = 5          // per minute
	AdminRateLimit        = 30         // per minute
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
//  Rate Limiter (Sliding Window, Per-IP)
// ═══════════════════════════════════════════════

type rateLimitEntry struct {
	timestamps []int64
	banUntil   int64
	failCount  int
}

var (
	rateLimitStore = make(map[string]*rateLimitEntry)
	rateLimitMu    sync.Mutex
)

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

func checkRateLimit(ip string, limit int) bool {
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()

	entry, exists := rateLimitStore[ip]
	if !exists {
		entry = &rateLimitEntry{}
		rateLimitStore[ip] = entry
	}

	now := time.Now().Unix()

	// Check auto-ban
	if entry.banUntil > now {
		return false
	}

	// Clean old timestamps outside window
	windowStart := now - RateLimitWindowSeconds
	var valid []int64
	for _, ts := range entry.timestamps {
		if ts > windowStart {
			valid = append(valid, ts)
		}
	}
	entry.timestamps = valid

	// Check limit
	if len(entry.timestamps) >= limit {
		return false
	}

	entry.timestamps = append(entry.timestamps, now)
	return true
}

func recordLoginFailure(ip string) (remaining int, banned bool) {
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()

	entry, exists := rateLimitStore[ip]
	if !exists {
		entry = &rateLimitEntry{}
		rateLimitStore[ip] = entry
	}

	entry.failCount++

	if entry.failCount >= AutoBanThreshold {
		entry.banUntil = time.Now().Add(AutoBanDuration).Unix()
		return 0, true
	}

	return AutoBanThreshold - entry.failCount, false
}

func resetLoginFailures(ip string) {
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()

	if entry, exists := rateLimitStore[ip]; exists {
		entry.failCount = 0
		entry.banUntil = 0
	}
}

func getBanRemaining(ip string) int64 {
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()

	entry, exists := rateLimitStore[ip]
	if !exists {
		return 0
	}

	remaining := entry.banUntil - time.Now().Unix()
	if remaining < 0 {
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
