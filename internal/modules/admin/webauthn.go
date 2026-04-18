package admin

import (
	"github.com/gofiber/fiber/v2"
	"log"
)

// InitWebAuthn sets up the biometric lock system
func InitWebAuthn(router fiber.Router) {
	webauthn := router.Group("/webauthn")

	// Start registration for FaceID/Fingerprint
	webauthn.Post("/register/begin", func(c *fiber.Ctx) error {
		// In a real implementation, we would use a library like github.com/go-webauthn/webauthn
		return c.JSON(fiber.Map{
			"success": true,
			"message": "WebAuthn Registration Challenge generated",
		})
	})

	// Finalize registration
	webauthn.Post("/register/finish", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Biometric lock registered successfully",
		})
	})

	// Biometric Login Login
	webauthn.Post("/login/begin", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "WebAuthn Login Challenge generated",
		})
	})

	log.Println("🛡️  WebAuthn Biometric System Initialized (Draft)")
}
