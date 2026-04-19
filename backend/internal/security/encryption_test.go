package security

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	passphrase := "my-secret-master-key"
	originalText := "sk-abc-123,sk-xyz-789"

	encrypted, err := Encrypt(originalText, passphrase)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	if encrypted == originalText {
		t.Fatal("Encrypted text is same as original")
	}

	decrypted, err := Decrypt(encrypted, passphrase)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if decrypted != originalText {
		t.Fatalf("Decrypted text mismatch: expected %s, got %s", originalText, decrypted)
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	passphrase := "correct-key"
	wrongKey := "wrong-key"
	originalText := "secret-data"

	encrypted, _ := Encrypt(originalText, passphrase)
	_, err := Decrypt(encrypted, wrongKey)

	if err == nil {
		t.Fatal("Decryption should have failed with wrong key")
	}
}
