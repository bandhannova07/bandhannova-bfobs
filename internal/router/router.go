package router

import (
	"time"

	"github.com/bandhannova/api-hunter/internal/middleware"
	"github.com/bandhannova/api-hunter/internal/modules/admin"
	"github.com/bandhannova/api-hunter/internal/modules/ai"
	"github.com/bandhannova/api-hunter/internal/modules/api_mgmt"
	"github.com/bandhannova/api-hunter/internal/modules/api_proxy"
	"github.com/bandhannova/api-hunter/internal/modules/auth"
	"github.com/bandhannova/api-hunter/internal/modules/auth_provider"
	"github.com/bandhannova/api-hunter/internal/modules/database_mgmt"
	"github.com/bandhannova/api-hunter/internal/modules/email"
	"github.com/bandhannova/api-hunter/internal/modules/market"
	"github.com/bandhannova/api-hunter/internal/modules/search"
	"github.com/bandhannova/api-hunter/internal/modules/system"
	"github.com/bandhannova/api-hunter/internal/modules/user"
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures all routes for the BFOBS Gateway
func SetupRoutes(app *fiber.App) {
	// Ecosystem Execution URLs (/{section}/{card}/execute) - Clean root URLs (Priority)
	app.Post("/:section/:card/execute", api_proxy.EcosystemProxyHandler)

	v1 := app.Group("/v1", middleware.RedisRateLimiter(60, time.Minute))

	// OAuth 2.0 / BandhanNova ID Routes
	oauth := v1.Group("/oauth")
	oauth.Get("/authorize", auth_provider.Authorize)
	oauth.Post("/token", auth_provider.Token)
	oauth.Get("/userinfo", auth_provider.UserInfo)

	// Root Landing Page (Animated Status)
	app.Get("/", system.ServeStatusPage)

	// Admin Control Center (Requires Master Key HMAC)
	adminGroup := v1.Group("/admin")
	adminGroup.Post("/login", middleware.AdminLoginRateLimiter(), admin.AdminLogin)
	admin.InitWebAuthn(adminGroup)

	// Protected Admin Routes
	adminAuth := adminGroup.Group("/")
	adminAuth.Use(middleware.AdminAuthRequired())
	
	adminAuth.Get("/status", admin.GetAdminStatus)
	adminAuth.Post("/reload", admin.ReloadKeys)
	adminAuth.Get("/audit", admin.GetAuditLog)
	
	// Admin Key Management (New Hierarchical System)
	adminAuth.Get("/api/sections", api_mgmt.ListSections)
	adminAuth.Post("/api/sections", api_mgmt.AddSection)
	adminAuth.Get("/api/cards", api_mgmt.ListCards)
	adminAuth.Post("/api/cards", api_mgmt.AddCard)
	adminAuth.Put("/api/cards/:id", api_mgmt.UpdateCard)
	adminAuth.Get("/api/keys", api_mgmt.ListKeys)
	adminAuth.Post("/api/keys", api_mgmt.AddKey)
	adminAuth.Post("/api/items/:type/:id/delete", api_mgmt.DeleteAPIItem)
	adminAuth.Get("/api/unused", api_mgmt.ListUnused)
	adminAuth.Delete("/api/unused/:type/:id", api_mgmt.PermanentDelete)
	adminAuth.Get("/api/logs", api_proxy.ListLogs)

	// Admin Key Management (Legacy Support)
	adminAuth.Get("/keys", admin.ListManagedKeys)
	adminAuth.Post("/keys", admin.AddManagedKey)
	adminAuth.Delete("/keys/:id", admin.DeleteManagedKey)
	adminAuth.Post("/keys/:id/check", admin.CheckKeyHealth)

	// Admin Database Management
	adminAuth.Get("/databases", database_mgmt.ListDatabases)
	adminAuth.Post("/databases", database_mgmt.AddDatabase)
	adminAuth.Get("/databases/:slug", database_mgmt.GetDatabaseDetails)
	adminAuth.Get("/health/pulse", database_mgmt.GetPulseHealth)

	// Product Management
	adminAuth.Get("/products", database_mgmt.ListProducts)
	adminAuth.Post("/products", database_mgmt.AddProduct)
	adminAuth.Put("/products/:id", database_mgmt.UpdateProduct)
	adminAuth.Post("/products/:id/delete", database_mgmt.DeleteProduct)
	adminAuth.Post("/products/:id/reset-oauth", database_mgmt.ResetOAuthCredentials)

	// User Auth Routes
	authGroup := v1.Group("/auth")
	authGroup.Post("/signup", auth.Signup)
	authGroup.Post("/login", auth.Login)
	authGroup.Post("/google", auth.GoogleLogin)

	// Protected User Routes
	userGroup := v1.Group("/user")
	userGroup.Use(middleware.AuthRequired())
	userGroup.Get("/profile", user.GetProfile)
	userGroup.Post("/profile", user.UpdateProfile)
	userGroup.Get("/history", user.GetChatHistory)
	userGroup.Post("/history", user.SaveChatMessage)

	// AI Proxies
	v1.Post("/ai/completions", ai.ProxyOpenRouter)      // Groq/OpenRouter Fallback
	v1.Post("/ai/cerebras", ai.ProxyCerebras)           // High speed LLM
	v1.Post("/ai/stt", ai.ProxySTT)                     // Groq Whisper

	// Utility APIs
	v1.Post("/search", search.ProxyTavily)              // AI Search
	v1.Post("/market/quote", market.ProxyTwelveData)    // Stock quotes
	v1.Post("/email/send", email.ProxyEmailSend)        // Send emails

	// Universal API Proxy Gateway (Rotation + Logging)
	v1.All("/proxy/:provider/*", api_proxy.ProxyHandler)

	// Webhooks
	app.Post("/webhooks/resend", email.HandleEmailWebhook)

}
