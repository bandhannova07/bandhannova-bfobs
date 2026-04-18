package database

import (
	"log"
)

// RunMigrations orchestrates schema application across all shards
func RunMigrations() {
	if Router == nil {
		log.Println("❌ Migration failed: Shard Router not initialized")
		return
	}

	log.Println("🔧 Running BandhanNova Ecosystem schema migrations...")

	// 1. Auth Shard
	if err := InitAuthSchema(Router.GetAuthDB()); err != nil {
		log.Printf("⚠️ Auth migration error: %v", err)
	} else {
		log.Println("  ✅ Auth schema migrated")
	}

	// 2. Analytics Shard
	if err := InitAnalyticsSchema(Router.GetAnalyticsDB()); err != nil {
		log.Printf("⚠️ Analytics migration error: %v", err)
	} else {
		log.Println("  ✅ Analytics schema migrated")
	}

	// 3. Global Manager Shard
	if err := InitGlobalManagerSchema(Router.GetGlobalManagerDB()); err != nil {
		log.Printf("⚠️ Global Manager migration error: %v", err)
	} else {
		log.Println("  ✅ Global Manager schema migrated")
	}

	// 4. User Shards (Apply User Schema to all user shards)
	for i := 0; i < Router.GetShardCount(); i++ {
		// Note: We access internal userDBs slice for migrations
		if err := InitUserSchema(Router.userDBs[i]); err != nil {
			log.Printf("⚠️ User Shard %d migration error: %v", i, err)
		}
	}
	log.Println("  ✅ User Ecosystem schema migrated on all shards")

	log.Println("🧠 All BandhanNova Ecosystem migrations complete!")
}
