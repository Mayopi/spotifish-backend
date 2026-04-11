package crypto

import (
	"encoding/hex"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	// Generate a valid 32-byte key as hex
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	plaintext := []byte("hello, this is a secret Drive refresh token!")

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Ciphertext should be different from plaintext
	if string(ciphertext) == string(plaintext) {
		t.Fatal("ciphertext should not equal plaintext")
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("decrypted text mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecrypt_DifferentKeys(t *testing.T) {
	key1 := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	key2 := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"

	plaintext := []byte("secret data")

	ciphertext, err := Encrypt(plaintext, key1)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Decrypting with wrong key should fail
	_, err = Decrypt(ciphertext, key2)
	if err == nil {
		t.Fatal("Decrypt with wrong key should fail")
	}
}

func TestEncrypt_InvalidKey(t *testing.T) {
	// Key too short
	_, err := Encrypt([]byte("test"), "0123456789abcdef")
	if err == nil {
		t.Fatal("Encrypt with short key should fail")
	}

	// Invalid hex
	_, err = Encrypt([]byte("test"), "not-a-hex-key-not-a-hex-key-not-a-hex-key-not-a-hex-key-not-hex")
	if err == nil {
		t.Fatal("Encrypt with invalid hex key should fail")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	_, err := Decrypt([]byte("short"), key)
	if err == nil {
		t.Fatal("Decrypt with too-short ciphertext should fail")
	}
}

func TestEncryptDecrypt_EmptyPlaintext(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	ciphertext, err := Encrypt([]byte(""), key)
	if err != nil {
		t.Fatalf("Encrypt empty failed: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt empty failed: %v", err)
	}

	if string(decrypted) != "" {
		t.Fatalf("expected empty string, got %q", decrypted)
	}
}

func TestEncrypt_DifferentCiphertextEachCall(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	plaintext := []byte("same input")

	ct1, _ := Encrypt(plaintext, key)
	ct2, _ := Encrypt(plaintext, key)

	// Due to random nonce, ciphertexts should differ
	if hex.EncodeToString(ct1) == hex.EncodeToString(ct2) {
		t.Fatal("two encryptions of the same plaintext should produce different ciphertexts")
	}
}
