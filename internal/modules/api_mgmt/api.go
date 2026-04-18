package api_mgmt

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/bandhannova/api-hunter/internal/modules/admin"
	"github.com/bandhannova/api-hunter/internal/security"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type APISection struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt int64     `json:"created_at"`
	CardCount int       `json:"card_count"`
}

type APICard struct {
	ID           string `json:"id"`
	SectionID    string `json:"section_id"`
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	EndpointURL  string `json:"endpoint_url"`
	PlatformType string `json:"platform_type"`
	RateLimit    int    `json:"rate_limit"`
	KeyCount     int    `json:"key_count"`
	TotalReq     int    `json:"total_req"`
	IsDeleted    bool   `json:"is_deleted"`
	CreatedAt    int64  `json:"created_at"`
}

type APIKeyResponse struct {
	ID            string `json:"id"`
	CardID        string `json:"card_id"`
	Label         string `json:"label"`
	Status        string `json:"status"`
	MaskedValue   string `json:"masked_value"`
	APIURL        string `json:"api_url"`
	UseURL        bool   `json:"use_url"`
	IsDeleted     bool   `json:"is_deleted"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

// ─── SECTIONS ────────────────────────────────────────────────────────────────

func ListSections(c *fiber.Ctx) error {
	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return c.Status(503).JSON(fiber.Map{"error": true, "message": "Database not ready"})
	}
	rows, err := database.Router.GetGlobalManagerDB().Query(`
		SELECT s.id, s.name, s.created_at, COUNT(c.id) as card_count
		FROM api_sections s
		LEFT JOIN api_cards c ON s.id = c.section_id AND c.is_deleted = 0
		GROUP BY s.id
		ORDER BY s.created_at ASC
	`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to list sections"})
	}
	defer rows.Close()

	var sections []APISection
	for rows.Next() {
		var s APISection
		rows.Scan(&s.ID, &s.Name, &s.CreatedAt, &s.CardCount)
		sections = append(sections, s)
	}
	return c.JSON(fiber.Map{"success": true, "sections": sections})
}

func AddSection(c *fiber.Ctx) error {
	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return c.Status(503).JSON(fiber.Map{"error": true, "message": "Database not ready"})
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid request"})
	}

	id := strings.ToLower(strings.ReplaceAll(body.Name, " ", "_")) + "_" + uuid.New().String()[:4]
	_, err := database.Router.GetGlobalManagerDB().Exec(
		"INSERT INTO api_sections (id, name, created_at) VALUES (?, ?, ?)",
		id, body.Name, time.Now().Unix(),
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to create section"})
	}
	return c.JSON(fiber.Map{"success": true, "id": id})
}

// ─── CARDS ───────────────────────────────────────────────────────────────────

func ListCards(c *fiber.Ctx) error {
	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return c.Status(503).JSON(fiber.Map{"error": true, "message": "Database not ready"})
	}
	sectionID := c.Query("section_id")
	query := "SELECT id, section_id, name, icon, endpoint_url, platform_type, rate_limit, is_deleted, created_at FROM api_cards WHERE is_deleted = 0"
	var args []interface{}

	if sectionID != "" {
		query += " AND section_id = ?"
		args = append(args, sectionID)
	}

	rows, err := database.Router.GetGlobalManagerDB().Query(query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to list cards"})
	}
	defer rows.Close()

	var cards []APICard
	for rows.Next() {
		var cd APICard
		var isDel int
		err := rows.Scan(
			&cd.ID, 
			&cd.SectionID, 
			&cd.Name, 
			&cd.Icon, 
			&cd.EndpointURL, 
			&cd.PlatformType, 
			&cd.RateLimit,
			&isDel, 
			&cd.CreatedAt,
		)
		if err != nil {
			log.Printf("⚠️  Error scanning card: %v", err)
			continue
		}
		cd.IsDeleted = isDel == 1
		
		// Get key count
		_ = database.Router.GetGlobalManagerDB().QueryRow("SELECT COUNT(*) FROM managed_api_keys WHERE card_id = ? AND is_deleted = 0", cd.ID).Scan(&cd.KeyCount)
		
		// Get total usage from logs
		_ = database.Router.GetGlobalManagerDB().QueryRow("SELECT COUNT(*) FROM api_usage_logs WHERE card_id = ?", cd.ID).Scan(&cd.TotalReq)

		cards = append(cards, cd)
	}
	return c.JSON(fiber.Map{"success": true, "cards": cards})
}

func AddCard(c *fiber.Ctx) error {
	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return c.Status(503).JSON(fiber.Map{"error": true, "message": "Database not ready"})
	}
	var body struct {
		SectionID    string `json:"section_id"`
		Name         string `json:"name"`
		Icon         string `json:"icon"`
		EndpointURL  string `json:"endpoint_url"`
		PlatformType string `json:"platform_type"`
		RateLimit    int    `json:"rate_limit"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid request"})
	}

	id := uuid.New().String()
	_, err := database.Router.GetGlobalManagerDB().Exec(
		"INSERT INTO api_cards (id, section_id, name, icon, endpoint_url, platform_type, rate_limit, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, body.SectionID, body.Name, body.Icon, body.EndpointURL, body.PlatformType, body.RateLimit, time.Now().Unix(),
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to create card"})
	}
	return c.JSON(fiber.Map{"success": true, "id": id})
}

func UpdateCard(c *fiber.Ctx) error {
	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return c.Status(503).JSON(fiber.Map{"error": true, "message": "Database not ready"})
	}
	id := c.Params("id")
	var body struct {
		Name         string `json:"name"`
		Icon         string `json:"icon"`
		EndpointURL  string `json:"endpoint_url"`
		PlatformType string `json:"platform_type"`
		RateLimit    int    `json:"rate_limit"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid request"})
	}

	_, err := database.Router.GetGlobalManagerDB().Exec(`
		UPDATE api_cards 
		SET name = ?, icon = ?, endpoint_url = ?, platform_type = ?, rate_limit = ?
		WHERE id = ?
	`, body.Name, body.Icon, body.EndpointURL, body.PlatformType, body.RateLimit, id)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to update card"})
	}
	return c.JSON(fiber.Map{"success": true})
}

// ─── KEYS ────────────────────────────────────────────────────────────────────

func ListKeys(c *fiber.Ctx) error {
	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return c.Status(503).JSON(fiber.Map{"error": true, "message": "Database not ready"})
	}
	cardID := c.Query("card_id")
	if cardID == "" {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Card ID required"})
	}

	rows, err := database.Router.GetGlobalManagerDB().Query(`
		SELECT id, card_id, label, status, encrypted_value, api_url, use_url, is_deleted, created_at, updated_at 
		FROM managed_api_keys 
		WHERE card_id = ? AND is_deleted = 0
	`, cardID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to list keys"})
	}
	defer rows.Close()

	var keys []APIKeyResponse
	for rows.Next() {
		var k APIKeyResponse
		var encrypted string
		var useURL, isDel int
		rows.Scan(&k.ID, &k.CardID, &k.Label, &k.Status, &encrypted, &k.APIURL, &useURL, &isDel, &k.CreatedAt, &k.UpdatedAt)
		k.UseURL = useURL == 1
		k.IsDeleted = isDel == 1

		// Mask Key
		decrypted, _ := security.Decrypt(encrypted, config.AppConfig.BandhanNovaMasterKey)
		if len(decrypted) > 8 {
			k.MaskedValue = decrypted[:4] + "..." + decrypted[len(decrypted)-4:]
		} else {
			k.MaskedValue = "***"
		}

		keys = append(keys, k)
	}
	return c.JSON(fiber.Map{"success": true, "keys": keys})
}

func AddKey(c *fiber.Ctx) error {
	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return c.Status(503).JSON(fiber.Map{"error": true, "message": "Database not ready"})
	}
	var body struct {
		CardID   string   `json:"card_id"`
		Label    string   `json:"label"`
		Value    string   `json:"value"`
		Values   []string `json:"values"` // For bulk insert
		APIURL   string   `json:"api_url"`
		UseURL   bool     `json:"use_url"`
		Provider string   `json:"provider"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid request"})
	}

	now := time.Now().Unix()
	useURLInt := 0
	if body.UseURL { useURLInt = 1 }

	// Handle Bulk Insert
	if len(body.Values) > 0 {
		tx, err := database.Router.GetGlobalManagerDB().Begin()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": true, "message": "Transaction failed"})
		}
		for i, val := range body.Values {
			if strings.TrimSpace(val) == "" { continue }
			encrypted, _ := security.Encrypt(val, config.AppConfig.BandhanNovaMasterKey)
			label := fmt.Sprintf("%s #%d", body.Label, i+1)
			id := uuid.New().String()
			_, _ = tx.Exec(`
				INSERT INTO managed_api_keys (id, card_id, provider, encrypted_value, label, api_url, use_url, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, id, body.CardID, body.Provider, encrypted, label, body.APIURL, useURLInt, now, now)
		}
		tx.Commit()
		return c.JSON(fiber.Map{"success": true, "message": "Bulk keys saved"})
	}

	// Handle Single Insert
	encrypted, err := security.Encrypt(body.Value, config.AppConfig.BandhanNovaMasterKey)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Encryption failed"})
	}

	id := uuid.New().String()
	_, err = database.Router.GetGlobalManagerDB().Exec(`
		INSERT INTO managed_api_keys (id, card_id, provider, encrypted_value, label, api_url, use_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, body.CardID, body.Provider, encrypted, body.Label, body.APIURL, useURLInt, now, now)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to save key"})
	}
	return c.JSON(fiber.Map{"success": true, "id": id})
}

// ─── DELETION & SOFT DELETE ──────────────────────────────────────────────────

func DeleteAPIItem(c *fiber.Ctx) error {
	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return c.Status(503).JSON(fiber.Map{"error": true, "message": "Database not ready"})
	}
	itemType := c.Params("type") // section, card, key
	id := c.Params("id")
	ip, _ := c.Locals("admin_ip").(string)

	var err error
	switch itemType {
	case "card":
		_, err = database.Router.GetGlobalManagerDB().Exec("UPDATE api_cards SET is_deleted = 1, section_id = 'unused' WHERE id = ?", id)
	case "key":
		_, err = database.Router.GetGlobalManagerDB().Exec("UPDATE managed_api_keys SET is_deleted = 1 WHERE id = ?", id)
	default:
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid item type"})
	}

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to move to unused"})
	}

	admin.LogAudit("SOFT_DELETE_API", itemType, ip, fmt.Sprintf("Moved %s %s to Unused", itemType, id))
	return c.JSON(fiber.Map{"success": true})
}

func ListUnused(c *fiber.Ctx) error {
	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return c.Status(503).JSON(fiber.Map{"error": true, "message": "Database not ready"})
	}
	
	var cards []fiber.Map
	var keys []fiber.Map

	// List deleted cards
	cRows, err := database.Router.GetGlobalManagerDB().Query("SELECT id, name, icon FROM api_cards WHERE is_deleted = 1")
	if err == nil && cRows != nil {
		defer cRows.Close()
		for cRows.Next() {
			var id, name, icon string
			cRows.Scan(&id, &name, &icon)
			cards = append(cards, fiber.Map{"id": id, "name": name, "icon": icon, "type": "card"})
		}
	}

	// List deleted keys
	kRows, err := database.Router.GetGlobalManagerDB().Query("SELECT id, label FROM managed_api_keys WHERE is_deleted = 1")
	if err == nil && kRows != nil {
		defer kRows.Close()
		for kRows.Next() {
			var id, label string
			kRows.Scan(&id, &label)
			keys = append(keys, fiber.Map{"id": id, "name": label, "type": "key"})
		}
	}

	return c.JSON(fiber.Map{"success": true, "items": append(cards, keys...)})
}

func PermanentDelete(c *fiber.Ctx) error {
	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return c.Status(503).JSON(fiber.Map{"error": true, "message": "Database not ready"})
	}
	itemType := c.Params("type")
	id := c.Params("id")
	ip, _ := c.Locals("admin_ip").(string)

	var err error
	if itemType == "card" {
		_, err = database.Router.GetGlobalManagerDB().Exec("DELETE FROM api_cards WHERE id = ?", id)
	} else if itemType == "key" {
		_, err = database.Router.GetGlobalManagerDB().Exec("DELETE FROM managed_api_keys WHERE id = ?", id)
	}

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Delete failed"})
	}

	admin.LogAudit("PERM_DELETE_API", itemType, ip, fmt.Sprintf("Permanently deleted %s %s", itemType, id))
	return c.JSON(fiber.Map{"success": true})
}
