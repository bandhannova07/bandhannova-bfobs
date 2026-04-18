package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/security"
)

const ValidationText = "BANDHANNOVA_VALID_2026"

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("🔐 BandhanNova API Hunter - Secure Registry Tool")
	fmt.Println("-----------------------------------------------")

	// 1. Get Master Key
	fmt.Print("Enter your BANDHANNOVA_MASTER_KEY: ")
	masterKey, _ := reader.ReadString('\n')
	masterKey = strings.TrimSpace(masterKey)

	if masterKey == "" {
		fmt.Println("❌ Error: Master Key cannot be empty.")
		return
	}

	// 2. Verify Master Key
	validationHash := config.InternalRegistry["VALIDATION_HASH"]
	if validationHash == "" {
		fmt.Println("⚠️  No Validation Hash found in internal_registry.go.")
		fmt.Print("Do you want to initialize it with this Master Key? (y/n): ")
		choice, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(choice)) == "y" {
			hash, err := security.Encrypt(ValidationText, masterKey)
			if err != nil {
				fmt.Printf("❌ Failed to create validation hash: %v\n", err)
				return
			}
			fmt.Println("\n✅ New Validation Hash generated!")
			fmt.Println("Copy this into config/internal_registry.go under \"VALIDATION_HASH\":")
			fmt.Printf("\n\"VALIDATION_HASH\": \"%s\",\n", hash)
			fmt.Println("\nAfter updating the file, run this tool again.")
			return
		} else {
			fmt.Println("❌ Initialization cancelled. Cannot proceed without verification.")
			return
		}
	}

	// Try to decrypt the validation hash
	decrypted, err := security.Decrypt(validationHash, masterKey)
	if err != nil || decrypted != ValidationText {
		fmt.Println("❌ Error: Master Key is WRONG! Verification failed.")
		return
	}

	fmt.Println("✅ Master Key Verified Successfully!")
	fmt.Println("-----------------------------------------------")

	// 3. Interactive Loop
	for {
		fmt.Print("\nEnter Key Name (e.g., OPENROUTER_KEYS) or type 'exit' to quit: ")
		keyName, _ := reader.ReadString('\n')
		keyName = strings.TrimSpace(keyName)

		if keyName == "exit" || keyName == "quit" {
			break
		}

		fmt.Printf("Enter Value(s) for %s: ", keyName)
		keyValue, _ := reader.ReadString('\n')
		keyValue = strings.TrimSpace(keyValue)

		if keyValue == "" {
			fmt.Println("⚠️ Value cannot be empty.")
			continue
		}

		encrypted, err := security.Encrypt(keyValue, masterKey)
		if err != nil {
			fmt.Printf("❌ Encryption failed: %v\n", err)
			continue
		}

		fmt.Println("\n✅ Encrypted Successfully!")
		fmt.Printf("Paste this into config/internal_registry.go:\n")
		fmt.Printf("\"%s\": \"%s\",\n", keyName, encrypted)
		fmt.Println("-----------------------------------------------")
	}

	fmt.Println("\n👋 Done. Don't forget to push your code to Hugging Face!")
}
