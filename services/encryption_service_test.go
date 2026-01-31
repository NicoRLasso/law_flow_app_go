package services

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncryptDecrypt(t *testing.T) {
	// Generate a test key
	testKey, err := GenerateEncryptionKey()
	assert.NoError(t, err)
	assert.NotEmpty(t, testKey)

	// Set the key for testing
	os.Setenv("DATA_ENCRYPTION_KEY", testKey)
	defer os.Unsetenv("DATA_ENCRYPTION_KEY")

	t.Run("Encrypt and Decrypt", func(t *testing.T) {
		plaintext := "1234567890" // Simulating a Cédula
		encrypted, err := EncryptSensitiveData(plaintext)
		assert.NoError(t, err)
		assert.NotEmpty(t, encrypted)
		assert.NotEqual(t, plaintext, encrypted)

		decrypted, err := DecryptSensitiveData(encrypted)
		assert.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("Empty string", func(t *testing.T) {
		encrypted, err := EncryptSensitiveData("")
		assert.NoError(t, err)
		assert.Empty(t, encrypted)

		decrypted, err := DecryptSensitiveData("")
		assert.NoError(t, err)
		assert.Empty(t, decrypted)
	})

	t.Run("Different ciphertexts for same plaintext", func(t *testing.T) {
		plaintext := "test-value"
		encrypted1, _ := EncryptSensitiveData(plaintext)
		encrypted2, _ := EncryptSensitiveData(plaintext)
		// Due to random nonce, ciphertexts should be different
		assert.NotEqual(t, encrypted1, encrypted2)
	})
}

func TestEncryptionWithoutKey(t *testing.T) {
	os.Unsetenv("DATA_ENCRYPTION_KEY")

	_, err := EncryptSensitiveData("test")
	assert.ErrorIs(t, err, ErrEncryptionKeyNotSet)

	_, err = DecryptSensitiveData("test")
	assert.ErrorIs(t, err, ErrEncryptionKeyNotSet)
}

func TestInvalidCiphertext(t *testing.T) {
	testKey, _ := GenerateEncryptionKey()
	os.Setenv("DATA_ENCRYPTION_KEY", testKey)
	defer os.Unsetenv("DATA_ENCRYPTION_KEY")

	// Invalid base64
	_, err := DecryptSensitiveData("not-valid-base64!!!")
	assert.Error(t, err)

	// Too short (less than nonce size)
	_, err = DecryptSensitiveData("YWJj") // "abc" in base64
	assert.ErrorIs(t, err, ErrInvalidCiphertext)
}

func TestGenerateEncryptionKey(t *testing.T) {
	key1, err := GenerateEncryptionKey()
	assert.NoError(t, err)
	assert.NotEmpty(t, key1)

	key2, err := GenerateEncryptionKey()
	assert.NoError(t, err)
	assert.NotEqual(t, key1, key2) // Should be random
}
