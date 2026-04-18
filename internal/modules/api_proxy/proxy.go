package api_proxy

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/bandhannova/api-hunter/internal/security"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_2_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1",
}

func ProxyHandler(c *fiber.Ctx) error {
	provider := c.Params("provider")
	if provider == "" {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Provider is required"})
	}

	// Retry logic (Try up to 3 different keys if one fails)
	for attempt := 0; attempt < 3; attempt++ {
		var keyID, cardID, encrypted, cardEndpoint, platformType string
		
		err := database.Router.GetGlobalManagerDB().QueryRow(`
			SELECT k.id, k.card_id, k.encrypted_value, c.endpoint_url
			FROM managed_api_keys k
			JOIN api_cards c ON k.card_id = c.id
			WHERE (c.name = ? OR k.provider = ?) AND k.status = 'active' AND k.is_deleted = 0
			ORDER BY k.updated_at ASC
			LIMIT 1
		`, provider, provider).Scan(&keyID, &cardID, &encrypted, &cardEndpoint)

		if err != nil {
			break
		}

		apiKey, _ := security.Decrypt(encrypted, config.AppConfig.BandhanNovaMasterKey)
		fullURL := cardEndpoint + c.Params("*")
		if qs := string(c.Request().URI().QueryString()); qs != "" {
			fullURL += "?" + qs
		}

		req, err := http.NewRequest(c.Method(), fullURL, bytes.NewReader(c.Body()))
		if err != nil {
			continue
		}

		// Header processing
		c.Request().Header.VisitAll(func(key, value []byte) {
			k := string(key)
			if k != "Host" && k != "Authorization" && k != "User-Agent" {
				req.Header.Set(k, string(value))
			}
		})
		
		// Anonymization
		ua := userAgents[rand.Intn(len(userAgents))]
		req.Header.Set("User-Agent", ua)
		fakeIP := fmt.Sprintf("%d.%d.%d.%d", rand.Intn(254)+1, rand.Intn(254)+1, rand.Intn(254)+1, rand.Intn(254)+1)
		req.Header.Set("X-Forwarded-For", fakeIP)
		
		// Smart Architecture Detection
		if strings.Contains(cardEndpoint, "anthropic.com") {
			req.Header.Set("x-api-key", apiKey)
			req.Header.Set("anthropic-version", "2023-06-01")
		} else if strings.Contains(cardEndpoint, "googlegenesis") || strings.Contains(cardEndpoint, "generativelanguage") {
			// Google style often uses ?key= API_KEY
			if !strings.Contains(fullURL, "key=") {
				if strings.Contains(fullURL, "?") {
					fullURL += "&key=" + apiKey
				} else {
					fullURL += "?key=" + apiKey
				}
				// Re-create request with new URL if needed
				req.URL, _ = url.Parse(fullURL)
			}
		} else {
			// Default: OpenAI Compatible
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		client := &http.Client{Timeout: 45 * time.Second}
		start := time.Now()
		resp, err := client.Do(req)
		latency := time.Since(start).Milliseconds()

		if err != nil {
			database.Router.GetGlobalManagerDB().Exec("UPDATE managed_api_keys SET updated_at = ? WHERE id = ?", time.Now().Unix()+300, keyID)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 429 || resp.StatusCode == 401 {
			database.Router.GetGlobalManagerDB().Exec("UPDATE managed_api_keys SET updated_at = ? WHERE id = ?", time.Now().Unix()+1800, keyID)
			continue
		}

		// Success
		database.Router.GetGlobalManagerDB().Exec("UPDATE managed_api_keys SET updated_at = ? WHERE id = ?", time.Now().Unix(), keyID)
		logUsage(keyID, cardID, c.Method(), fullURL, resp.StatusCode, int(latency), c.IP())

		c.Status(resp.StatusCode)
		for k, v := range resp.Header {
			if len(v) > 0 { c.Set(k, v[0]) }
		}
		body, _ := io.ReadAll(resp.Body)
		return c.Send(body)
	}

	return c.Status(502).JSON(fiber.Map{"error": true, "message": "All keys exhausted or rate-limited"})
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
