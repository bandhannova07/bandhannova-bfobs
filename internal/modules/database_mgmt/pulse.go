package database_mgmt

import (
	"log"
	"sync"
	"time"

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

	// 1. Check Core Shards
	checkShard(tempResults, "Auth Shard", "core", database.Router.GetAuthDB())
	checkShard(tempResults, "Analytics Shard", "core", database.Router.GetAnalyticsDB())
	checkShard(tempResults, "Global Manager", "core", database.Router.GetGlobalManagerDB())

	// 2. Check User Shards
	for i := 0; i < database.Router.GetShardCount(); i++ {
		name := "User Shard " + string(rune(48+i))
		checkShard(tempResults, name, "user", database.Router.GetUserDB(name))
	}

	// 3. Check Managed Databases
	managedDBs := database.Router.GetAllManagedDBs()
	for _, mdb := range managedDBs {
		checkShard(tempResults, mdb.Name, "managed", mdb.DB)
	}

	// Update Global Map
	pulseMu.Lock()
	PulseResults = tempResults
	pulseMu.Unlock()
	log.Printf("💓 Pulse Check Complete: %d shards analyzed", len(tempResults))
}

func checkShard(results map[string]ShardHealth, name, shardType string, db interface{}) {
	if db == nil {
		results[name] = ShardHealth{Name: name, Type: shardType, Status: "offline", LastCheck: time.Now()}
		return
	}

	// Dynamic type checking for different DB types if needed (SQL, Redis etc.)
	// For now, focusing on SQL DBs
	sqlDB, ok := db.(interface{ Ping() error })
	if !ok {
		return
	}

	start := time.Now()
	err := sqlDB.Ping()
	latency := time.Since(start)

	status := "healthy"
	errMsg := ""
	if err != nil {
		status = "offline"
		errMsg = err.Error()
	} else if latency > 500*time.Millisecond {
		status = "unstable"
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
