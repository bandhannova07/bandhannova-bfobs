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
	"github.com/bandhannova/api-hunter/internal/database"
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

// GenerateSessionToken creates an HMAC-SHA256 signed session token for any role
func GenerateSessionToken(identifier string, secret string) string {
	expiry := time.Now().Add(AdminTokenTTL).Unix()
	payload := fmt.Sprintf("bfobs-session:%s:%d", identifier, expiry)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("%s.%s", payload, signature)
}

// ValidateSessionToken verifies an HMAC-SHA256 signed session token
func ValidateSessionToken(token string, secret string) (string, bool) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", false
	}

	payload := parts[0]
	signature := parts[1]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return "", false
	}

	// Parse payload: bfobs-session:identifier:expiry
	pParts := strings.Split(payload, ":")
	if len(pParts) != 3 {
		return "", false
	}

	expiry, _ := strconv.ParseInt(pParts[2], 10, 64)
	if time.Now().Unix() > expiry {
		return "", false
	}

	return pParts[1], true
}

// AdminAuthRequired handles both Admin and Developer tokens
func AdminAuthRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := getClientIP(c)
		token := c.Get("X-Admin-Token")
		if token == "" {
			return c.Status(401).JSON(fiber.Map{"error": true, "message": "Authentication required"})
		}

		// 1. Try Admin (Master Key)
		if identifier, ok := ValidateSessionToken(token, config.AppConfig.BandhanNovaMasterKey); ok {
			if identifier == "admin" {
				c.Locals("admin_ip", ip)
				c.Locals("user_role", "admin")
				return c.Next()
			}
		}

		// 2. Try Developer (Fetch secret from global DB)
		parts := strings.SplitN(token, ".", 2)
		if len(parts) == 2 {
			pParts := strings.Split(parts[0], ":")
			if len(pParts) == 3 && pParts[0] == "bfobs-session" {
				slug := pParts[1]
				var secret string
				found := false
				
				for _, db := range database.Router.GetAllGlobalManagerDBs() {
					err := db.QueryRow("SELECT client_secret FROM oauth_clients oc JOIN managed_products p ON oc.product_id = p.id WHERE p.slug = ?", slug).Scan(&secret)
					if err == nil {
						found = true
						break
					}
				}

				if found {
					if ident, ok := ValidateSessionToken(token, secret); ok && ident == slug {
						c.Locals("admin_ip", ip)
						c.Locals("user_role", "developer")
						c.Locals("allowed_slug", slug)
						return c.Next()
					}
				}
			}
		}

		return c.Status(401).JSON(fiber.Map{"error": true, "message": "Invalid or expired session"})
	}
}

// Keep these for internal use
func GenerateAdminToken(masterKey string) string { return GenerateSessionToken("admin", masterKey) }
func ValidateAdminToken(token string, secret string) bool { _, ok := ValidateSessionToken(token, secret); return ok }

// ═══════════════════════════════════════════════
//  Rate Limiter & IP Helpers
// ═══════════════════════════════════════════════

func getClientIP(c *fiber.Ctx) string {
	if ip := c.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := c.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return c.IP()
}

func checkRateLimit(key string, limit int) bool {
	count, err := cache.Incr("ratelimit:"+key, time.Duration(RateLimitWindowSeconds)*time.Second)
	if err != nil {
		return true 
	}
	return int(count) <= limit
}

func recordLoginFailure(ip string) (remaining int, banned bool) {
	key := "failcount:" + ip
	count, _ := cache.Incr(key, 24*time.Hour)
	if int(count) >= AutoBanThreshold {
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
