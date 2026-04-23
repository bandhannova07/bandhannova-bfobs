package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/bandhannova/api-hunter/internal/proxy"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"net/smtp"
	"crypto/tls"
)

var ResendKM *proxy.KeyManager

func InitEmailHandlers() {
	ResendKM = proxy.NewKeyManager(config.AppConfig.ResendKeys, "Resend")
	log.Printf("📧 Email Handler initialized: %d Resend keys Pooled", len(config.AppConfig.ResendKeys))
}

type EmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
	Text    string   `json:"text,omitempty"`
}

const BandhanNovaLogoURL = "https://raw.githubusercontent.com/BandhanNova/branding/main/logo-dark.png"

func wrapWithBranding(content string) string {
	return `
<!DOCTYPE html>
<html>
<head>
    <style>
        .container { font-family: 'Outfit', 'Segoe UI', sans-serif; color: #1e293b; max-width: 600px; margin: 0 auto; border: 1px solid #e2e8f0; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1); }
        .header { background: #0f172a; padding: 25px; text-align: center; }
        .logo { height: 45px; }
        .content { padding: 40px; line-height: 1.7; font-size: 16px; }
        .footer { background: #f8fafc; padding: 25px; text-align: center; font-size: 12px; color: #64748b; border-top: 1px solid #e2e8f0; }
        .brand { font-weight: 600; color: #0f172a; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <img src="` + BandhanNovaLogoURL + `" alt="BandhanNova" class="logo">
        </div>
        <div class="content">
            ` + content + `
        </div>
        <div class="footer">
            <div class="brand">BandhanNova Platforms</div>
            Empowering Innovation, Elevating Excellence.<br>
            © ` + fmt.Sprintf("%d", time.Now().Year()) + ` All rights reserved.
        </div>
    </div>
</body>
</html>
`
}

// ProxyEmailSend sends a transactional email using pooled Resend keys
func ProxyEmailSend(c *fiber.Ctx) error {
	var req EmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Default to mail@bandhannova.in if no 'from' provided or simple prefix used
	if req.From == "" || req.From == "mail" || req.From == "info" || req.From == "support" {
		prefix := "mail"
		if req.From != "" {
			prefix = req.From
		}
		req.From = fmt.Sprintf("BandhanNova <%s@bandhannova.in>", prefix)
	}

	if len(req.To) == 0 || req.Subject == "" {
		return c.Status(400).JSON(fiber.Map{"error": "To and Subject are required"})
	}

	// Auto-wrap HTML with branding if not already wrapped
	if req.HTML != "" && !strings.Contains(req.HTML, "<!DOCTYPE html>") {
		req.HTML = wrapWithBranding(req.HTML)
	}

	key := ResendKM.GetNextKey()
	if key == nil {
		return c.Status(503).JSON(fiber.Map{"error": "No available email keys"})
	}

	providerURL := "https://api.resend.com/emails"
	
	// Resend expects a slightly different JSON structure if 'to' is a list
	// but standard lib handled it fine. Let's make it robust.
	payload, _ := json.Marshal(req)

	client := &http.Client{Timeout: 30 * time.Second}
	httpReq, _ := http.NewRequest("POST", providerURL, bytes.NewBuffer(payload))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+key.Value)

	resp, err := client.Do(httpReq)
	if err != nil {
		ResendKM.UpdateKeyStatus(key.Value, 500, err.Error(), nil)
		return c.Status(502).JSON(fiber.Map{"error": fmt.Sprintf("Email provider error: %v", err)})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	ResendKM.UpdateKeyStatus(key.Value, resp.StatusCode, string(body), resp.Header)

	// Save to outbound logs in Analytics Shard
	if database.Router != nil {
		go func() {
			db := database.Router.GetAnalyticsDB()
			now := time.Now().Unix()
			status := "sent"
			if resp.StatusCode >= 400 {
				status = "error"
			}
			_, _ = db.Exec(
				"INSERT INTO outbound_emails (id, to_email, subject, provider, api_key_used, status, timestamp) VALUES (?, ?, ?, 'resend', ?, ?, ?)",
				uuid.New().String(), req.To[0], req.Subject, key.Value[:8]+"...", status, now,
			)
		}()
	}

	return c.Status(resp.StatusCode).Send(body)
}

// SendViaSMTP sends an email using a custom SMTP configuration
func SendViaSMTP(host string, port int, user, pass, encryption, from, to, subject, body string) error {
	auth := smtp.PlainAuth("", user, pass, host)
	
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s", to, subject, body))
	
	addr := fmt.Sprintf("%s:%d", host, port)

	if encryption == "ssl" || encryption == "tls" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         host,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return err
		}
		defer conn.Close()

		c, err := smtp.NewClient(conn, host)
		if err != nil {
			return err
		}

		if err = c.Auth(auth); err != nil {
			return err
		}

		if err = c.Mail(from); err != nil {
			return err
		}

		if err = c.Rcpt(to); err != nil {
			return err
		}

		w, err := c.Data()
		if err != nil {
			return err
		}

		_, err = w.Write(msg)
		if err != nil {
			return err
		}

		err = w.Close()
		if err != nil {
			return err
		}

		return c.Quit()
	}

	// Standard SMTP
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

// RelayEmailHandler is the high-level handler for the hybrid relay
func RelayEmailHandler(c *fiber.Ctx) error {
	var req EmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	// 1. Try Resend first (if keys available)
	if len(config.AppConfig.ResendKeys) > 0 {
		err := ProxyEmailSend(c)
		if err == nil {
			return nil
		}
		log.Printf("⚠️ Resend failed, falling back to custom SMTP: %v", err)
	}

	// 2. Fallback to custom SMTP providers from DB
	db := database.Router.GetCoreMasterDB()
	var host, user, pass, encryption, fromEmail string
	var port int
	err := db.QueryRow("SELECT host, port, username, password, encryption, from_email FROM managed_smtp_providers WHERE status = 'active' LIMIT 1").Scan(
		&host, &port, &user, &pass, &encryption, &fromEmail,
	)

	if err != nil {
		return c.Status(503).JSON(fiber.Map{"error": "No available mail providers configured"})
	}

	// Wrap branding
	if !strings.Contains(req.HTML, "<!DOCTYPE html>") {
		req.HTML = wrapWithBranding(req.HTML)
	}

	err = SendViaSMTP(host, port, user, pass, encryption, fromEmail, req.To[0], req.Subject, req.HTML)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": fmt.Sprintf("SMTP Relay failed: %v", err)})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Email relayed successfully via custom SMTP"})
}

// ResendWebhookPayload represents the inbound mail structure from Resend
type ResendWebhookPayload struct {
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
	Data      struct {
		ID      string `json:"id"`
		From    string `json:"from"`
		To      []string `json:"to"`
		Subject string `json:"subject"`
		HTML    string `json:"html"`
		Text    string `json:"text"`
	} `json:"data"`
}

// HandleEmailWebhook receives inbound emails from Resend
func HandleEmailWebhook(c *fiber.Ctx) error {
	// 1. Verify Webhook Secret (Optional but recommended)
	// secret := config.AppConfig.ResendWebhookSecret
	// if secret != "" && c.Get("X-Resend-Webhook-Secret") != secret {
	//     return c.Status(401).SendString("Unauthorized")
	// }

	var payload ResendWebhookPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).SendString("Invalid payload")
	}

	// We only care about email.received events
	if payload.Type != "email.received" {
		return c.SendString("Ignored event type")
	}

	// 2. Save to Inbound logs in Analytics Shard
	if database.Router != nil {
		db := database.Router.GetAnalyticsDB()
		now := time.Now().Unix()
		content := payload.Data.HTML
		if content == "" {
			content = payload.Data.Text
		}
		
		toEmail := ""
		if len(payload.Data.To) > 0 {
			toEmail = payload.Data.To[0]
		}

		_, err := db.Exec(
			"INSERT INTO inbound_emails (id, from_email, to_email, subject, content, resend_id, timestamp) VALUES (?, ?, ?, ?, ?, ?, ?)",
			uuid.New().String(), payload.Data.From, toEmail, payload.Data.Subject, content, payload.Data.ID, now,
		)
		if err != nil {
			log.Printf("⚠️ Failed to save inbound email: %v", err)
		} else {
			log.Printf("📥 Inbound email saved: From=%s Subject=%s", payload.Data.From, payload.Data.Subject)
		}
	}

	return c.Status(200).SendString("OK")
}
