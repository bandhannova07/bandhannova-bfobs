package database_mgmt

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/bandhannova/api-hunter/internal/cache"
	"github.com/bandhannova/api-hunter/internal/database"
)

// ShardHealth represents the real-time status of a database shard
type ShardHealth struct {
	Name      string        `json:"name"`
	Type      string        `json:"type"`
	Status    string        `json:"status"` // healthy, unstable, offline
	Latency   time.Duration `json:"latency"`
	LastCheck time.Time     `json:"last_check"`
	Error     string        `json:"error,omitempty"`
}

var (
	PulseResults = make(map[string]ShardHealth)
	pulseMu      sync.RWMutex
)

// StartPulseWorker runs in the background and pings all shards periodically
func StartPulseWorker(interval time.Duration) {
	log.Printf("💓 Pulse Monitoring System started (Interval: %v)", interval)
	ticker := time.NewTicker(interval)
	
	// Initial run
	checkAllShards()

	go func() {
		for range ticker.C {
			checkAllShards()
		}
	}()
}

func checkAllShards() {
	if database.Router == nil {
		return
	}

	tempResults := make(map[string]ShardHealth)
	log.Println("💓 [PULSE] Initiating Fleet-Wide Heartbeat Check...")

	// 1. Core Master Shard (Always present)
	checkShard(tempResults, "Core Master", "system", database.Router.GetCoreMasterDB())

	// 2. Discover and Check all Infrastructure Shards
	infraDBs := database.Router.GetAllGlobalManagerDBs()
	for i, db := range infraDBs {
		name := fmt.Sprintf("Global-Infra-%d", i+1)
		checkShard(tempResults, name, "infrastructure", db)
	}

	authDBs := database.Router.GetAllAuthDBs()
	for i, db := range authDBs {
		name := fmt.Sprintf("Auth-Node-%d", i+1)
		checkShard(tempResults, name, "authentication", db)
	}

	analyticsDBs := database.Router.GetAllAnalyticsDBs()
	for i, db := range analyticsDBs {
		name := fmt.Sprintf("Analytics-Node-%d", i+1)
		checkShard(tempResults, name, "analytics", db)
	}

	// 3. User Shards (Dynamic)
	for i, db := range database.Router.GetAllUserDBs() {
		name := fmt.Sprintf("User-Shard-%d", i+1)
		checkShard(tempResults, name, "user_data", db)
	}

	// 4. Managed Product Databases
	managedDBs := database.Router.GetAllManagedDBs()
	for _, mdb := range managedDBs {
		checkShard(tempResults, mdb.Name, "managed_product", mdb.DB)
	}

	// 5. Redis Cache
	checkRedisStatus(tempResults)

	// Update Global Map
	pulseMu.Lock()
	PulseResults = tempResults
	pulseMu.Unlock()
	
	log.Printf("📊 [PULSE] Summary: %d Nodes Active | Fleet Status: STABLE", len(tempResults))
}

func checkShard(results map[string]ShardHealth, name, shardType string, db interface{}) {
	if db == nil || (reflect.ValueOf(db).Kind() == reflect.Ptr && reflect.ValueOf(db).IsNil()) {
		results[name] = ShardHealth{Name: name, Type: shardType, Status: "offline", LastCheck: time.Now()}
		log.Printf("❌ [PULSE] %-20s | Status: OFFLINE", name)
		return
	}

	sqlDB, ok := db.(interface {
		QueryRow(query string, args ...interface{}) *sql.Row
	})
	if !ok {
		return
	}

	start := time.Now()
	// Using a real query to prevent Turso idle sleep
	var one int
	err := sqlDB.QueryRow("SELECT 1").Scan(&one)
	latency := time.Since(start)

	status := "healthy"
	errMsg := ""
	if err != nil {
		status = "offline"
		errMsg = err.Error()
		log.Printf("❌ [PULSE] %-20s | Type: %-15s | Status: FAILED | Error: %v", name, shardType, err)
	} else {
		if latency > 500*time.Millisecond {
			status = "unstable"
		}
		log.Printf("✅ [PULSE] %-20s | Type: %-15s | Status: ONLINE | Latency: %v", name, shardType, latency)
	}

	results[name] = ShardHealth{
		Name:      name,
		Type:      shardType,
		Status:    status,
		Latency:   latency,
		LastCheck: time.Now(),
		Error:     errMsg,
	}
}

// GetPulseStatus returns the current health of all shards
func GetPulseStatus() map[string]ShardHealth {
	pulseMu.RLock()
	defer pulseMu.RUnlock()
	return PulseResults
}

func checkRedisStatus(results map[string]ShardHealth) {
	if cache.RedisClient == nil {
		results["Upstash Redis"] = ShardHealth{Name: "Upstash Redis", Type: "cache", Status: "offline", LastCheck: time.Now(), Error: "Redis not configured"}
		return
	}

	start := time.Now()
	err := cache.RedisClient.Ping(context.Background()).Err()
	latency := time.Since(start)

	status := "healthy"
	errMsg := ""
	if err != nil {
		status = "offline"
		errMsg = err.Error()
	}

	results["Upstash Redis"] = ShardHealth{
		Name:      "Upstash Redis",
		Type:      "cache",
		Status:    status,
		Latency:   latency,
		LastCheck: time.Now(),
		Error:     errMsg,
	}
}

