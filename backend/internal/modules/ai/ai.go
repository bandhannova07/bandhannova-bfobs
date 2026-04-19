package ai

import (
	"encoding/json"
	"log"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/proxy"
	"github.com/gofiber/fiber/v2"
)

var (
	OpenRouterKM *proxy.KeyManager
	GroqKM       *proxy.KeyManager
	CerebrasKM   *proxy.KeyManager
)

func InitAIHandlers() {
	OpenRouterKM = proxy.NewKeyManager(config.AppConfig.OpenRouterKeys, "OpenRouter")
	GroqKM = proxy.NewKeyManager(config.AppConfig.GroqKeys, "Groq")
	CerebrasKM = proxy.NewKeyManager(config.AppConfig.CerebrasKeys, "Cerebras")
}

// ProxyOpenRouter handles AI chat completion requests with automatic model fallback
func ProxyOpenRouter(c *fiber.Ctx) error {
	targetURL := "https://openrouter.ai/api/v1/chat/completions"

	// Preferred model chain (OpenRouter)
	models := []string{
		"arcee-ai/trinity-large-preview:free",
		"arcee-ai/trinity-mini:free",
		"mistralai/mistral-small-3.1-24b-instruct:free",
	}

	rawBody := c.Body()
	var lastErr error

	for _, model := range models {
		var bodyMap map[string]interface{}
		if err := json.Unmarshal(rawBody, &bodyMap); err != nil {
			bodyMap = make(map[string]interface{})
		}
		bodyMap["model"] = model

		newBody, _ := json.Marshal(bodyMap)

		err := proxy.ProxyRequestCustomBody(c, []string{targetURL}, []*proxy.KeyManager{OpenRouterKM}, []string{"Bearer %s"}, newBody)
		if err == nil {
			return nil
		}

		lastErr = err
		log.Printf("Model %s failed, trying next fallback...", model)
	}

	// Ultimate Fallback: Cerebras gpt-oss-120b
	log.Printf("All OpenRouter models exhausted, falling back to Cerebras gpt-oss-120b...")
	cerebrasErr := ProxyCerebras(c)
	if cerebrasErr == nil {
		return nil
	}

	return lastErr
}

// ProxyCerebras handles AI chat completions via Cerebras Cloud (gpt-oss-120b)
func ProxyCerebras(c *fiber.Ctx) error {
	targetURL := "https://api.cerebras.ai/v1/chat/completions"

	rawBody := c.Body()
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(rawBody, &bodyMap); err != nil {
		bodyMap = make(map[string]interface{})
	}
	bodyMap["model"] = "gpt-oss-120b"

	newBody, _ := json.Marshal(bodyMap)

	return proxy.ProxyRequestCustomBody(c, []string{targetURL}, []*proxy.KeyManager{CerebrasKM}, []string{"Bearer %s"}, newBody)
}
