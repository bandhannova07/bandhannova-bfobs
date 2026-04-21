package main

import (
	"fmt"
	"log"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
)

func main() {
	config.LoadConfig()

	// 1. Core Master (Shard 1)
	fmt.Println("🚀 Initializing Core Master (Shard 1)...")
	wipeAndInit(config.AppConfig.TursoCoreURL, config.AppConfig.TursoCoreToken, true)

	// 2. Global Manager Series (2, 4, 5, 6) - WIPE
	gmURLs := config.AppConfig.TursoGlobalURLs
	gmTokens := config.AppConfig.TursoGlobalTokens

	for i := 0; i < len(gmURLs); i++ {
		shardNum := i + 2 // Shard 2 is index 0
		isCurrent := (shardNum == 3)
		
		if isCurrent {
			fmt.Printf("🛡️ Skipping Wipe for Shard 3 (Current Primary). Updating Schema only...\n")
			wipeAndInit(gmURLs[i], gmTokens[i], false)
		} else {
			fmt.Printf("🧹 Wiping and Initializing Shard %d...\n", shardNum)
			wipeAndInit(gmURLs[i], gmTokens[i], true)
		}
	}

	fmt.Println("✅ All Shards Initialized Successfully!")
}

func wipeAndInit(url, token string, wipe bool) {
	if url == "" {
		return
	}
	db, err := database.ConnectTurso(url, token)
	if err != nil {
		log.Printf("❌ Failed to connect to %s: %v", url, err)
		return
	}
	defer db.Close()

	if wipe {
		// Drop all tables
		rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
		if err == nil {
			var tables []string
			for rows.Next() {
				var name string
				rows.Scan(&name)
				tables = append(tables, name)
			}
			rows.Close()
			for _, table := range tables {
				db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
			}
		}
	}

	// Apply Manager Schema
	_, err = db.Exec(database.GlobalManagerSchema)
	if err != nil {
		log.Printf("❌ Failed to apply schema to %s: %v", url, err)
	} else {
		fmt.Printf("✨ Schema applied to %s\n", url)
	}
}
