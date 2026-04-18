package middleware

import (
	"fmt"
	"strings"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// AuthRequired ensures that only authorized BandhanNova apps OR authenticated users can access the gateway
func AuthRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Check for Master Key (App-to-App)
		apiKey := c.Get("X-BandhanNova-Key")
		if apiKey != "" {
			if apiKey == config.AppConfig.BandhanNovaMasterKey {
				return c.Next()
			}
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   true,
				"message": "Invalid BandhanNova Key",
			})
		}

		// 2. Check for Bearer Token (User-to-App)
		authHeader := c.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr := authHeader[7:]

			// Try Local Secret first
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return []byte(config.AppConfig.JWTSecret), nil
			})

			if err == nil && token.Valid {
				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					c.Locals("user_id", claims["user_id"])
					c.Locals("email", claims["email"])
					c.Locals("auth_type", "jwt_local")
					return c.Next()
				}
			}

			// Try Supabase Secret if configured
			if config.AppConfig.SupabaseJWTSecret != "" {
				token, err = jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
					if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, fmt.Errorf("unexpected signing method")
					}
					return []byte(config.AppConfig.SupabaseJWTSecret), nil
				})

				if err == nil && token.Valid {
					if claims, ok := token.Claims.(jwt.MapClaims); ok {
						c.Locals("user_id", claims["sub"])
						c.Locals("email", claims["email"])
						c.Locals("auth_type", "jwt_supabase")
						return c.Next()
					}
				}
			}

			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   true,
				"message": "Invalid or expired authorization token",
			})
		}

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   true,
			"message": "Authorization required (Master Key or Bearer Token)",
		})
	}
}
