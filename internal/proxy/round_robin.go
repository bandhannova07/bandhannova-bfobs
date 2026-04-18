package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

const (
	StatusHealthy     = "Healthy"
	StatusRateLimited = "RateLimited"
	StatusDead        = "Dead"
)

// KeyMetadata tracks the status and health of a single API key
type KeyMetadata struct {
	Value             string
	Provider          string
	Status            string
	SuccessCount      int
	FailureCount      int
	LastUsed          time.Time
	RateLimitReset    time.Time
	LastErrorMessage  string
	LimitRequests     int `json:"limit_requests"`
	RemainingRequests int `json:"remaining_requests"`
}

// RequestLog represents a single proxy request for analytics
type RequestLog struct {
	Timestamp  time.Time
	Method     string
	Path       string
	TargetURL  string
	StatusCode int
	Latency    time.Duration
	Error      string
	KeyUsed    string
}

var (
	GlobalRequestLogs []RequestLog
	LogsMutex         sync.Mutex
)

// AddRequestLog records a request for analytics
func AddRequestLog(log RequestLog) {
	LogsMutex.Lock()
	defer LogsMutex.Unlock()

	// Keep last 100 logs
	GlobalRequestLogs = append(GlobalRequestLogs, log)
	if len(GlobalRequestLogs) > 100 {
		GlobalRequestLogs = GlobalRequestLogs[1:]
	}
}

// KeyManager handles rotation and health of API keys
type KeyManager struct {
	mu   sync.Mutex
	keys []*KeyMetadata
	idx  int
}

// NewKeyManager initializes the manager with metadata
func NewKeyManager(keys []string, provider string) *KeyManager {
	metadata := make([]*KeyMetadata, len(keys))
	for i, k := range keys {
		// Initialize with sensible defaults for unknown limits
		initialRemaining := 1000
		if provider == "Groq" {
			initialRemaining = 100 // Groq will update this via headers
		}

		metadata[i] = &KeyMetadata{
			Value:             k,
			Provider:          provider,
			Status:            StatusHealthy,
			LimitRequests:     initialRemaining,
			RemainingRequests: initialRemaining,
		}
	}
	return &KeyManager{
		keys: metadata,
		idx:  0,
	}
}

// GetKeys returns a snapshot of key metadata for the dashboard
func (km *KeyManager) GetKeys() []KeyMetadata {
	km.mu.Lock()
	defer km.mu.Unlock()

	snapshot := make([]KeyMetadata, len(km.keys))
	for i, k := range km.keys {
		snapshot[i] = *k
	}
	return snapshot
}

// GetKeyCount returns the number of keys managed
func (km *KeyManager) GetKeyCount() int {
	km.mu.Lock()
	defer km.mu.Unlock()
	return len(km.keys)
}

// GetNextKey gets the key with the HIGHEST remaining rate limit
func (km *KeyManager) GetNextKey() *KeyMetadata {
	km.mu.Lock()
	defer km.mu.Unlock()

	if len(km.keys) == 0 {
		return nil
	}

	var bestKey *KeyMetadata
	maxRemaining := -1

	for _, k := range km.keys {
		isAvailable := k.Status == StatusHealthy ||
			(k.Status == StatusRateLimited && time.Now().After(k.RateLimitReset))

		if isAvailable {
			// Update status if it was rate limited but now passed
			if k.Status == StatusRateLimited {
				k.Status = StatusHealthy
				// Reset remaining to a small value to "probationary" test it
				if k.RemainingRequests <= 0 {
					k.RemainingRequests = 10
				}
			}

			// Selection logic: Highest Remaining balance wins
			// If tied, pick the one used longest ago
			if k.RemainingRequests > maxRemaining {
				maxRemaining = k.RemainingRequests
				bestKey = k
			} else if k.RemainingRequests == maxRemaining {
				if bestKey == nil || k.LastUsed.Before(bestKey.LastUsed) {
					bestKey = k
				}
			}
		}
	}

	if bestKey != nil {
		bestKey.LastUsed = time.Now()
		// Optimistically decrement to prevent race conditions before response
		if bestKey.RemainingRequests > 0 {
			bestKey.RemainingRequests--
		}
	}

	return bestKey
}

// UpdateKeyStatus updates the health and rate limits of a key
func (km *KeyManager) UpdateKeyStatus(keyValue string, statusCode int, errMsg string, headers http.Header) {
	km.mu.Lock()
	defer km.mu.Unlock()

	for _, k := range km.keys {
		if k.Value == keyValue {
			// Parse Rate Limit Headers if present (specifically for Groq)
			if headers != nil {
				if remaining := headers.Get("x-ratelimit-remaining-requests"); remaining != "" {
					if val, err := strconv.Atoi(remaining); err == nil {
						k.RemainingRequests = val
					}
				}
				if limit := headers.Get("x-ratelimit-limit-requests"); limit != "" {
					if val, err := strconv.Atoi(limit); err == nil {
						k.LimitRequests = val
					}
				}
			}

			if statusCode >= 200 && statusCode < 400 {
				k.Status = StatusHealthy
				k.SuccessCount++
				k.LastErrorMessage = ""
			} else {
				k.FailureCount++
				k.LastErrorMessage = errMsg

				if statusCode == 429 {
					k.Status = StatusRateLimited
					k.RemainingRequests = 0
					k.RateLimitReset = time.Now().Add(1 * time.Minute)
				} else if statusCode == 401 || statusCode == 403 {
					k.Status = StatusDead
					k.RemainingRequests = 0
				}
			}
			return
		}
	}
}

// ProxyRequestCustomBody sends the request with a custom body and tries multiple providers
func ProxyRequestCustomBody(c *fiber.Ctx, targetURLs []string, keyManagers []*KeyManager, authHeaderFormats []string, customBody []byte) error {
	startTime := time.Now()
	var body []byte
	if customBody != nil {
		body = customBody
	} else {
		body = c.Body()
	}

	if len(keyManagers) == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "No providers configured"})
	}

	var lastRespStatusCode int
	var lastErrorMsg string
	var lastUsedKey string

	// Iterate through providers
	for pIdx, km := range keyManagers {
		targetURL := targetURLs[pIdx]
		authHeaderFormat := authHeaderFormats[pIdx]
		keyCount := km.GetKeyCount()

		// Try keys within this provider
		for attempt := 0; attempt < keyCount; attempt++ {
			keyMeta := km.GetNextKey()
			if keyMeta == nil {
				break
			}

			key := keyMeta.Value
			lastUsedKey = key

			req, err := http.NewRequest(c.Method(), targetURL, bytes.NewBuffer(body))
			if err != nil {
				log.Printf("Error creating request: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create proxy request"})
			}

			req.Header.Set("Content-Type", "application/json")
			if authHeaderFormat != "" {
				req.Header.Set("Authorization", fmt.Sprintf(authHeaderFormat, key))
			} else {
				req.Header.Set("Authorization", "Bearer "+key)
			}

			if authHeaderFormat == "X-API-KEY" {
				req.Header.Del("Authorization")
				req.Header.Set("X-API-KEY", key)

				if c.Method() == "POST" {
					trimmedBody := bytes.TrimSpace(body)
					if bytes.HasPrefix(trimmedBody, []byte("{")) && !bytes.Contains(trimmedBody, []byte("\"api_key\"")) {
						injectedBody := bytes.Replace(trimmedBody, []byte("{"), []byte(fmt.Sprintf("{\"api_key\":\"%s\",", key)), 1)
						req.Body = io.NopCloser(bytes.NewBuffer(injectedBody))
						req.ContentLength = int64(len(injectedBody))
					}
				}
			}

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)

			if err != nil {
				lastErrorMsg = err.Error()
				km.UpdateKeyStatus(key, 500, lastErrorMsg, nil)
				continue
			}

			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close() // Close immediately to avoid leak in loop
			lastRespStatusCode = resp.StatusCode

			km.UpdateKeyStatus(key, resp.StatusCode, string(respBody), resp.Header)

			if resp.StatusCode == 429 || resp.StatusCode == 401 || resp.StatusCode == 403 {
				continue
			}

			AddRequestLog(RequestLog{
				Timestamp:  time.Now(),
				Method:     c.Method(),
				Path:       c.Path(),
				TargetURL:  targetURL,
				StatusCode: resp.StatusCode,
				Latency:    time.Since(startTime),
				KeyUsed:    key[:8] + "...",
				Error:      fmt.Sprintf("Provider: %s", km.keys[0].Provider),
			})

			c.Set("Content-Type", resp.Header.Get("Content-Type"))
			return c.Status(resp.StatusCode).Send(respBody)
		}
	}

	AddRequestLog(RequestLog{
		Timestamp:  time.Now(),
		Method:     c.Method(),
		Path:       c.Path(),
		TargetURL:  "ALL_PROVIDERS",
		StatusCode: lastRespStatusCode,
		Latency:    time.Since(startTime),
		Error:      "All providers and keys exhausted",
		KeyUsed:    lastUsedKey,
	})

	return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
		"error":   true,
		"message": "All API providers and keys exhausted or rate limited",
	})
}

// ProxyRequestMulti is a wrapper that uses c.Body()
func ProxyRequestMulti(c *fiber.Ctx, targetURLs []string, keyManagers []*KeyManager, authHeaderFormats []string) error {
	return ProxyRequestCustomBody(c, targetURLs, keyManagers, authHeaderFormats, nil)
}

// ProxyRequest is a wrapper for a single provider
func ProxyRequest(c *fiber.Ctx, targetURL string, keyManager *KeyManager, authHeaderFormat string) error {
	return ProxyRequestCustomBody(c, []string{targetURL}, []*KeyManager{keyManager}, []string{authHeaderFormat}, nil)
}
