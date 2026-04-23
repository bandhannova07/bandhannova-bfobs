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
	"github.com/bandhannova/api-hunter/internal/database_mgmt"
	"github.com/bandhannova/api-hunter/internal/modules/email"
	"github.com/bandhannova/api-hunter/internal/modules/market"
	"github.com/bandhannova/api-hunter/internal/modules/search"
	"github.com/bandhannova/api-hunter/internal/modules/system"
	"github.com/bandhannova/api-hunter/internal/modules/user"
	"github.com/bandhannova/api-hunter/internal/storage_mgmt"
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures all routes for the BFOBS Gateway
func SetupRoutes(app *fiber.App) {
	// Ecosystem Execution URLs (/{section}/{card}/execute) - Clean root URLs (Priority)
	// Virtual Database Gateway (The "No Turso Relation" layer)
	app.Post("/db/p/:product_slug/execute", database_mgmt.DatabaseProxyHandler)
	app.Get("/storage/view/:product_slug/:bucket/:filename", storage_mgmt.ProxyViewFile)

	// Main API Group (No more /v1)
	api := app.Group("/", middleware.RedisRateLimiter(60, time.Minute))

	// Cloud Storage System
	storage := api.Group("/storage", middleware.AdminAuthRequired())
	storage.Post("/upload", storage_mgmt.UploadToHuggingFace)
	
	// Bucket Management
	storage.Get("/p/:product_slug/buckets", storage_mgmt.ListBuckets)
	storage.Post("/p/:product_slug/buckets", storage_mgmt.CreateBucket)
	storage.Delete("/buckets/:id", storage_mgmt.DeleteBucket)
	storage.Get("/p/:product_slug/b/:bucket_slug/files", storage_mgmt.ListBucketFiles)
	storage.Delete("/p/:product_slug/b/:bucket_slug/f/:filename", storage_mgmt.DeleteFile)

	// OAuth 2.0 / BandhanNova ID Routes
	oauth := api.Group("/oauth")
	oauth.Get("/authorize", auth_provider.Authorize)
	oauth.Post("/token", auth_provider.Token)
	oauth.Get("/userinfo", auth_provider.UserInfo)

	// Root Landing Page (Animated Status)
	app.Get("/", system.ServeStatusPage)

	// Admin Control Center (Requires Master Key HMAC)
	adminGroup := api.Group("/admin")
	adminGroup.Post("/login", middleware.AdminLoginRateLimiter(), admin.AdminLogin)
	adminGroup.Post("/developer/login", admin.DeveloperLogin)
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

	// Admin Database Management (Supabase-style DB Lab)
	adminAuth.Get("/db/status", admin.ListAllDatabases)
	adminAuth.Post("/db/provision", admin.ProvisionDatabase)
	adminAuth.Put("/db/update/:id", admin.UpdateDatabase)
	adminAuth.Post("/db/remove/:id", database_mgmt.RemoveDatabase)
	adminAuth.Post("/db/execute", admin.ExecuteSQLHandler)
	adminAuth.Post("/db/execute-bulk", admin.BulkExecuteSQLHandler)
	adminAuth.Post("/db/execute-category", admin.ExecuteCategorySQLHandler)
	adminAuth.Post("/db/reset-fleet", admin.ResetFleetHandler)
	adminAuth.Get("/databases", database_mgmt.ListDatabases)
	adminAuth.Post("/databases", database_mgmt.AddDatabase)
	adminAuth.Get("/databases/:slug", database_mgmt.GetDatabaseDetails)
	adminAuth.Get("/health/pulse", database_mgmt.GetPulseHealth)

	// Infrastructure Management (Default Shards)
	adminAuth.Get("/infrastructure/shards", database_mgmt.ListInfrastructureShards)
	adminAuth.Post("/infrastructure/shards", database_mgmt.AddInfrastructureShard)
	adminAuth.Put("/infrastructure/shards/:id", database_mgmt.UpdateInfrastructureShard)
	adminAuth.Delete("/infrastructure/shards/:id", database_mgmt.RemoveInfrastructureShard)
	adminAuth.Post("/infrastructure/shards/:id/query", database_mgmt.QueryInfrastructureShard)
	adminAuth.Post("/infrastructure/shards/:id/clear", database_mgmt.ClearInfrastructureShard)
	adminAuth.Post("/infrastructure/shards/:id/init", database_mgmt.InitializeInfrastructureShard)

	// Product Management
	adminAuth.Get("/products", database_mgmt.ListProducts)
	adminAuth.Get("/products/:slug", database_mgmt.GetProductDetails)
	adminAuth.Post("/products", database_mgmt.AddProduct)
	adminAuth.Put("/products/:id", database_mgmt.UpdateProduct)
	adminAuth.Post("/products/:id/delete", database_mgmt.DeleteProduct)
	adminAuth.Post("/products/:id/reset-oauth", database_mgmt.ResetOAuthCredentials)

	// User Auth Routes
	authGroup := api.Group("/auth")
	authGroup.Post("/signup", auth.Signup)
	authGroup.Post("/login", auth.Login)
	authGroup.Post("/google", auth.GoogleLogin)

	// Protected User Routes
	userGroup := api.Group("/user")
	userGroup.Use(middleware.AuthRequired())
	userGroup.Get("/profile", user.GetProfile)
	userGroup.Post("/profile", user.UpdateProfile)
	userGroup.Get("/history", user.GetChatHistory)
	userGroup.Post("/history", user.SaveChatMessage)

	// AI Proxies
	api.Post("/ai/completions", ai.ProxyOpenRouter)      // Groq/OpenRouter Fallback
	api.Post("/ai/cerebras", ai.ProxyCerebras)           // High speed LLM
	api.Post("/ai/stt", ai.ProxySTT)                     // Groq Whisper

	// Utility APIs
	api.Post("/search", search.ProxyTavily)              // AI Search
	api.Post("/market/quote", market.ProxyTwelveData)    // Stock quotes
	api.Post("/email/send", email.ProxyEmailSend)        // Send emails

	// Universal API Proxy Gateway (Rotation + Logging)
	api.All("/proxy/:provider/*", api_proxy.ProxyHandler)

	// Webhooks
	app.Post("/webhooks/resend", email.HandleEmailWebhook)

}
