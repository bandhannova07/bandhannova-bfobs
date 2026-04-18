package main

import (
	"log"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/bandhannova/api-hunter/internal/ex-db/cache"
	"github.com/bandhannova/api-hunter/internal/modules/admin"
	"github.com/bandhannova/api-hunter/internal/modules/ai"
	"github.com/bandhannova/api-hunter/internal/modules/database_mgmt"
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

	// 2. Initialize Core Subsystems
	err := database.InitShardRouter(
		config.AppConfig.TursoAuthURL, config.AppConfig.TursoAuthToken,
		config.AppConfig.TursoAnalyticsURL, config.AppConfig.TursoAnalyticsToken,
		config.AppConfig.TursoGlobalURL, config.AppConfig.TursoGlobalToken,
		config.AppConfig.TursoUserShardURLs, config.AppConfig.TursoUserShardTokens,
	)
	if err != nil {
		log.Printf("⚠️  Database Shards initialization warning: %v", err)
	}

	// 3. Run Schema Migrations
	database.RunMigrations()

	// 4. Initialize Redis Cache (Optional but recommended)
	if config.AppConfig.RedisURL != "" {
		if err := cache.InitRedis(config.AppConfig.RedisURL); err != nil {
			log.Printf("⚠️  Redis initialization warning: %v", err)
		}
	}

	admin.InitAdminHandlers()
	ai.InitAIHandlers()
	search.InitSearchHandlers()
	email.InitEmailHandlers()
	market.InitMarketHandlers()

	// 3. Hot-Reload Dynamic Managed Databases
	go func() {
		if err := database_mgmt.ReloadManagedDatabases(); err != nil {
			log.Printf("⚠️  Dynamic database sync warning: %v", err)
		}
		if err := database_mgmt.ReloadManagedAPIKeys(); err != nil {
			log.Printf("⚠️  API Key sync warning: %v", err)
		}
		database_mgmt.HarmonizeNames()
		// Start Pulse Monitoring (Every 60 seconds)
		database_mgmt.StartPulseWorker(60 * time.Second)
		// Start Anti-Sleep System (Every 3 minutes)
		system.StartAntiSleepWorker(3 * time.Minute)
	}()

	// 4. Initialize Fiber App
	app := fiber.New(fiber.Config{
		AppName:           "BandhanNova BFOBS v2.1",
		EnablePrintRoutes: false,
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
