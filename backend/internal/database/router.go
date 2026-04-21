package database

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"log"
	"sync"

	"github.com/bandhannova/api-hunter/internal/security"
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
	authDBs         []*sql.DB
	analyticsDBs    []*sql.DB
	globalManagerDBs []*sql.DB
	coreMasterDB    *sql.DB
	userDBs         []*sql.DB
	managedDBs      map[string]ManagedDB

	// Public access for modules
	coreAuthDBs          []*sql.DB
	coreAnalyticsDBs     []*sql.DB
	coreGlobalManagerDBs []*sql.DB
	coreCoreMasterDB     *sql.DB
	coreUserDBs          []*sql.DB
}

var Router *ShardRouter

	// RefreshFleet re-loads all infrastructure shards from the Core Master registry
func (r *ShardRouter) RefreshFleet(masterKey string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	log.Println("♻️  Refreshing Infrastructure Fleet Registry...")

	// Clear current dynamic pools (keep core ones if any)
	r.authDBs = nil
	r.analyticsDBs = nil
	r.globalManagerDBs = nil
	r.userDBs = nil

	// Add Core Master to global managers by default
	if r.coreMasterDB != nil {
		r.globalManagerDBs = append(r.globalManagerDBs, r.coreMasterDB)
	}

	rows, err := r.coreMasterDB.Query("SELECT id, name, type, db_url, encrypted_token FROM infrastructure_shards")
	if err != nil {
		return fmt.Errorf("failed to query fleet registry: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, name, sType, dbURL, encryptedToken string
		if err := rows.Scan(&id, &name, &sType, &dbURL, &encryptedToken); err != nil {
			continue
		}

		token, err := security.Decrypt(encryptedToken, masterKey)
		if err != nil {
			log.Printf("⚠️  Failed to decrypt token for shard %s: %v", name, err)
			continue
		}

		connStr := fmt.Sprintf("%s?authToken=%s", dbURL, token)
		db, err := sql.Open("libsql", connStr)
		if err != nil {
			log.Printf("⚠️  Failed to connect to fleet shard %s: %v", name, err)
			continue
		}

		// Distribute into pools
		switch sType {
		case "auth":
			r.authDBs = append(r.authDBs, db)
		case "analytics":
			r.analyticsDBs = append(r.analyticsDBs, db)
		case "global_manager":
			r.globalManagerDBs = append(r.globalManagerDBs, db)
		case "user":
			r.userDBs = append(r.userDBs, db)
		}
	}

	log.Printf("✅ Fleet Registry Synced: %d Global, %d Auth, %d Analytics, %d User nodes online", 
		len(r.globalManagerDBs), len(r.authDBs), len(r.analyticsDBs), len(r.userDBs))
	
	return nil
}

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

	router.managedDBs = make(map[string]ManagedDB)
	router.coreCoreMasterDB = router.coreMasterDB // CRITICAL: Set backup field
	Router = router

	// 2. Load all other shards from the registry
	if err := Router.RefreshFleet(masterKey); err != nil {
		log.Printf("⚠️ Initial fleet sync failed: %v", err)
	}

	// 3. Fallback: If no shards found in registry but env vars exist, add them
	Router.mu.Lock()
	if len(Router.authDBs) == 0 && authURL != "" {
		db, _ := ConnectTurso(authURL, authToken)
		if db != nil { Router.authDBs = append(Router.authDBs, db) }
	}
	if len(Router.analyticsDBs) == 0 && analyticsURL != "" {
		db, _ := ConnectTurso(analyticsURL, analyticsToken)
		if db != nil { Router.analyticsDBs = append(Router.analyticsDBs, db) }
	}

	// Synchronize core slices for dynamic reloads
	Router.coreAuthDBs = append([]*sql.DB{}, Router.authDBs...)
	Router.coreAnalyticsDBs = append([]*sql.DB{}, Router.analyticsDBs...)
	Router.coreGlobalManagerDBs = append([]*sql.DB{}, Router.globalManagerDBs...)
	Router.coreUserDBs = append([]*sql.DB{}, Router.userDBs...)
	Router.mu.Unlock()

	log.Printf("🧠 Shard Router Initialized: %d Global, %d Auth, %d Analytics, %d User nodes online", 
		len(Router.globalManagerDBs), len(Router.authDBs), len(Router.analyticsDBs), len(Router.userDBs))
	
	return nil
}

func (sr *ShardRouter) GetAuthDB(identifier ...string) *sql.DB {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	if len(sr.authDBs) == 0 {
		return nil
	}
	if len(identifier) > 0 && identifier[0] != "" {
		h := fnv.New32a()
		h.Write([]byte(identifier[0]))
		idx := int(h.Sum32()) % len(sr.authDBs)
		return sr.authDBs[idx]
	}
	return sr.authDBs[0]
}

func (sr *ShardRouter) GetAllAuthDBs() []*sql.DB {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return sr.authDBs
}

func (sr *ShardRouter) GetAnalyticsDB(identifier ...string) *sql.DB {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	if len(sr.analyticsDBs) == 0 {
		return nil
	}
	if len(identifier) > 0 && identifier[0] != "" {
		h := fnv.New32a()
		h.Write([]byte(identifier[0]))
		idx := int(h.Sum32()) % len(sr.analyticsDBs)
		return sr.analyticsDBs[idx]
	}
	return sr.analyticsDBs[0]
}

func (sr *ShardRouter) GetAllAnalyticsDBs() []*sql.DB {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return sr.analyticsDBs
}

func (sr *ShardRouter) GetCoreMasterDB() *sql.DB { return sr.coreMasterDB }

func (sr *ShardRouter) GetAllGlobalManagerDBs() []*sql.DB {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return sr.globalManagerDBs
}

func (sr *ShardRouter) GetGlobalManagerDB() *sql.DB {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
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
	sr.authDBs = append([]*sql.DB{}, sr.coreAuthDBs...)
	sr.analyticsDBs = append([]*sql.DB{}, sr.coreAnalyticsDBs...)
	sr.coreMasterDB = sr.coreCoreMasterDB
	sr.globalManagerDBs = append([]*sql.DB{}, sr.coreGlobalManagerDBs...)
	sr.userDBs = append([]*sql.DB{}, sr.coreUserDBs...)
	
	newManaged := make(map[string]ManagedDB)
	for _, mdb := range dbs {
		newManaged[mdb.Slug] = mdb
		switch mdb.Category {
		case "auth":
			sr.authDBs = append(sr.authDBs, mdb.DB)
		case "analytics":
			sr.analyticsDBs = append(sr.analyticsDBs, mdb.DB)
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
		return sr.GetAuthDB()
	}
	if slug == "core-analytics" {
		return sr.GetAnalyticsDB()
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
func (r *ShardRouter) GetAllUserDBs() []*sql.DB {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.userDBs
}
