package email

import (
	"time"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type SMTPProvider struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	Encryption string `json:"encryption"`
	FromEmail  string `json:"from_email"`
	Status     string `json:"status"`
	CreatedAt  int64  `json:"created_at"`
}

// ListSMTPProviders returns all configured SMTP relays
func ListSMTPProviders(c *fiber.Ctx) error {
	db := database.Router.GetCoreMasterDB()
	rows, err := db.Query("SELECT id, name, host, port, username, encryption, from_email, status, created_at FROM managed_smtp_providers")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": err.Error()})
	}
	defer rows.Close()

	var providers []SMTPProvider
	for rows.Next() {
		var p SMTPProvider
		err := rows.Scan(&p.ID, &p.Name, &p.Host, &p.Port, &p.Username, &p.Encryption, &p.FromEmail, &p.Status, &p.CreatedAt)
		if err == nil {
			providers = append(providers, p)
		}
	}

	return c.JSON(fiber.Map{"success": true, "providers": providers})
}

// AddSMTPProvider registers a new relay
func AddSMTPProvider(c *fiber.Ctx) error {
	var p SMTPProvider
	if err := c.BodyParser(&p); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid request"})
	}

	p.ID = uuid.New().String()
	p.CreatedAt = time.Now().Unix()
	if p.Status == "" { p.Status = "active" }

	db := database.Router.GetCoreMasterDB()
	_, err := db.Exec(
		"INSERT INTO managed_smtp_providers (id, name, host, port, username, password, encryption, from_email, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		p.ID, p.Name, p.Host, p.Port, p.Username, p.Password, p.Encryption, p.FromEmail, p.Status, p.CreatedAt,
	)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "SMTP Provider added", "id": p.ID})
}

// DeleteSMTPProvider removes a relay
func DeleteSMTPProvider(c *fiber.Ctx) error {
	id := c.Params("id")
	db := database.Router.GetCoreMasterDB()
	_, err := db.Exec("DELETE FROM managed_smtp_providers WHERE id = ?", id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Provider removed"})
}
