package main

import (
	"log"
	"time"

	"github.com/bandhannova/api-hunter/internal/cache"
	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/bandhannova/api-hunter/internal/modules/admin"
	"github.com/bandhannova/api-hunter/internal/modules/ai"
	"github.com/bandhannova/api-hunter/internal/database_mgmt"
	"github.com/bandhannova/api-hunter/internal/modules/email"
	"github.com/bandhannova/api-hunter/internal/modules/market"
	"github.com/bandhannova/api-hunter/internal/modules/search"
	"github.com/bandhannova/api-hunter/internal/modules/system"
	"github.com/bandhannova/api-hunter/internal/router"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// 1. Initialize Configuration
	config.LoadConfig()

	// 2. Initialize Database Shards
	err := database.InitShardRouter(
		config.AppConfig.TursoAuthURL, config.AppConfig.TursoAuthToken,
		config.AppConfig.TursoAnalyticsURL, config.AppConfig.TursoAnalyticsToken,
		config.AppConfig.TursoGlobalURL, config.AppConfig.TursoGlobalToken,
		config.AppConfig.TursoUserShardURLs, config.AppConfig.TursoUserShardTokens,
	)
	if err != nil {
		log.Printf("⚠️  Database Shards initialization warning: %v", err)
	}

	// 3. Initialize Cache (Upstash Redis)
	cache.InitRedis()

	// 4. Run Schema Migrations
	database.RunMigrations()


	admin.InitAdminHandlers()
	ai.InitAIHandlers()
	search.InitSearchHandlers()
	email.InitEmailHandlers()
	market.InitMarketHandlers()

	// 3. Sync Dynamic Managed Databases & Keys (Synchronous on boot for stability)
	if err := database_mgmt.ReloadManagedDatabases(); err != nil {
		log.Printf("⚠️  Initial database sync warning: %v", err)
	}
	if err := database_mgmt.ReloadManagedAPIKeys(); err != nil {
		log.Printf("⚠️  Initial API Key sync warning: %v", err)
	}

	// Start Background Workers
	go func() {
		// Start Pulse Monitoring (Every 60 seconds)
		database_mgmt.StartPulseWorker(60 * time.Second)
		// Start Anti-Sleep System (Every 3 minutes)
		system.StartAntiSleepWorker(3 * time.Minute)
	}()

	// Log Cleanup Worker (Runs daily, deletes logs older than 30 days)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		for range ticker.C {
			if database.Router != nil && database.Router.GetGlobalManagerDB() != nil {
				cutoff := time.Now().Unix() - (30 * 24 * 3600)
				result, err := database.Router.GetGlobalManagerDB().Exec(
					"DELETE FROM api_usage_logs WHERE timestamp < ?", cutoff,
				)
				if err == nil {
					deleted, _ := result.RowsAffected()
					log.Printf("🧹 Log Cleanup: Removed %d entries older than 30 days", deleted)
				}
			}
		}
	}()

	// 4. Initialize Fiber App
	app := fiber.New(fiber.Config{
		AppName:           "BandhanNova BFOBS v2.1",
		EnablePrintRoutes: false,
		BodyLimit:         500 * 1024 * 1024,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
			})
		},
	})

	// 5. Global Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, X-BandhanNova-Key, X-Admin-Token, Authorization",
	}))

	// 6. Setup all Routes
	router.SetupRoutes(app)

	// 7. Start Server
	port := config.AppConfig.Port
	if port == "" {
		port = "8080"
	}
	log.Printf("🧠 BFOBS Gateway starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
