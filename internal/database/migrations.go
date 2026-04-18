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
	if db := Router.GetAuthDB(); db != nil {
		if err := InitAuthSchema(db); err != nil {
			log.Printf("⚠️ Auth migration error: %v", err)
		} else {
			log.Println("  ✅ Auth schema migrated")
		}
	}

	// 2. Analytics Shard
	if db := Router.GetAnalyticsDB(); db != nil {
		if err := InitAnalyticsSchema(db); err != nil {
			log.Printf("⚠️ Analytics migration error: %v", err)
		} else {
			log.Println("  ✅ Analytics schema migrated")
		}
	}

	// 3. Global Manager Shard
	if db := Router.GetGlobalManagerDB(); db != nil {
		if err := InitGlobalManagerSchema(db); err != nil {
			log.Printf("⚠️ Global Manager migration error: %v", err)
		} else {
			log.Println("  ✅ Global Manager schema migrated")
		}
	}

	// 4. User Shards (Apply User Schema to all user shards)
	for i := 0; i < Router.GetShardCount(); i++ {
		if db := Router.userDBs[i]; db != nil {
			if err := InitUserSchema(db); err != nil {
				log.Printf("⚠️ User Shard %d migration error: %v", i, err)
			}
		}
	}
	if Router.GetShardCount() > 0 {
		log.Println("  ✅ User Ecosystem schema migrated on all shards")
	}

	log.Println("🧠 All BandhanNova Ecosystem migrations complete!")
}
