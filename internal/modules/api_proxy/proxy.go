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

// ProxyHandler handles the API request redirection with key rotation
func ProxyHandler(c *fiber.Ctx) error {
	provider := c.Params("provider")
	if provider == "" {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Provider is required"})
	}

	// 1. Fetch Least Recently Used key for this provider
	var keyID, cardID, encrypted, apiURL string
	var useURL int
	query := `
		SELECT k.id, k.card_id, k.encrypted_value, k.api_url, k.use_url
		FROM managed_api_keys k
		JOIN api_cards c ON k.card_id = c.id
		WHERE (c.name = ? OR k.provider = ?) AND k.status = 'active' AND k.is_deleted = 0
		ORDER BY k.updated_at ASC
		LIMIT 1
	`
	err := database.Router.GetGlobalManagerDB().QueryRow(query, provider, provider).Scan(
		&keyID, &cardID, &encrypted, &apiURL, &useURL,
	)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "No active API keys for: " + provider})
	}

	// 2. Decrypt the API Key
	apiKey, err := security.Decrypt(encrypted, config.AppConfig.BandhanNovaMasterKey)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Security layer error"})
	}

	// 3. Construct Target URL
	if useURL == 0 || apiURL == "" {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Target endpoint not configured"})
	}

	fullURL := apiURL + c.Params("*")
	queryString := string(c.Request().URI().QueryString())
	if queryString != "" {
		fullURL += "?" + queryString
	}

	// 4. Create HTTP Request
	req, err := http.NewRequest(c.Method(), fullURL, bytes.NewReader(c.Body()))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to initialize request"})
	}

	// Copy headers from original request
	c.Request().Header.VisitAll(func(key, value []byte) {
		k := string(key)
		if k != "Host" && k != "Authorization" && k != "X-Bandhannova-Key" {
			req.Header.Set(k, string(value))
		}
	})

	// Inject Authentication
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("api-key", apiKey)

	// 5. Execute with Timeout
	client := &http.Client{Timeout: 60 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		logUsage(keyID, cardID, c.Method(), fullURL, 502, int(latency), c.IP())
		return c.Status(502).JSON(fiber.Map{"error": true, "message": "Upstream timeout or error"})
	}
	defer resp.Body.Close()

	// 6. Update Usage Timestamp (Rotation Logic)
	_, _ = database.Router.GetGlobalManagerDB().Exec(
		"UPDATE managed_api_keys SET updated_at = ? WHERE id = ?",
		time.Now().Unix(), keyID,
	)

	// 7. Record Metrics
	logUsage(keyID, cardID, c.Method(), fullURL, resp.StatusCode, int(latency), c.IP())

	// 8. Stream Response back
	c.Status(resp.StatusCode)
	for k, v := range resp.Header {
		if len(v) > 0 {
			c.Set(k, v[0])
		}
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
	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return c.Status(503).JSON(fiber.Map{"error": true, "message": "Database disconnected"})
	}

	rows, err := database.Router.GetGlobalManagerDB().Query(`
		SELECT l.id, l.method, l.path, l.status_code, l.latency_ms, l.ip_address, l.timestamp, k.label, c.name
		FROM api_usage_logs l
		JOIN managed_api_keys k ON l.key_id = k.id
		JOIN api_cards c ON l.card_id = c.id
		ORDER BY l.timestamp DESC
		LIMIT 100
	`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Logs unavailable"})
	}
	defer rows.Close()

	var logs []fiber.Map
	for rows.Next() {
		var id, method, path, ip, keyLabel, cardName string
		var status, latency int
		var ts int64
		if err := rows.Scan(&id, &method, &path, &status, &latency, &ip, &ts, &keyLabel, &cardName); err == nil {
			logs = append(logs, fiber.Map{
				"id": id, "method": method, "path": path, "status": status,
				"latency": latency, "ip": ip, "timestamp": ts,
				"key_label": keyLabel, "card_name": cardName,
			})
		}
	}
	return c.JSON(fiber.Map{"success": true, "logs": logs})
}
