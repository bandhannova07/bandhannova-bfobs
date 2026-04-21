package api_proxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bandhannova/api-hunter/internal/cache"
	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/bandhannova/api-hunter/internal/security"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"sync"
)

type CachedCard struct {
	ID              string `json:"id"`
	EndpointURL     string `json:"endpoint_url"`
	LimitRPS        int    `json:"limit_rps"`
	LimitRPM        int    `json:"limit_rpm"`
	LimitRPH        int    `json:"limit_rph"`
	LimitRPD        int    `json:"limit_rpd"`
	LimitRPMonth    int    `json:"limit_rpmonth"`
	LimitConcurrent int    `json:"limit_concurrent"`
}

type CachedKey struct {
	ID             string `json:"id"`
	EncryptedValue string `json:"encrypted_value"`
}


var (
	userAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_2_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1",
	}
	concurrentRequests = make(map[string]int)
	concurrentMutex    sync.Mutex
)

type CardLimits struct {
	ID              string
	LimitRPS        int
	LimitRPM        int
	LimitRPH        int
	LimitRPD        int
	LimitRPMonth    int
	LimitConcurrent int
}

func checkRateLimits(card CardLimits) error {
	concurrentMutex.Lock()
	current := concurrentRequests[card.ID]
	if card.LimitConcurrent > 0 && current >= card.LimitConcurrent {
		concurrentMutex.Unlock()
		return fmt.Errorf("concurrent request limit reached (%d)", card.LimitConcurrent)
	}
	concurrentRequests[card.ID]++
	concurrentMutex.Unlock()

	// ─── Redis-based Rate Limiting ────────────────────────
	now := time.Now().Unix()
	
	windows := []struct {
		Name   string
		Window int64
		Limit  int
	}{
		{"sec", 1, card.LimitRPS},
		{"min", 60, card.LimitRPM},
		{"hour", 3600, card.LimitRPH},
		{"day", 86400, card.LimitRPD},
		{"month", 2592000, card.LimitRPMonth},
	}

	for _, w := range windows {
		if w.Limit > 0 {
			key := fmt.Sprintf("rl:%s:%s:%d", card.ID, w.Name, now/w.Window)
			count, err := cache.Incr(key, time.Duration(w.Window)*time.Second*2) // TTL is 2x window for safety
			if err == nil && int(count) > w.Limit {
				decrementConcurrent(card.ID)
				return fmt.Errorf("rate limit exceeded: %d requests per %s", w.Limit, w.Name)
			}
		}
	}

	return nil
}


func decrementConcurrent(cardID string) {
	concurrentMutex.Lock()
	if concurrentRequests[cardID] > 0 {
		concurrentRequests[cardID]--
	}
	concurrentMutex.Unlock()
}

// ═══════════════════════════════════════════════
//  Core Proxy Execution (Single Source of Truth)
// ═══════════════════════════════════════════════

// executeProxy is the unified proxy logic used by both ProxyHandler and EcosystemProxyHandler.
// It handles key rotation, rate limiting, URL building, header anonymization, and response forwarding.
func executeProxy(c *fiber.Ctx, cardID string) error {
	// 1. Try to get Keys and Card from Cache
	var keys []CachedKey
	var card CachedCard
	cacheKey := "card_data:" + cardID
	
	exists, _ := cache.Get(cacheKey, &card)
	keysExists, _ := cache.Get("card_keys:"+cardID, &keys)

	if !exists || !keysExists || len(keys) == 0 {
		// Cache Miss - Fetch from DB
		log.Printf("📥 Cache Miss: card_id=%s. Fetching from DB...", cardID)
		
		err := database.Router.GetGlobalManagerDB().QueryRow(`
			SELECT id, endpoint_url, limit_rps, limit_rpm, limit_rph, limit_rpd, limit_rpmonth, limit_concurrent
			FROM api_cards WHERE id = ? AND is_deleted = 0
		`, cardID).Scan(
			&card.ID, &card.EndpointURL, 
			&card.LimitRPS, &card.LimitRPM, &card.LimitRPH, &card.LimitRPD, &card.LimitRPMonth, &card.LimitConcurrent,
		)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": true, "message": "Card not found"})
		}

		rows, err := database.Router.GetGlobalManagerDB().Query(`
			SELECT id, encrypted_value FROM managed_api_keys 
			WHERE card_id = ? AND status = 'active' AND is_deleted = 0
		`, cardID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var k CachedKey
				rows.Scan(&k.ID, &k.EncryptedValue)
				keys = append(keys, k)
			}
		}

		// Store in Cache (10 minutes)
		_ = cache.Set(cacheKey, card, 10*time.Minute)
		_ = cache.Set("card_keys:"+cardID, keys, 10*time.Minute)
	}

	if len(keys) == 0 {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "No active keys found for this provider"})
	}

	// Retry logic (Try up to 3 different keys if one fails)
	// We'll pick a random key from our cached list for load balancing
	for attempt := 0; attempt < 3 && attempt < len(keys); attempt++ {
		// Randomly select a key
		idx := rand.Intn(len(keys))
		selectedKey := keys[idx]
		
		limits := CardLimits{
			ID: card.ID,
			LimitRPS: card.LimitRPS,
			LimitRPM: card.LimitRPM,
			LimitRPH: card.LimitRPH,
			LimitRPD: card.LimitRPD,
			LimitRPMonth: card.LimitRPMonth,
			LimitConcurrent: card.LimitConcurrent,
		}

		// Enforce Rate Limits
		if err := checkRateLimits(limits); err != nil {
			return c.Status(429).JSON(fiber.Map{"error": true, "message": err.Error()})
		}
		
		apiKey, _ := security.Decrypt(selectedKey.EncryptedValue, config.AppConfig.BandhanNovaMasterKey)
		
		// URL-based key rotation support
		targetBase := card.EndpointURL
		actualKey := apiKey


		if strings.HasPrefix(apiKey, "http://") || strings.HasPrefix(apiKey, "https://") {
			// Split by | to support URL|KEY format
			parts := strings.Split(apiKey, "|")
			targetBase = parts[0]
			if len(parts) > 1 {
				actualKey = parts[1]
			} else {
				actualKey = "" // No key needed for this URL
			}
			// Append the card's defined endpoint path to the rotated node URL
			targetBase = targetBase + card.EndpointURL
		}

		fullURL := targetBase + c.Params("*")
		if qs := string(c.Request().URI().QueryString()); qs != "" {
			fullURL += "?" + qs
		}

		req, err := http.NewRequest(c.Method(), fullURL, bytes.NewReader(c.Body()))
		if err != nil {
			decrementConcurrent(cardID)
			continue
		}

		// Copy headers (excluding sensitive ones we'll override)
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
		
		// Smart Architecture Detection — set auth headers based on target platform
		if strings.Contains(targetBase, "anthropic.com") {
			req.Header.Set("x-api-key", actualKey)
			req.Header.Set("anthropic-version", "2023-06-01")
		} else if strings.Contains(targetBase, "googlegenesis") || strings.Contains(targetBase, "generativelanguage") {
			if actualKey != "" && !strings.Contains(fullURL, "key=") {
				if strings.Contains(fullURL, "?") {
					fullURL += "&key=" + actualKey
				} else {
					fullURL += "?key=" + actualKey
				}
				newURL, err := url.Parse(fullURL)
				if err == nil {
					req.URL = newURL
				}
			}
		} else if actualKey != "" {
			req.Header.Set("Authorization", "Bearer "+actualKey)
		}

		client := &http.Client{Timeout: 120 * time.Second}
		start := time.Now()
		resp, err := client.Do(req)
		latency := time.Since(start).Milliseconds()

		if err != nil {
			decrementConcurrent(card.ID)
			// On error, we could potentially remove this key from cache temporarily,
			// but for now we'll just continue to the next attempt.
			continue
		}

		// Check for retriable status codes BEFORE consuming the body
		if resp.StatusCode == 429 || resp.StatusCode == 401 {
			resp.Body.Close() 
			decrementConcurrent(card.ID)
			continue
		}

		// Success path — forward response
		decrementConcurrent(card.ID)
		logUsage(selectedKey.ID, card.ID, c.Method(), fullURL, resp.StatusCode, int(latency), c.IP())

		c.Status(resp.StatusCode)

		// Check if this is a streaming response
		contentType := resp.Header.Get("Content-Type")
		isStreaming := strings.Contains(contentType, "text/event-stream") ||
			resp.Header.Get("Transfer-Encoding") == "chunked"

		if isStreaming {
			// Stream: pipe response directly to client
			for k, v := range resp.Header {
				if len(v) > 0 { c.Set(k, v[0]) }
			}
			c.Set("Cache-Control", "no-cache")
			c.Set("Connection", "keep-alive")

			c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
				defer resp.Body.Close()
				buf := make([]byte, 4096)
				for {
					n, err := resp.Body.Read(buf)
					if n > 0 {
						w.Write(buf[:n])
						w.Flush()
					}
					if err != nil {
						break
					}
				}
			})
			return nil
		}

		// Non-streaming: read full body and send
		defer resp.Body.Close()
		for k, v := range resp.Header {
			if len(v) > 0 { c.Set(k, v[0]) }
		}
		body, _ := io.ReadAll(resp.Body)
		return c.Send(body)
	}

	return c.Status(502).JSON(fiber.Map{"error": true, "message": "All keys exhausted or rate-limited"})
}

// ═══════════════════════════════════════════════
//  Public Handlers
// ═══════════════════════════════════════════════

// ProxyHandler handles /v1/proxy/:provider/* requests (legacy path)
func ProxyHandler(c *fiber.Ctx) error {
	provider := strings.ToLower(c.Params("provider"))
	if provider == "" {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Provider is required"})
	}

	// 1. Try Cache first
	var cardID string
	cacheKey := "provider_to_id:" + provider
	exists, _ := cache.Get(cacheKey, &cardID)

	if !exists {
		// Find card ID by provider name or card name
		err := database.Router.GetGlobalManagerDB().QueryRow(`
			SELECT c.id FROM api_cards c
			LEFT JOIN managed_api_keys k ON k.card_id = c.id
			WHERE (LOWER(c.name) = ? OR LOWER(k.provider) = ?) AND c.is_deleted = 0
			LIMIT 1
		`, provider, provider).Scan(&cardID)

		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": true, "message": "Provider not found"})
		}
		// Store mapping for 1 hour
		_ = cache.Set(cacheKey, cardID, 1*time.Hour)
	}

	return executeProxy(c, cardID)
}

// EcosystemProxyHandler handles /:section/:card/execute requests (primary path)
func EcosystemProxyHandler(c *fiber.Ctx) error {
	sectionSlug := strings.ToLower(c.Params("section"))
	cardSlug := strings.ToLower(c.Params("card"))
	
	// 1. Try Cache first
	var cardID string
	slugKey := fmt.Sprintf("slug_to_id:%s:%s", sectionSlug, cardSlug)
	exists, _ := cache.Get(slugKey, &cardID)

	if exists {
		return executeProxy(c, cardID)
	}

	// 2. Cache Miss - Resolve Slugs (This is heavy, so we cache it)
	rows, err := database.Router.GetGlobalManagerDB().Query(`
		SELECT c.id, c.name, s.name
		FROM api_cards c
		JOIN api_sections s ON c.section_id = s.id
		WHERE c.is_deleted = 0
	`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Database error"})
	}
	defer rows.Close()

	for rows.Next() {
		var cid, cname, sname string
		rows.Scan(&cid, &cname, &sname)
		
		genSection := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(strings.TrimSpace(sname)), "(", ""), ")", "")
		genSection = strings.ReplaceAll(genSection, " ", "-")
		
		genCard := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(strings.TrimSpace(cname)), "(", ""), ")", "")
		genCard = strings.ReplaceAll(genCard, " ", "-")
		
		if genSection == sectionSlug && (genCard == cardSlug || cid == cardSlug) {
			cardID = cid
			// Cache this slug mapping for 1 hour
			_ = cache.Set(slugKey, cardID, 1*time.Hour)
			break
		}
	}

	if cardID == "" {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Ecosystem endpoint not found"})
	}
	return executeProxy(c, cardID)
}


// ═══════════════════════════════════════════════
//  Usage Logging
// ═══════════════════════════════════════════════

func logUsage(keyID, cardID, method, path string, status, latency int, ip string) {
	id := uuid.New().String()
	_, _ = database.Router.GetGlobalManagerDB().Exec(`
		INSERT INTO api_usage_logs (id, key_id, card_id, method, path, status_code, latency_ms, ip_address, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, keyID, cardID, method, path, status, latency, ip, time.Now().Unix())
}

// ═══════════════════════════════════════════════
//  Admin: List Usage Logs
// ═══════════════════════════════════════════════

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
