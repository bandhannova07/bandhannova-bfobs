package search

import (
	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/proxy"
	"github.com/gofiber/fiber/v2"
)

var TavilyKM *proxy.KeyManager

func InitSearchHandlers() {
	TavilyKM = proxy.NewKeyManager(config.AppConfig.TavilyKeys, "Tavily")
}

// ProxyTavily handles web search requests
func ProxyTavily(c *fiber.Ctx) error {
	targetURL := "https://api.tavily.com/search"

	// Tavily uses a JSON body field "api_key" or "X-API-KEY" header
	// Since raw ProxyRequest for now handles headers, we'll use X-API-KEY format
	return proxy.ProxyRequest(c, targetURL, TavilyKM, "X-API-KEY")
}
