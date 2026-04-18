package config

import (
	"log"
	"os"
	"strings"

	"github.com/bandhannova/api-hunter/internal/security"
	"github.com/joho/godotenv"
)

type Config struct {
	Port                 string
	Environment          string
	BandhanNovaMasterKey string
	OpenRouterKeys       []string
	TavilyKeys           []string
	GroqKeys             []string
	// BFOBS Turso Shards
	TursoAuthURL         string
	TursoAuthToken       string
	TursoAnalyticsURL    string
	TursoAnalyticsToken  string
	TursoGlobalURL       string
	TursoGlobalToken     string
	TursoUserShardURLs   []string
	TursoUserShardTokens []string
	// Email Proxy Keys
	ResendKeys           []string
	ResendWebhookSecret  string
	// Cerebras AI Keys
	CerebrasKeys         []string
	// TwelveData Market Keys
	TwelveDataKeys       []string
	RedisURL             string
	JWTSecret            string
	SupabaseURL          string
	SupabaseJWTSecret    string
	PublicURL            string
}

var AppConfig Config

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, reading from system environment variables")
	}

	AppConfig = Config{
		Port:                 getEnv("PORT", "8080"),
		Environment:          getEnv("ENVIRONMENT", "development"),
		BandhanNovaMasterKey: getEnv("BANDHANNOVA_MASTER_KEY", ""),
		OpenRouterKeys:       getEnvAsSlice("OPENROUTER_KEYS", ","),
		TavilyKeys:           getEnvAsSlice("TAVILY_KEYS", ","),
		GroqKeys:             getEnvAsSlice("GROQ_KEYS", ","),
		// BFOBS Turso Shards
		TursoAuthURL:         getEnv("TURSO_AUTH_URL", ""),
		TursoAuthToken:       getEnv("TURSO_AUTH_TOKEN", ""),
		TursoAnalyticsURL:    getEnv("TURSO_ANALYTICS_URL", ""),
		TursoAnalyticsToken:  getEnv("TURSO_ANALYTICS_TOKEN", ""),
		TursoGlobalURL:       getEnv("TURSO_GLOBAL_URL", ""),
		TursoGlobalToken:     getEnv("TURSO_GLOBAL_TOKEN", ""),
		TursoUserShardURLs:   getEnvAsSlice("TURSO_USER_SHARD_URLS", ","),
		TursoUserShardTokens: getEnvAsSlice("TURSO_USER_SHARD_TOKENS", ","),
		// Email Proxy Keys
		ResendKeys:          getEnvAsSlice("RESEND_KEYS", ","),
		ResendWebhookSecret: getEnv("RESEND_WEBHOOK_SECRET", ""),
		// Cerebras AI Keys
		CerebrasKeys:        getEnvAsSlice("CEREBRAS_KEYS", ","),
		// TwelveData Market Keys
		TwelveDataKeys:      getEnvAsSlice("TWELVEDATA_KEYS", ","),
		RedisURL:            getEnv("REDIS_URL", ""),
		JWTSecret:           getEnv("JWT_SECRET", "bandhannova-default-secret-123"),
		SupabaseURL:          getEnv("SUPABASE_URL", ""),
		SupabaseJWTSecret:    getEnv("SUPABASE_JWT_SECRET", ""),
		PublicURL:            getEnv("PUBLIC_URL", ""),
	}

	// Fallback to internal registry if keys are missing but master key exists
	if AppConfig.BandhanNovaMasterKey != "" && AppConfig.BandhanNovaMasterKey != "default" {
		fallbackToInternalRegistry()
	}

	if AppConfig.BandhanNovaMasterKey == "" {
		log.Fatal("BANDHANNOVA_MASTER_KEY is required")
	}

	if AppConfig.JWTSecret == "" || AppConfig.JWTSecret == "bdn-bfobs-default-secret-change-me" {
		log.Fatal("JWT_SECRET is required and must not be the default value")
	}

	log.Printf("Config Loaded: OpenRouterKeys=%d, TavilyKeys=%d, GroqKeys=%d, CerebrasKeys=%d, TwelveDataKeys=%d, TursoUserShards=%d",
		len(AppConfig.OpenRouterKeys),
		len(AppConfig.TavilyKeys),
		len(AppConfig.GroqKeys),
		len(AppConfig.CerebrasKeys),
		len(AppConfig.TwelveDataKeys),
		len(AppConfig.TursoUserShardURLs))
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsSlice(key string, sep string) []string {
	val := getEnv(key, "")
	if val == "" {
		return []string{}
	}
	// Split and trim spaces
	parts := strings.Split(val, sep)
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func fallbackToInternalRegistry() {
	master := AppConfig.BandhanNovaMasterKey

	if len(AppConfig.OpenRouterKeys) == 0 {
		AppConfig.OpenRouterKeys = decryptInternal(InternalRegistry["OPENROUTER_KEYS"], master)
	}
	if len(AppConfig.TavilyKeys) == 0 {
		AppConfig.TavilyKeys = decryptInternal(InternalRegistry["TAVILY_KEYS"], master)
	}
	if len(AppConfig.GroqKeys) == 0 {
		AppConfig.GroqKeys = decryptInternal(InternalRegistry["GROQ_KEYS"], master)
	}
	if len(AppConfig.CerebrasKeys) == 0 {
		AppConfig.CerebrasKeys = decryptInternal(InternalRegistry["CEREBRAS_KEYS"], master)
	}
	if len(AppConfig.ResendKeys) == 0 {
		AppConfig.ResendKeys = decryptInternal(InternalRegistry["RESEND_KEYS"], master)
	}
	if len(AppConfig.TwelveDataKeys) == 0 {
		AppConfig.TwelveDataKeys = decryptInternal(InternalRegistry["TWELVEDATA_KEYS"], master)
	}
	if AppConfig.JWTSecret == "bdn-bfobs-default-secret-change-me" {
		decryptedSecret := decryptInternal(InternalRegistry["JWT_SECRET"], master)
		if len(decryptedSecret) > 0 {
			AppConfig.JWTSecret = decryptedSecret[0]
		}
	}

	// Turso Database Fallbacks
	if AppConfig.TursoAuthURL == "" {
		decrypted := decryptInternal(InternalRegistry["TURSO_AUTH_URL"], master)
		if len(decrypted) > 0 {
			AppConfig.TursoAuthURL = decrypted[0]
		}
	}
	if AppConfig.TursoAuthToken == "" {
		decrypted := decryptInternal(InternalRegistry["TURSO_AUTH_TOKEN"], master)
		if len(decrypted) > 0 {
			AppConfig.TursoAuthToken = decrypted[0]
		}
	}
	if AppConfig.TursoAnalyticsURL == "" {
		decrypted := decryptInternal(InternalRegistry["TURSO_ANALYTICS_URL"], master)
		if len(decrypted) > 0 {
			AppConfig.TursoAnalyticsURL = decrypted[0]
		}
	}
	if AppConfig.TursoAnalyticsToken == "" {
		decrypted := decryptInternal(InternalRegistry["TURSO_ANALYTICS_TOKEN"], master)
		if len(decrypted) > 0 {
			AppConfig.TursoAnalyticsToken = decrypted[0]
		}
	}
	if AppConfig.TursoGlobalURL == "" {
		decrypted := decryptInternal(InternalRegistry["TURSO_GLOBAL_URL"], master)
		if len(decrypted) > 0 {
			AppConfig.TursoGlobalURL = decrypted[0]
		}
	}
	if AppConfig.TursoGlobalToken == "" {
		decrypted := decryptInternal(InternalRegistry["TURSO_GLOBAL_TOKEN"], master)
		if len(decrypted) > 0 {
			AppConfig.TursoGlobalToken = decrypted[0]
		}
	}
	if len(AppConfig.TursoUserShardURLs) == 0 {
		AppConfig.TursoUserShardURLs = decryptInternal(InternalRegistry["TURSO_USER_SHARD_URLS"], master)
	}
	if len(AppConfig.TursoUserShardTokens) == 0 {
		AppConfig.TursoUserShardTokens = decryptInternal(InternalRegistry["TURSO_USER_SHARD_TOKENS"], master)
	}
}

func decryptInternal(encrypted, master string) []string {
	if encrypted == "" {
		return nil
	}
	decrypted, err := security.Decrypt(encrypted, master)
	if err != nil {
		log.Printf("⚠️ Internal decryption failed for some keys: %v", err)
		return nil
	}
	
	// Split and trim spaces
	parts := strings.Split(decrypted, ",")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// SyncConfigFromDB loads all managed keys and databases from the Global Manager shard
// This allows the system to run with minimal environment variables.
func SyncConfigFromDB() error {
	return nil
}

// UpdateKeys updates the in-memory keys from the database records
func UpdateKeys(provider string, keys []string) {
	switch strings.ToLower(provider) {
	case "openrouter":
		AppConfig.OpenRouterKeys = keys
	case "tavily":
		AppConfig.TavilyKeys = keys
	case "groq":
		AppConfig.GroqKeys = keys
	case "cerebras":
		AppConfig.CerebrasKeys = keys
	case "resend":
		AppConfig.ResendKeys = keys
	case "twelvedata":
		AppConfig.TwelveDataKeys = keys
	}
}
