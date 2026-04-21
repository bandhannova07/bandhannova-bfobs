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

	// 1. Auth Shards Cluster
	for i, db := range Router.GetAllAuthDBs() {
		if err := InitAuthSchema(db); err != nil {
			log.Printf("⚠️ Auth Shard %d migration error: %v", i, err)
		} else {
			log.Printf("  ✅ Auth Shard %d migrated", i)
		}
	}

	// 2. Analytics Shards Cluster
	for i, db := range Router.GetAllAnalyticsDBs() {
		if err := InitAnalyticsSchema(db); err != nil {
			log.Printf("⚠️ Analytics Shard %d migration error: %v", i, err)
		} else {
			log.Printf("  ✅ Analytics Shard %d migrated", i)
		}
	}

	// 3. Global Manager Shards Cluster
	for i, db := range Router.GetAllGlobalManagerDBs() {
		if err := InitGlobalManagerSchema(db); err != nil {
			log.Printf("⚠️ Global Manager Shard %d migration error: %v", i, err)
		} else {
			log.Printf("  ✅ Global Manager Shard %d migrated", i)
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
