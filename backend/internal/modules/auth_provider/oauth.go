package auth_provider

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type OAuthClient struct {
	ClientID     string
	ClientSecret string
	RedirectURIs []string
}

// GenerateCode creates a random authorization code
func GenerateCode() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// Authorize handles the initial OAuth request and shows consent
func Authorize(c *fiber.Ctx) error {
	clientID := c.Query("client_id")
	redirectURI := c.Query("redirect_uri")
	state := c.Query("state")

	if clientID == "" || redirectURI == "" {
		return c.Status(400).SendString("Invalid OAuth Request")
	}

	// Verify Client
	var dbClientID string
	err := database.Router.GetGlobalManagerDB().QueryRow("SELECT client_id FROM oauth_clients WHERE client_id = ?", clientID).Scan(&dbClientID)
	if err != nil {
		return c.Status(401).SendString("Client not registered in BandhanNova Ecosystem")
	}

	// In a real flow, here we would check if user is logged in
	// For now, let's assume we redirect to a login/consent UI
	// After login, we would generate a code
	
	return c.JSON(fiber.Map{
		"message": "BandhanNova Auth Portal (Consent Required)",
		"client_id": clientID,
		"redirect_uri": redirectURI,
		"state": state,
		"action": "Redirecting to UI...",
	})
}

type TokenRequest struct {
	GrantType    string `json:"grant_type" form:"grant_type"`
	Code         string `json:"code" form:"code"`
	ClientID     string `json:"client_id" form:"client_id"`
	ClientSecret string `json:"client_secret" form:"client_secret"`
	RedirectURI  string `json:"redirect_uri" form:"redirect_uri"`
}

// Token exchanges authorization code for JWT
func Token(c *fiber.Ctx) error {
	var req TokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid_request"})
	}

	// Verify Client Credentials
	var dbSecret string
	err := database.Router.GetGlobalManagerDB().QueryRow("SELECT client_secret FROM oauth_clients WHERE client_id = ?", req.ClientID).Scan(&dbSecret)
	if err != nil || dbSecret != req.ClientSecret {
		return c.Status(401).JSON(fiber.Map{"error": "invalid_client"})
	}

	// Issue JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "https://www.bandhannova.in",
		"sub": "user_12345", // This should be the real User ID from Shard
		"exp": time.Now().Add(time.Hour * 24).Unix(),
		"iat": time.Now().Unix(),
		"name": "Bandhan User",
		"email": "user@bandhannova.in",
	})

	tokenString, err := token.SignedString([]byte(config.AppConfig.JWTSecret))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "server_error"})
	}

	return c.JSON(fiber.Map{
		"access_token": tokenString,
		"token_type":   "Bearer",
		"expires_in":   86400,
		"id_token":     tokenString, // Simplified for now
	})
}

// UserInfo returns protected user data
func UserInfo(c *fiber.Ctx) error {
	// In production, verify JWT from Authorization header
	return c.JSON(fiber.Map{
		"sub": "user_12345",
		"name": "Bandhan User",
		"email": "user@bandhannova.in",
		"ecosystem": "BandhanNova",
		"verified": true,
	})
}
