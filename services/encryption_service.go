package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	// ErrEncryptionKeyNotSet indicates the encryption key environment variable is not configured
	ErrEncryptionKeyNotSet = errors.New("DATA_ENCRYPTION_KEY environment variable is not set")
	// ErrInvalidCiphertext indicates the ciphertext is malformed or too short
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
)

// getEncryptionKey retrieves the encryption key from environment variables.
// The key must be exactly 32 bytes (256 bits) for AES-256.
func getEncryptionKey() ([]byte, error) {
	keyStr := os.Getenv("DATA_ENCRYPTION_KEY")
	if keyStr == "" {
		return nil, ErrEncryptionKeyNotSet
	}

	// Decode base64 key
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption key: %w", err)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (got %d bytes)", len(key))
	}

	return key, nil
}

// EncryptSensitiveData encrypts plaintext using AES-256-GCM.
// Returns base64-encoded ciphertext.
func EncryptSensitiveData(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil // Don't encrypt empty strings
	}

	key, err := getEncryptionKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and prepend nonce to ciphertext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return as base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptSensitiveData decrypts base64-encoded ciphertext using AES-256-GCM.
// Returns the original plaintext.
func DecryptSensitiveData(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil // Don't decrypt empty strings
	}

	key, err := getEncryptionKey()
	if err != nil {
		return "", err
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(data) < gcm.NonceSize() {
		return "", ErrInvalidCiphertext
	}

	// Extract nonce and ciphertext
	nonce, cipherData := data[:gcm.NonceSize()], data[gcm.NonceSize():]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// GenerateEncryptionKey generates a new random 32-byte key for AES-256 and returns it as base64.
// Use this to generate a key for the DATA_ENCRYPTION_KEY environment variable.
func GenerateEncryptionKey() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
