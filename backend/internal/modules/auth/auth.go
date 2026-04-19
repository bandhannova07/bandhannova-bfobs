package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
    "encoding/json"
    "net/http"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWT secret is now loaded from config
func getJWTSecret() []byte {
	return []byte(config.AppConfig.JWTSecret)
}

func getSupabaseJWTSecret() []byte {
	return []byte(config.AppConfig.SupabaseJWTSecret)
}

// AuthRequest represents signup/login request body
type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// hashPassword creates a salted SHA-256 hash
func hashPassword(password string) string {
	salt := make([]byte, 16)
	rand.Read(salt)
	hash := sha256.Sum256(append([]byte(password), salt...))
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(hash[:])
}

// verifyPassword checks a password against a stored hash
func verifyPassword(password, stored string) bool {
	parts := splitHash(stored)
	if len(parts) != 2 {
		return false
	}
	salt, _ := hex.DecodeString(parts[0])
	hash := sha256.Sum256(append([]byte(password), salt...))
	return hex.EncodeToString(hash[:]) == parts[1]
}

func splitHash(s string) []string {
	for i, c := range s {
		if c == ':' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

// generateJWT creates a signed JWT token with standardized claims
func generateJWT(userID, email, name, plan string, shardIdx int) (string, error) {
	claims := jwt.MapClaims{
		"user_id":   userID,
		"email":     email,
		"name":      name,
		"plan":      plan,
		"shard":     shardIdx,
		"status":    "active",
		"iss":       "bandhannova-auth",
		"aud":       "bandhannova-ecosystem",
		"exp":       time.Now().Add(7 * 24 * time.Hour).Unix(), // 7 days
		"iat":       time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := getJWTSecret()
	return token.SignedString(secret)
}

// Signup creates a new user account
func Signup(c *fiber.Ctx) error {
	if database.Router == nil {
		return c.Status(503).JSON(fiber.Map{"error": "BFOBS database not initialized"})
	}

	var req AuthRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Email and password are required"})
	}

	// Check if user already exists
	var existingID string
	err := database.Router.GetAuthDB().QueryRow("SELECT id FROM users WHERE email = ?", req.Email).Scan(&existingID)
	if err == nil {
		return c.Status(409).JSON(fiber.Map{"error": "User already exists"})
	}

	userID := uuid.New().String()
	now := time.Now().Unix()
	passwordHash := hashPassword(req.Password)
	displayName := req.Name
	if displayName == "" {
		displayName = req.Email
	}

	// Insert user into Auth shard
	_, err = database.Router.GetAuthDB().Exec(
		"INSERT INTO users (id, email, password_hash, display_name, plan, storage_used, created_at, updated_at) VALUES (?, ?, ?, ?, 'free', 0, ?, ?)",
		userID, req.Email, passwordHash, displayName, now, now,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Failed to create user: %v", err)})
	}

	// Determine user's shard
	shardIdx := database.Router.GetUserShardIndex(userID)

	// Generate JWT
	token, err := generateJWT(userID, req.Email, displayName, "free", shardIdx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate token"})
	}

	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"user": fiber.Map{
			"id":    userID,
			"email": req.Email,
			"name":  displayName,
			"plan":  "free",
			"shard": shardIdx,
		},
		"token": token,
	})
}

// Login authenticates a user and returns a JWT
func Login(c *fiber.Ctx) error {
	if database.Router == nil {
		return c.Status(503).JSON(fiber.Map{"error": "BFOBS database not initialized"})
	}

	var req AuthRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var userID, passwordHash, displayName, plan string
	err := database.Router.GetAuthDB().QueryRow(
		"SELECT id, password_hash, display_name, plan FROM users WHERE email = ?",
		req.Email,
	).Scan(&userID, &passwordHash, &displayName, &plan)

	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid email or password"})
	}

	if !verifyPassword(req.Password, passwordHash) {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid email or password"})
	}

	shardIdx := database.Router.GetUserShardIndex(userID)
	token, err := generateJWT(userID, req.Email, displayName, plan, shardIdx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate token"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"user": fiber.Map{
			"id":    userID,
			"email": req.Email,
			"name":  displayName,
			"plan":  plan,
			"shard": shardIdx,
		},
		"token": token,
	})
}

// GoogleLogin handles authentication via Google ID token
func GoogleLogin(c *fiber.Ctx) error {
    if database.Router == nil {
		return c.Status(503).JSON(fiber.Map{"error": "BFOBS database not initialized"})
	}

    var req struct {
        Token string `json:"token"`
    }
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

    if req.Token == "" {
        return c.Status(400).JSON(fiber.Map{"error": "Google token is required"})
    }

    // 1. Verify token with Google
    resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + req.Token)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "Failed to verify Google token"})
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return c.Status(401).JSON(fiber.Map{"error": "Invalid Google token"})
    }

    var googleClaims struct {
        Sub           string `json:"sub"`
        Email         string `json:"email"`
        Name          string `json:"name"`
        Picture       string `json:"picture"`
        EmailVerified string `json:"email_verified"`
        Aud           string `json:"aud"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&googleClaims); err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "Failed to parse Google response"})
    }

    // 2. Check if user exists, if not create
    var userID, displayName, plan string
	err = database.Router.GetAuthDB().QueryRow(
		"SELECT id, display_name, plan FROM users WHERE email = ?",
		googleClaims.Email,
	).Scan(&userID, &displayName, &plan)

    if err != nil {
        // User doesn't exist, create one
        userID = uuid.New().String()
        displayName = googleClaims.Name
        plan = "free"
        now := time.Now().Unix()

        // Insert user into Auth shard
        _, err = database.Router.GetAuthDB().Exec(
            "INSERT INTO users (id, email, password_hash, display_name, plan, storage_used, created_at, updated_at) VALUES (?, ?, ?, ?, 'free', 0, ?, ?)",
            userID, googleClaims.Email, "google-auth", displayName, now, now,
        )
        if err != nil {
            return c.Status(500).JSON(fiber.Map{"error": "Failed to create Google user"})
        }
    }

	shardIdx := database.Router.GetUserShardIndex(userID)
	token, err := generateJWT(userID, googleClaims.Email, displayName, plan, shardIdx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate token"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"user": fiber.Map{
			"id":    userID,
			"email": googleClaims.Email,
			"name":  displayName,
			"plan":  plan,
			"shard": shardIdx,
		},
		"token": token,
	})
}

// JWTRequired is middleware that validates JWT tokens (Local or Supabase)
func JWTRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" || len(authHeader) < 8 {
			return c.Status(401).JSON(fiber.Map{"error": "Authorization token required"})
		}

		tokenStr := authHeader[7:] // Remove "Bearer "

		// 1. Try Local JWT Secret
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return getJWTSecret(), nil
		})

		if err == nil && token.Valid {
			claims, ok := token.Claims.(jwt.MapClaims)
			if ok {
				c.Locals("user_id", claims["user_id"])
				c.Locals("email", claims["email"])
				c.Locals("plan", claims["plan"])
				c.Locals("shard", claims["shard"])
				return c.Next()
			}
		}

		// 2. Try Supabase JWT Secret (if configured)
		if config.AppConfig.SupabaseJWTSecret != "" {
			token, err = jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return getSupabaseJWTSecret(), nil
			})

			if err == nil && token.Valid {
				claims, ok := token.Claims.(jwt.MapClaims)
				if ok {
					// Map Supabase claims to BFOBS locals
					c.Locals("user_id", claims["sub"])
					c.Locals("email", claims["email"])
					c.Locals("plan", "free") // Default for Supabase users
					// Shard logic for external users
					userID := fmt.Sprintf("%v", claims["sub"])
					shardIdx := database.Router.GetUserShardIndex(userID)
					c.Locals("shard", shardIdx)
					return c.Next()
				}
			}
		}

		return c.Status(401).JSON(fiber.Map{"error": "Invalid or expired token"})
	}
}
