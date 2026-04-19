package market

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/proxy"
	"github.com/gofiber/fiber/v2"
)

var TwelveDataKM *proxy.KeyManager

func InitMarketHandlers() {
	TwelveDataKM = proxy.NewKeyManager(config.AppConfig.TwelveDataKeys, "TwelveData")
}

// ProxyTwelveData handles market data requests via TwelveData API
// It supports all TwelveData REST endpoints by passing the path after /v1/market/
// Examples:
//   POST /v1/market/quote?symbol=AAPL
//   POST /v1/market/time_series?symbol=AAPL&interval=1day
//   POST /v1/market/exchange_rate?symbol=USD/INR
//   POST /v1/market/price?symbol=AAPL
func ProxyTwelveData(c *fiber.Ctx) error {
	// Extract the sub-path after /v1/market/
	endpoint := c.Params("*")
	if endpoint == "" {
		endpoint = "quote" // Default to quote
	}

	targetURL := fmt.Sprintf("https://api.twelvedata.com/%s", endpoint)

	startTime := time.Now()

	// Get the next healthy key
	keyMeta := TwelveDataKM.GetNextKey()
	if keyMeta == nil {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":   true,
			"message": "All TwelveData API keys exhausted or rate limited",
		})
	}

	key := keyMeta.Value

	// Build request URL with query params + apikey
	queryString := string(c.Request().URI().QueryString())
	fullURL := targetURL
	if queryString != "" {
		fullURL = fmt.Sprintf("%s?%s&apikey=%s", targetURL, queryString, key)
	} else {
		fullURL = fmt.Sprintf("%s?apikey=%s", targetURL, key)
	}

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create request",
		})
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		TwelveDataKM.UpdateKeyStatus(key, 500, err.Error(), nil)
		log.Printf("TwelveData request failed: %v", err)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": "TwelveData upstream error",
		})
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// Update key health
	TwelveDataKM.UpdateKeyStatus(key, resp.StatusCode, string(respBody), resp.Header)

	// Log the request
	proxy.AddRequestLog(proxy.RequestLog{
		Timestamp:  time.Now(),
		Method:     c.Method(),
		Path:       c.Path(),
		TargetURL:  targetURL,
		StatusCode: resp.StatusCode,
		Latency:    time.Since(startTime),
		KeyUsed:    key[:8] + "...",
		Error:      fmt.Sprintf("Provider: TwelveData | Endpoint: %s", endpoint),
	})

	c.Set("Content-Type", resp.Header.Get("Content-Type"))
	return c.Status(resp.StatusCode).Send(respBody)
}
