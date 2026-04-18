package api_proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/bandhannova/api-hunter/internal/security"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ProxyHandler is the core engine for API rotation and logging
func ProxyHandler(c *fiber.Ctx) error {
	provider := c.Params("provider")
	if provider == "" {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Provider alias required"})
	}

	// 1. Find a healthy key for this provider (Round Robin style via database)
	// We select a key that is active, not deleted, and has the earliest updated_at (least recently used)
	var keyID, cardID, encrypted, apiURL string
	var useURL int
	err := database.Router.GetGlobalManagerDB().QueryRow(`
		SELECT k.id, k.card_id, k.encrypted_value, k.api_url, k.use_url
		FROM managed_api_keys k
		JOIN api_cards c ON k.card_id = c.id
		WHERE (c.name = ? OR k.provider = ?) AND k.status = 'active' AND k.is_deleted = 0
		ORDER BY k.updated_at ASC
		LIMIT 1
	`, provider, provider).Scan(&keyID, &cardID, &encrypted, &apiURL, &useURL)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "No active keys found for provider: " + provider})
	}

	// 2. Decrypt Key
	apiKey, err := security.Decrypt(encrypted, config.AppConfig.BandhanNovaMasterKey)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Internal encryption error"})
	}

	// 3. Prepare Proxy Request
	targetURL := apiURL
	if useURL == 0 || targetURL == "" {
		// Fallback/Default logic if no custom URL is set
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "No endpoint URL configured for this API"})
	}

	// Append original path and query
	// Note: You might want to refine how you join paths here
	fullURL := targetURL + c.Params("*")
	if len(c.Request().URI().QueryString()) > 0 {
		fullURL += "?" + string(c.Request().URI().QueryString())
	}

	req, err := http.NewRequest(string(c.Request().Header.Method()), fullURL, bytes.NewReader(c.Body()))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to prepare request"})
	}

	// Copy headers (except Host and Auth which we will override)
	c.Request().Header.VisitAll(func(key, value []byte) {
		k := string(key)
		if k != "Host" && k != "Authorization" && k != "X-Bandhannova-Key" {
			req.Header.Set(k, string(value))
		}
	})

	// Inject Real API Key
	// Common formats: Bearer, Api-Key, or just the key
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("api-key", apiKey) // Support for Azure/Anthropic etc

	// 4. Execute Request
	client := &http.Client{Timeout: 30 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		logUsage(keyID, cardID, string(c.Request().Header.Method()), fullURL, 502, int(latency), c.IP())
		return c.Status(502).JSON(fiber.Map{"error": true, "message": "Provider unreachable: " + err.Error()})
	}
	defer resp.Body.Close()

	// 5. Update Key Usage (for rotation)
	database.Router.GetGlobalManagerDB().Exec("UPDATE managed_api_keys SET updated_at = ? WHERE id = ?", time.Now().Unix(), keyID)

	// 6. Log Usage
	logUsage(keyID, cardID, string(c.Request().Header.Method()), fullURL, resp.StatusCode, int(latency), c.IP())

	// 7. Stream Response back to client
	c.Status(resp.StatusCode)
	// Copy response headers
	for k, v := range resp.Header {
		c.Set(k, v[0])
	}
	
	body, _ := io.ReadAll(resp.Body)
	return c.Send(body)
}

func logUsage(keyID, cardID, method, path string, status, latency int, ip string) {
	id := uuid.New().String()
	_, _ = database.Router.GetGlobalManagerDB().Exec(`
		INSERT INTO api_usage_logs (id, key_id, card_id, method, path, status_code, latency_ms, ip_address, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, keyID, cardID, method, path, status, latency, ip, time.Now().Unix())
}

func ListLogs(c *fiber.Ctx) error {
	rows, err := database.Router.GetGlobalManagerDB().Query(`
		SELECT l.id, l.method, l.path, l.status_code, l.latency_ms, l.ip_address, l.timestamp, k.label, c.name
		FROM api_usage_logs l
		JOIN managed_api_keys k ON l.key_id = k.id
		JOIN api_cards c ON l.card_id = c.id
		ORDER BY l.timestamp DESC
		LIMIT 100
	`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to fetch logs"})
	}
	defer rows.Close()

	var logs []fiber.Map
	for rows.Next() {
		var id, method, path, ip, keyLabel, cardName string
		var status, latency int
		var ts int64
		rows.Scan(&id, &method, &path, &status, &latency, &ip, &ts, &keyLabel, &cardName)
		logs = append(logs, fiber.Map{
			"id": id,
			"method": method,
			"path": path,
			"status": status,
			"latency": latency,
			"ip": ip,
			"timestamp": ts,
			"key_label": keyLabel,
			"card_name": cardName,
		})
	}
	return c.JSON(fiber.Map{"success": true, "logs": logs})
}
