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
	globalManagerDB *sql.DB
	userDBs         []*sql.DB

	coreAuthDB          *sql.DB
	coreAnalyticsDB     *sql.DB
	coreGlobalManagerDB *sql.DB
	coreUserDBs         []*sql.DB

	managedDBs map[string]ManagedDB
}

var Router *ShardRouter

func InitShardRouter(authURL, authToken, analyticsURL, analyticsToken, globalURL, globalToken string, userURLs, userTokens []string) error {
	router := &ShardRouter{}
	var err error

	if authURL != "" {
		router.authDB, err = ConnectTurso(authURL, authToken)
		if err != nil {
			log.Printf("⚠️ Auth shard connection failed: %v", err)
		}
	}

	if analyticsURL != "" {
		router.analyticsDB, err = ConnectTurso(analyticsURL, analyticsToken)
		if err != nil {
			log.Printf("⚠️ Analytics shard connection failed: %v", err)
		}
	}

	if globalURL != "" {
		router.globalManagerDB, err = ConnectTurso(globalURL, globalToken)
		if err != nil {
			return fmt.Errorf("global manager shard connection failed: %w", err)
		}
	} else {
		return fmt.Errorf("global manager URL is required")
	}

	for i, url := range userURLs {
		if url == "" {
			continue
		}
		db, err := ConnectTurso(url, userTokens[i])
		if err != nil {
			log.Printf("⚠️ User shard %d connection failed: %v", i, err)
			continue
		}
		router.userDBs = append(router.userDBs, db)
	}

	router.coreAuthDB = router.authDB
	router.coreAnalyticsDB = router.analyticsDB
	router.coreGlobalManagerDB = router.globalManagerDB
	router.coreUserDBs = append([]*sql.DB{}, router.userDBs...)
	router.managedDBs = make(map[string]ManagedDB)

	Router = router
	log.Printf("🧠 Shard Router Ready: [Auth, Analytics, GlobalManager, %d User Shards]", len(router.userDBs))
	return nil
}

func (sr *ShardRouter) GetAuthDB() *sql.DB             { return sr.authDB }
func (sr *ShardRouter) GetAnalyticsDB() *sql.DB        { return sr.analyticsDB }
func (sr *ShardRouter) GetGlobalManagerDB() *sql.DB    { return sr.globalManagerDB }

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
	sr.globalManagerDB = sr.coreGlobalManagerDB
	sr.userDBs = append([]*sql.DB{}, sr.coreUserDBs...)
	newManaged := make(map[string]ManagedDB)
	for _, mdb := range dbs {
		newManaged[mdb.Slug] = mdb
		switch mdb.Category {
		case "auth":
			sr.authDB = mdb.DB
		case "analytics":
			sr.analyticsDB = mdb.DB
		case "global":
			sr.globalManagerDB = mdb.DB
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
		return sr.coreGlobalManagerDB
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
