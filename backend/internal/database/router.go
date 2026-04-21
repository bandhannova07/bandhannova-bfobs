package database

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"log"
	"sync"
)

// ShardType represents the category of a database shard
type ShardType string

const (
	ShardAuth           ShardType = "auth"
	ShardAnalytics      ShardType = "analytics"
	ShardGlobalManager  ShardType = "global"
	ShardCoreMaster     ShardType = "core"
	ShardUser           ShardType = "user"
)

// StoragePlan defines limits per subscription tier
type StoragePlan struct {
	Name             string
	MaxChatMessages  int
	MaxConversations int
	MaxStorageBytes  int64
	MaxSavedItems    int
	MaxDailyQuota    int
}

var Plans = map[string]StoragePlan{
	"free": {
		Name:             "Free",
		MaxChatMessages:  500,
		MaxConversations: 20,
		MaxStorageBytes:  5 * 1024 * 1024,
		MaxSavedItems:    50,
		MaxDailyQuota:    100,
	},
	"pro": {
		Name:             "Pro",
		MaxChatMessages:  5000,
		MaxConversations: 200,
		MaxStorageBytes:  50 * 1024 * 1024,
		MaxSavedItems:    500,
		MaxDailyQuota:    1000,
	},
	"ultra": {
		Name:             "Ultra",
		MaxChatMessages:  25000,
		MaxConversations: 1000,
		MaxStorageBytes:  250 * 1024 * 1024,
		MaxSavedItems:    2500,
		MaxDailyQuota:    10000,
	},
	"maxx": {
		Name:             "Maxx",
		MaxChatMessages:  100000,
		MaxConversations: 5000,
		MaxStorageBytes:  1024 * 1024 * 1024,
		MaxSavedItems:    10000,
		MaxDailyQuota:    100000,
	},
}

type ManagedDB struct {
	Slug     string
	Name     string
	Category string
	DB       *sql.DB
}

type ShardRouter struct {
	mu              sync.RWMutex
	authDB          *sql.DB
	analyticsDB     *sql.DB
	coreMasterDB    *sql.DB
	globalManagerDBs []*sql.DB
	userDBs         []*sql.DB

	coreAuthDB          *sql.DB
	coreAnalyticsDB     *sql.DB
	coreCoreMasterDB    *sql.DB
	coreGlobalManagerDBs []*sql.DB
	coreUserDBs         []*sql.DB

	managedDBs map[string]ManagedDB
}

var Router *ShardRouter

func InitShardRouter(authURL, authToken, analyticsURL, analyticsToken, coreURL, coreToken string, masterKey string) error {
	router := &ShardRouter{}
	var err error

	// 1. Connect to Core Master (Shard 1 - The Brain)
	if coreURL != "" {
		router.coreMasterDB, err = ConnectTurso(coreURL, coreToken)
		if err != nil {
			log.Printf("⚠️ Core Master shard connection failed: %v", err)
			return fmt.Errorf("core master connection failed: %w", err)
		}
		
		// Ensure Infrastructure Schema exists
		if err := InitInfrastructureSchema(router.coreMasterDB); err != nil {
			log.Printf("⚠️ Failed to apply infrastructure schema: %v", err)
		}
	} else {
		return fmt.Errorf("TURSO_CORE_URL is required for bootstrapping")
	}

	// 2. Fetch Global Manager Shards from Infrastructure Registry
	rows, err := router.coreMasterDB.Query("SELECT id, name, type, db_url, encrypted_token FROM infrastructure_shards WHERE status = 'active'")
	if err != nil {
		log.Printf("⚠️ Failed to query infrastructure shards: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var id, name, sType, dbURL, encrypted string
			if err := rows.Scan(&id, &name, &sType, &dbURL, &encrypted); err != nil {
				continue
			}

			token, err := security.Decrypt(encrypted, masterKey)
			if err != nil {
				log.Printf("⚠️ Failed to decrypt shard token for %s: %v", name, err)
				continue
			}

			db, err := ConnectTurso(dbURL, token)
			if err != nil {
				log.Printf("⚠️ Failed to connect to shard %s: %v", name, err)
				continue
			}

			switch sType {
			case "global_manager":
				router.globalManagerDBs = append(router.globalManagerDBs, db)
			case "auth":
				router.authDB = db
			case "analytics":
				router.analyticsDB = db
			case "user":
				router.userDBs = append(router.userDBs, db)
			}
		}
	}

	// 3. Fallback to Env-based Auth/Analytics if not found in Registry
	if router.authDB == nil && authURL != "" {
		router.authDB, _ = ConnectTurso(authURL, authToken)
	}
	if router.analyticsDB == nil && analyticsURL != "" {
		router.analyticsDB, _ = ConnectTurso(analyticsURL, analyticsToken)
	}

	if len(router.globalManagerDBs) == 0 {
		log.Println("⚠️ No Global Manager shards found in registry. System functionality will be limited.")
	}

	router.coreAuthDB = router.authDB
	router.coreAnalyticsDB = router.analyticsDB
	router.coreCoreMasterDB = router.coreMasterDB
	router.coreGlobalManagerDBs = append([]*sql.DB{}, router.globalManagerDBs...)
	router.coreUserDBs = append([]*sql.DB{}, router.userDBs...)
	router.managedDBs = make(map[string]ManagedDB)

	Router = router
	log.Printf("🧠 Shard Router Ready: [CoreMaster, %d GlobalManager, %d User Shards]", len(router.globalManagerDBs), len(router.userDBs))
	return nil
}

func (sr *ShardRouter) GetAuthDB() *sql.DB             { return sr.authDB }
func (sr *ShardRouter) GetAnalyticsDB() *sql.DB        { return sr.analyticsDB }
func (sr *ShardRouter) GetCoreMasterDB() *sql.DB       { return sr.coreMasterDB }

func (sr *ShardRouter) GetAllGlobalManagerDBs() []*sql.DB {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return sr.globalManagerDBs
}

func (sr *ShardRouter) GetGlobalManagerDB() *sql.DB {
	// Default to first shard if no slug provided
	if len(sr.globalManagerDBs) == 0 {
		return nil
	}
	return sr.globalManagerDBs[0]
}

func (sr *ShardRouter) GetGlobalManagerDBBySlug(slug string) *sql.DB {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	if len(sr.globalManagerDBs) == 0 {
		return nil
	}
	h := fnv.New32a()
	h.Write([]byte(slug))
	idx := int(h.Sum32()) % len(sr.globalManagerDBs)
	return sr.globalManagerDBs[idx]
}

func (sr *ShardRouter) GetUserDB(userID string) *sql.DB {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	if len(sr.userDBs) == 0 {
		return nil
	}
	h := fnv.New32a()
	h.Write([]byte(userID))
	idx := int(h.Sum32()) % len(sr.userDBs)
	return sr.userDBs[idx]
}

func (sr *ShardRouter) GetShardCount() int {
	return len(sr.userDBs)
}

func (sr *ShardRouter) GetUserShardIndex(userID string) int {
	if len(sr.userDBs) == 0 {
		return 0
	}
	h := fnv.New32a()
	h.Write([]byte(userID))
	return int(h.Sum32()) % len(sr.userDBs)
}

func GetPlan(tier string) StoragePlan {
	if plan, ok := Plans[tier]; ok {
		return plan
	}
	return Plans["free"]
}

func (sr *ShardRouter) ReloadDynamicDBs(dbs []ManagedDB) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.authDB = sr.coreAuthDB
	sr.analyticsDB = sr.coreAnalyticsDB
	sr.coreMasterDB = sr.coreCoreMasterDB
	sr.globalManagerDBs = append([]*sql.DB{}, sr.coreGlobalManagerDBs...)
	sr.userDBs = append([]*sql.DB{}, sr.coreUserDBs...)
	
	newManaged := make(map[string]ManagedDB)
	for _, mdb := range dbs {
		newManaged[mdb.Slug] = mdb
		switch mdb.Category {
		case "auth":
			sr.authDB = mdb.DB
		case "analytics":
			sr.analyticsDB = mdb.DB
		case "core":
			sr.coreMasterDB = mdb.DB
		case "global":
			sr.globalManagerDBs = append(sr.globalManagerDBs, mdb.DB)
		case "user":
			sr.userDBs = append(sr.userDBs, mdb.DB)
		}
	}
	sr.managedDBs = newManaged
}

func (sr *ShardRouter) GetManagedDBBySlug(slug string) *sql.DB {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	if mdb, ok := sr.managedDBs[slug]; ok {
		return mdb.DB
	}
	if slug == "core-auth" {
		return sr.coreAuthDB
	}
	if slug == "core-analytics" {
		return sr.coreAnalyticsDB
	}
	if slug == "core-global" {
		return sr.GetGlobalManagerDB()
	}
	if slug == "core-master" {
		return sr.coreCoreMasterDB
	}
	return nil
}

func (sr *ShardRouter) GetAllManagedDBs() []ManagedDB {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	var list []ManagedDB
	for _, mdb := range sr.managedDBs {
		list = append(list, mdb)
	}
	return list
}
