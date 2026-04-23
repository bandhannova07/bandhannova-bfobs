package main

import (
	"fmt"
	"github.com/bandhannova/api-hunter/internal/security"
)

func main() {
	masterKey := "bdn-bandhannova-master-key-ehb66vk7jhbfl4kjzufg7a456734twrddsbh67363vxfdy64gvaghase32rdvuz"
	
	// VALIDATION_HASH from internal_registry.go
	validationHash := "Mzt6/XsK4e+20m06BrKc8xt5SHzDE9rhtLl0n7HLx3yv2XLbqLQlfZtQuqrfSny6AS4="
	
	fmt.Printf("🔑 Master Key: %s\n", masterKey)
	fmt.Printf("🔒 Validation Hash: %s\n", validationHash)
	
	decrypted, err := security.Decrypt(validationHash, masterKey)
	if err != nil {
		fmt.Printf("❌ Validation FAILED: %v\n", err)
		fmt.Println("\n⚠️  This Master Key is NOT the one used for the Internal Registry.")
	} else {
		fmt.Printf("✅ Validation SUCCESS! Decrypted: %s\n", decrypted)
	}
}
