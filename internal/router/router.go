package router

import (
	"time"

	"github.com/bandhannova/api-hunter/internal/middleware"
	"github.com/bandhannova/api-hunter/internal/modules/admin"
	"github.com/bandhannova/api-hunter/internal/modules/ai"
	"github.com/bandhannova/api-hunter/internal/modules/auth"
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
	// Root Landing Page (Animated Status)
	app.Get("/", system.ServeStatusPage)

	// API Gateway Versioning
	v1 := app.Group("/v1", middleware.RedisRateLimiter(60, time.Minute))

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
	
	// Admin Key Management
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

	// Webhooks
	app.Post("/webhooks/resend", email.HandleEmailWebhook)
}
