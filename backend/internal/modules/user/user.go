package user

import (
	"fmt"
	"time"

	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetProfile returns the authenticated user's profile
func GetProfile(c *fiber.Ctx) error {
	if database.Router == nil {
		return c.Status(503).JSON(fiber.Map{"error": "BFOBS database not initialized"})
	}

	userID := c.Locals("user_id").(string)

	var email, displayName, plan, avatarURL string
	var storageUsed int64
	var createdAt int64
	err := database.Router.GetAuthDB().QueryRow(
		"SELECT email, display_name, plan, avatar_url, storage_used, created_at FROM users WHERE id = ?",
		userID,
	).Scan(&email, &displayName, &plan, &avatarURL, &storageUsed, &createdAt)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	// Get plan limits
	planLimits := database.GetPlan(plan)

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":           userID,
			"email":        email,
			"display_name": displayName,
			"avatar_url":   avatarURL,
			"plan":         plan,
			"storage_used": storageUsed,
			"created_at":   createdAt,
		},
		"limits": fiber.Map{
			"max_chat_messages":  planLimits.MaxChatMessages,
			"max_conversations":  planLimits.MaxConversations,
			"max_storage_bytes":  planLimits.MaxStorageBytes,
			"max_saved_items":    planLimits.MaxSavedItems,
		},
	})
}

// UpdateProfile updates the authenticated user's profile
func UpdateProfile(c *fiber.Ctx) error {
	if database.Router == nil {
		return c.Status(503).JSON(fiber.Map{"error": "BFOBS database not initialized"})
	}

	userID := c.Locals("user_id").(string)

	var body struct {
		DisplayName string `json:"display_name"`
		AvatarURL   string `json:"avatar_url"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	now := time.Now().Unix()
	_, err := database.Router.GetAuthDB().Exec(
		"UPDATE users SET display_name = ?, avatar_url = ?, updated_at = ? WHERE id = ?",
		body.DisplayName, body.AvatarURL, now, userID,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Failed to update profile: %v", err)})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Profile updated"})
}

// SaveChatMessage saves a chat message to the user's shard
func SaveChatMessage(c *fiber.Ctx) error {
	if database.Router == nil {
		return c.Status(503).JSON(fiber.Map{"error": "BFOBS database not initialized"})
	}

	userID := c.Locals("user_id").(string)
	plan := c.Locals("plan").(string)

	var body struct {
		ConversationID string `json:"conversation_id"`
		Role           string `json:"role"`
		Content        string `json:"content"`
		Model          string `json:"model"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Check plan limits
	planLimits := database.GetPlan(plan)
	db := database.Router.GetUserDB(userID)

	var msgCount int
	err := db.QueryRow("SELECT COUNT(*) FROM chat_history WHERE user_id = ?", userID).Scan(&msgCount)
	if err == nil && msgCount >= planLimits.MaxChatMessages {
		return c.Status(429).JSON(fiber.Map{
			"error":   "Chat storage limit reached",
			"plan":    plan,
			"limit":   planLimits.MaxChatMessages,
			"current": msgCount,
			"upgrade": "Upgrade to Pro/Ultra/Maxx for more storage",
		})
	}

	msgID := uuid.New().String()
	now := time.Now().Unix()

	_, err = db.Exec(
		"INSERT INTO chat_history (id, user_id, conversation_id, role, content, model, timestamp) VALUES (?, ?, ?, ?, ?, ?, ?)",
		msgID, userID, body.ConversationID, body.Role, body.Content, body.Model, now,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Failed to save message: %v", err)})
	}

	return c.Status(201).JSON(fiber.Map{"success": true, "message_id": msgID})
}

// GetChatHistory returns chat history for the authenticated user
func GetChatHistory(c *fiber.Ctx) error {
	if database.Router == nil {
		return c.Status(503).JSON(fiber.Map{"error": "BFOBS database not initialized"})
	}

	userID := c.Locals("user_id").(string)
	conversationID := c.Query("conversation_id", "")

	db := database.Router.GetUserDB(userID)

	var query string
	var args []interface{}

	if conversationID != "" {
		query = "SELECT id, conversation_id, role, content, model, timestamp FROM chat_history WHERE user_id = ? AND conversation_id = ? ORDER BY timestamp ASC LIMIT 100"
		args = []interface{}{userID, conversationID}
	} else {
		query = "SELECT id, conversation_id, role, content, model, timestamp FROM chat_history WHERE user_id = ? ORDER BY timestamp DESC LIMIT 50"
		args = []interface{}{userID}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Failed to fetch history: %v", err)})
	}
	defer rows.Close()

	var messages []fiber.Map
	for rows.Next() {
		var id, convID, role, content, model string
		var timestamp int64
		if err := rows.Scan(&id, &convID, &role, &content, &model, &timestamp); err != nil {
			continue
		}
		messages = append(messages, fiber.Map{
			"id":              id,
			"conversation_id": convID,
			"role":            role,
			"content":         content,
			"model":           model,
			"timestamp":       timestamp,
		})
	}

	if messages == nil {
		messages = []fiber.Map{}
	}

	return c.JSON(fiber.Map{
		"messages": messages,
		"count":    len(messages),
	})
}
