package ai

import (
	"github.com/bandhannova/api-hunter/internal/proxy"
	"github.com/gofiber/fiber/v2"
)

// ProxySTT handles STT (Speech-to-Text) requests (Dedicated to Groq)
func ProxySTT(c *fiber.Ctx) error {
	targetURL := "https://api.groq.com/openai/v1/audio/transcriptions"

	// Use GroqKM which is initialized in ai_handler.go
	return proxy.ProxyRequest(c, targetURL, GroqKM, "Bearer %s")
}
