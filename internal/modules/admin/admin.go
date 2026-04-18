package admin

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/bandhannova/api-hunter/internal/middleware"
	"github.com/bandhannova/api-hunter/internal/modules/ai"
	"github.com/bandhannova/api-hunter/internal/modules/email"
	"github.com/bandhannova/api-hunter/internal/modules/market"
	"github.com/bandhannova/api-hunter/internal/modules/search"
	"github.com/bandhannova/api-hunter/internal/proxy"
	"github.com/bandhannova/api-hunter/internal/security"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ═══════════════════════════════════════════════
//  Admin Handler — BandhanNova Command Center API
// ═══════════════════════════════════════════════

// InitAdminHandlers initializes admin-specific state
func InitAdminHandlers() {
	log.Println("🛡️  Admin Command Center handlers initialized")
}

// ─── LOGIN ───────────────────────────────────

type AdminLoginRequest struct {
	MasterKey string `json:"master_key"`
}

// AdminLogin verifies the Master Key and returns an HMAC session token
func AdminLogin(c *fiber.Ctx) error {
	ip, _ := c.Locals("client_ip").(string)
	if ip == "" {
		ip = c.IP()
	}

	var req AdminLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid request body",
		})
	}

	if req.MasterKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Master key is required",
		})
	}

	// Verify Master Key
	if req.MasterKey != config.AppConfig.BandhanNovaMasterKey {
		remaining, banned := middleware.RecordFailedLogin(ip)
		LogAudit("LOGIN_FAILED", "admin", ip, "Invalid master key attempt")

		if banned {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   true,
				"message": "IP banned for 1 hour due to excessive failed attempts",
				"banned":  true,
			})
		}

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":            true,
			"message":          "Invalid master key",
			"attempts_remaining": remaining,
		})
	}

	// Success — generate session token
	middleware.ResetFailedLogins(ip)
	token := middleware.GenerateAdminToken(config.AppConfig.BandhanNovaMasterKey)
	LogAudit("LOGIN_SUCCESS", "admin", ip, "Admin session started")

	return c.JSON(fiber.Map{
		"success": true,
		"token":   token,
		"expires": time.Now().Add(24 * time.Hour).Unix(),
	})
}

// ─── API KEYS CRUD ───────────────────────────

type ManagedKeyResponse struct {
	ID        string `json:"id"`
	Provider  string `json:"provider"`
	Label     string `json:"label"`
	Status    string `json:"status"`
	MaskedKey string `json:"masked_key"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type AddKeyRequest struct {
	Provider string `json:"provider"`
	KeyValue string `json:"key_value"`
	Label    string `json:"label"`
}

// ListManagedKeys returns all managed API keys (values masked)
func ListManagedKeys(c *fiber.Ctx) error {
	ip, _ := c.Locals("admin_ip").(string)

	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   true,
			"message": "Database not connected",
		})
	}

	rows, err := database.Router.GetGlobalManagerDB().Query(
		"SELECT id, provider, encrypted_value, label, status, created_at, updated_at FROM managed_api_keys ORDER BY provider, created_at DESC",
	)
	if err != nil {
		log.Printf("Admin: Failed to list keys: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to fetch keys",
		})
	}
	defer rows.Close()

	var keys []ManagedKeyResponse
	for rows.Next() {
		var id, provider, encrypted, label, status string
		var createdAt, updatedAt int64
		if err := rows.Scan(&id, &provider, &encrypted, &label, &status, &createdAt, &updatedAt); err != nil {
			continue
		}

		// Decrypt value to create masked version
		masked := "***encrypted***"
		decrypted, err := security.Decrypt(encrypted, config.AppConfig.BandhanNovaMasterKey)
		if err == nil && len(decrypted) > 8 {
			masked = decrypted[:4] + "..." + decrypted[len(decrypted)-4:]
		} else if err == nil && len(decrypted) > 0 {
			masked = decrypted[:1] + "***"
		}

		keys = append(keys, ManagedKeyResponse{
			ID:        id,
			Provider:  provider,
			Label:     label,
			Status:    status,
			MaskedKey: masked,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}

	if keys == nil {
		keys = []ManagedKeyResponse{}
	}

	LogAudit("LIST_KEYS", "managed_api_keys", ip, fmt.Sprintf("Listed %d keys", len(keys)))

	return c.JSON(fiber.Map{
		"success": true,
		"keys":    keys,
		"count":   len(keys),
	})
}

// AddManagedKey adds a new API key (encrypted at rest)
func AddManagedKey(c *fiber.Ctx) error {
	ip, _ := c.Locals("admin_ip").(string)

	var req AddKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid request body",
		})
	}

	// Validate
	validProviders := map[string]bool{
		"OpenRouter": true, "Cerebras": true, "Groq": true,
		"Tavily": true, "Resend": true, "TwelveData": true,
	}
	if !validProviders[req.Provider] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid provider. Must be one of: OpenRouter, Cerebras, Groq, Tavily, Resend, TwelveData",
		})
	}

	if req.KeyValue == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Key value is required",
		})
	}

	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   true,
			"message": "Database not connected",
		})
	}

	// Encrypt the key value
	encrypted, err := security.Encrypt(req.KeyValue, config.AppConfig.BandhanNovaMasterKey)
	if err != nil {
		log.Printf("Admin: Failed to encrypt key: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Encryption failed",
		})
	}

	id := uuid.New().String()
	now := time.Now().Unix()

	_, err = database.Router.GetGlobalManagerDB().Exec(
		"INSERT INTO managed_api_keys (id, provider, encrypted_value, label, status, created_at, updated_at) VALUES (?, ?, ?, ?, 'active', ?, ?)",
		id, req.Provider, encrypted, req.Label, now, now,
	)
	if err != nil {
		log.Printf("Admin: Failed to insert key: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to save key",
		})
	}

	masked := req.KeyValue[:4] + "..." + req.KeyValue[len(req.KeyValue)-4:]
	LogAudit("ADD_KEY", req.Provider, ip, fmt.Sprintf("Added key: %s (%s)", masked, req.Label))

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"key": ManagedKeyResponse{
			ID:        id,
			Provider:  req.Provider,
			Label:     req.Label,
			Status:    "active",
			MaskedKey: masked,
			CreatedAt: now,
			UpdatedAt: now,
		},
	})
}

// DeleteManagedKey removes an API key
func DeleteManagedKey(c *fiber.Ctx) error {
	ip, _ := c.Locals("admin_ip").(string)
	keyID := c.Params("id")

	if keyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Key ID is required",
		})
	}

	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   true,
			"message": "Database not connected",
		})
	}

	// Get key info before deletion for audit
	var provider, label string
	err := database.Router.GetGlobalManagerDB().QueryRow(
		"SELECT provider, label FROM managed_api_keys WHERE id = ?", keyID,
	).Scan(&provider, &label)

	if err == sql.ErrNoRows {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   true,
			"message": "Key not found",
		})
	}

	// Delete
	_, err = database.Router.GetGlobalManagerDB().Exec(
		"DELETE FROM managed_api_keys WHERE id = ?", keyID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to delete key",
		})
	}

	LogAudit("DELETE_KEY", provider, ip, fmt.Sprintf("Deleted key %s (%s)", keyID[:8], label))

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Key deleted successfully",
	})
}

// CheckKeyHealth tests if an API key is valid by making a lightweight request to the provider
func CheckKeyHealth(c *fiber.Ctx) error {
	ip, _ := c.Locals("admin_ip").(string)
	keyID := c.Params("id")

	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   true,
			"message": "Database not connected",
		})
	}

	var provider, encrypted string
	err := database.Router.GetGlobalManagerDB().QueryRow(
		"SELECT provider, encrypted_value FROM managed_api_keys WHERE id = ?", keyID,
	).Scan(&provider, &encrypted)

	if err == sql.ErrNoRows {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   true,
			"message": "Key not found",
		})
	}

	// Decrypt the key
	keyValue, err := security.Decrypt(encrypted, config.AppConfig.BandhanNovaMasterKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to decrypt key",
		})
	}

	// Test the key based on provider
	status, details := testProviderKey(provider, keyValue)

	// Update status in DB
	newStatus := "active"
	if status != "healthy" {
		newStatus = status
	}
	_, _ = database.Router.GetGlobalManagerDB().Exec(
		"UPDATE managed_api_keys SET status = ?, updated_at = ? WHERE id = ?",
		newStatus, time.Now().Unix(), keyID,
	)

	LogAudit("CHECK_KEY", provider, ip, fmt.Sprintf("Health check: %s - %s", status, details))

	return c.JSON(fiber.Map{
		"success": true,
		"status":  status,
		"details": details,
	})
}

// testProviderKey makes a lightweight API call to verify key validity
func testProviderKey(provider, keyValue string) (string, string) {
	// Simple validation based on key prefix patterns
	switch provider {
	case "OpenRouter":
		if !strings.HasPrefix(keyValue, "sk-or-") {
			return "invalid", "Key does not match OpenRouter format (sk-or-...)"
		}
		return "healthy", "Key format valid (sk-or-...)"
	case "Cerebras":
		if !strings.HasPrefix(keyValue, "csk-") {
			return "invalid", "Key does not match Cerebras format (csk-...)"
		}
		return "healthy", "Key format valid (csk-...)"
	case "Groq":
		if !strings.HasPrefix(keyValue, "gsk_") {
			return "invalid", "Key does not match Groq format (gsk_...)"
		}
		return "healthy", "Key format valid (gsk_...)"
	case "Tavily":
		if !strings.HasPrefix(keyValue, "tvly-") {
			return "invalid", "Key does not match Tavily format (tvly-...)"
		}
		return "healthy", "Key format valid (tvly-...)"
	case "Resend":
		if !strings.HasPrefix(keyValue, "re_") {
			return "invalid", "Key does not match Resend format (re_...)"
		}
		return "healthy", "Key format valid (re_...)"
	case "TwelveData":
		if len(keyValue) < 10 {
			return "invalid", "Key too short for TwelveData"
		}
		return "healthy", "Key format valid"
	default:
		return "unknown", "Unknown provider"
	}
}

// ReloadKeys hot-reloads all managed keys from the database into active KeyManagers
func ReloadKeys(c *fiber.Ctx) error {
	ip, _ := c.Locals("admin_ip").(string)

	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   true,
			"message": "Database not connected",
		})
	}

	rows, err := database.Router.GetGlobalManagerDB().Query(
		"SELECT provider, encrypted_value FROM managed_api_keys WHERE status = 'active' ORDER BY provider",
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to load keys from database",
		})
	}
	defer rows.Close()

	// Group keys by provider
	providerKeys := make(map[string][]string)
	for rows.Next() {
		var provider, encrypted string
		if err := rows.Scan(&provider, &encrypted); err != nil {
			continue
		}
		decrypted, err := security.Decrypt(encrypted, config.AppConfig.BandhanNovaMasterKey)
		if err != nil {
			log.Printf("Admin: Failed to decrypt key for provider %s: %v", provider, err)
			continue
		}
		providerKeys[provider] = append(providerKeys[provider], decrypted)
	}

	// Rebuild KeyManagers with combined keys (existing env + managed DB keys)
	reloadCount := 0
	for provider, keys := range providerKeys {
		switch provider {
		case "OpenRouter":
			combined := append(config.AppConfig.OpenRouterKeys, keys...)
			ai.OpenRouterKM = proxy.NewKeyManager(removeDuplicates(combined), "OpenRouter")
			reloadCount += len(keys)
		case "Cerebras":
			combined := append(config.AppConfig.CerebrasKeys, keys...)
			ai.CerebrasKM = proxy.NewKeyManager(removeDuplicates(combined), "Cerebras")
			reloadCount += len(keys)
		case "Groq":
			combined := append(config.AppConfig.GroqKeys, keys...)
			ai.GroqKM = proxy.NewKeyManager(removeDuplicates(combined), "Groq")
			reloadCount += len(keys)
		case "Tavily":
			combined := append(config.AppConfig.TavilyKeys, keys...)
			search.TavilyKM = proxy.NewKeyManager(removeDuplicates(combined), "Tavily")
			reloadCount += len(keys)
		case "Resend":
			combined := append(config.AppConfig.ResendKeys, keys...)
			email.ResendKM = proxy.NewKeyManager(removeDuplicates(combined), "Resend")
			reloadCount += len(keys)
		case "TwelveData":
			combined := append(config.AppConfig.TwelveDataKeys, keys...)
			market.TwelveDataKM = proxy.NewKeyManager(removeDuplicates(combined), "TwelveData")
			reloadCount += len(keys)
		}
	}

	LogAudit("RELOAD_KEYS", "all", ip, fmt.Sprintf("Reloaded %d managed keys across %d providers", reloadCount, len(providerKeys)))

	return c.JSON(fiber.Map{
		"success":         true,
		"message":         "Keys reloaded successfully",
		"managed_keys":    reloadCount,
		"providers_count": len(providerKeys),
	})
}

func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// ─── API STATUS (Enhanced) ───────────────────

// ShardStatus represents the health and metrics of a database shard
type ShardStatus struct {
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	Status    string         `json:"status"`
	Storage   int64          `json:"storage"`
	RowCounts map[string]int `json:"row_counts"`
}

// DashboardData represents the combined data for the UI
type DashboardData struct {
	Keys       []proxy.KeyMetadata `json:"keys"`
	Logs       []proxy.RequestLog  `json:"logs"`
	Emails     []fiber.Map         `json:"emails"`
	Shards     []ShardStatus       `json:"shards"`
	Timeline struct {
		Success int `json:"success"`
		Failed  int `json:"failed"`
	} `json:"timeline"`
}

// GetAdminStatus returns comprehensive system status for the admin dashboard
func GetAdminStatus(c *fiber.Ctx) error {
	data := DashboardData{
		Keys: append(append(append(append(append(ai.OpenRouterKM.GetKeys(), search.TavilyKM.GetKeys()...), ai.GroqKM.GetKeys()...), email.ResendKM.GetKeys()...), ai.CerebrasKM.GetKeys()...), market.TwelveDataKM.GetKeys()...),
	}

	proxy.LogsMutex.Lock()
	data.Logs = make([]proxy.RequestLog, len(proxy.GlobalRequestLogs))
	copy(data.Logs, proxy.GlobalRequestLogs)
	proxy.LogsMutex.Unlock()

	for _, log := range data.Logs {
		if log.StatusCode >= 200 && log.StatusCode < 400 {
			data.Timeline.Success++
		} else {
			data.Timeline.Failed++
		}
	}

	// Shard health
	data.Shards = []ShardStatus{}
	if database.Router != nil {
		data.Shards = append(data.Shards, getShardMetrics(database.Router.GetAuthDB(), "Auth Shard", "auth", []string{"users", "sessions"}))
		data.Shards = append(data.Shards, getShardMetrics(database.Router.GetAnalyticsDB(), "Analytics Shard", "analytics", []string{"request_logs", "inbound_emails", "outbound_emails"}))
		data.Shards = append(data.Shards, getShardMetrics(database.Router.GetGlobalManagerDB(), "Global Shard", "global", []string{"app_config", "feature_flags", "managed_api_keys", "admin_audit_log"}))
		for i := 0; i < database.Router.GetShardCount(); i++ {
			shardName := fmt.Sprintf("User Shard %d", i)
			db := database.Router.GetUserDB(fmt.Sprintf("shard-%d", i))
			data.Shards = append(data.Shards, getShardMetrics(db, shardName, "user", []string{"user_data", "chat_history", "saved_items"}))
		}
	}

	// Email data
	if database.Router != nil {
		rows, err := database.Router.GetAnalyticsDB().Query(
			"SELECT from_email, subject, timestamp FROM inbound_emails ORDER BY timestamp DESC LIMIT 20",
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var from, sub string
				var ts int64
				rows.Scan(&from, &sub, &ts)
				data.Emails = append(data.Emails, fiber.Map{
					"from":      from,
					"subject":   sub,
					"timestamp": ts,
				})
			}
		}
	}

	if data.Emails == nil {
		data.Emails = []fiber.Map{}
	}

	return c.JSON(data)
}

func getShardMetrics(db *sql.DB, name, shardType string, tables []string) ShardStatus {
	stats := ShardStatus{
		Name:      name,
		Type:      shardType,
		Status:    "Healthy",
		RowCounts: make(map[string]int),
	}

	if db == nil {
		stats.Status = "Disconnected"
		return stats
	}

	// Check Health
	if err := db.Ping(); err != nil {
		stats.Status = "Offline"
		return stats
	}

	// Get Storage Usage (PRAGMA)
	var pageCount, pageSize int64
	_ = db.QueryRow("PRAGMA page_count").Scan(&pageCount)
	_ = db.QueryRow("PRAGMA page_size").Scan(&pageSize)
	stats.Storage = pageCount * pageSize

	// Get Row Counts (Gently)
	for _, table := range tables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		err := db.QueryRow(query).Scan(&count)
		if err != nil {
			stats.RowCounts[table] = 0 // Table might not exist yet
		} else {
			stats.RowCounts[table] = count
		}
	}

	return stats
}

// ─── AUDIT LOG ───────────────────────────────

type AuditLogEntry struct {
	ID        string `json:"id"`
	Action    string `json:"action"`
	Target    string `json:"target"`
	IPAddress string `json:"ip_address"`
	Details   string `json:"details"`
	Timestamp int64  `json:"timestamp"`
}

// GetAuditLog returns paginated admin audit log
func GetAuditLog(c *fiber.Ctx) error {
	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   true,
			"message": "Database not connected",
		})
	}

	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	if limit > 200 {
		limit = 200
	}

	rows, err := database.Router.GetGlobalManagerDB().Query(
		"SELECT id, action, target, ip_address, details, timestamp FROM admin_audit_log ORDER BY timestamp DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to fetch audit log",
		})
	}
	defer rows.Close()

	var logs []AuditLogEntry
	for rows.Next() {
		var entry AuditLogEntry
		if err := rows.Scan(&entry.ID, &entry.Action, &entry.Target, &entry.IPAddress, &entry.Details, &entry.Timestamp); err != nil {
			continue
		}
		logs = append(logs, entry)
	}

	if logs == nil {
		logs = []AuditLogEntry{}
	}

	// Get total count
	var total int
	_ = database.Router.GetGlobalManagerDB().QueryRow("SELECT COUNT(*) FROM admin_audit_log").Scan(&total)

	return c.JSON(fiber.Map{
		"success": true,
		"logs":    logs,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// ─── HELPERS ─────────────────────────────────

// LogAudit records an admin action to the audit log
func LogAudit(action, target, ip, details string) {
	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		log.Printf("AUDIT [%s] %s | target=%s | ip=%s | %s", time.Now().Format(time.RFC3339), action, target, ip, details)
		return
	}

	id := uuid.New().String()
	_, err := database.Router.GetGlobalManagerDB().Exec(
		"INSERT INTO admin_audit_log (id, action, target, ip_address, details, timestamp) VALUES (?, ?, ?, ?, ?, ?)",
		id, action, target, ip, details, time.Now().Unix(),
	)
	if err != nil {
		log.Printf("AUDIT LOG WRITE FAILED: %v | action=%s target=%s", err, action, target)
	}
}
