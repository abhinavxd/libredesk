// Package crypto provides AES-256-GCM encryption/decryption utilities
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

const (
	// EncryptedPrefix is prepended to encrypted values to identify them.
	EncryptedPrefix = "enc:"
)

var (
	ErrInvalidKey        = errors.New("encryption key must be 32 bytes")
	ErrInvalidCiphertext = errors.New("invalid ciphertext format")
	ErrDecryptionFailed  = errors.New("decryption failed")
)

// Encrypt encrypts plaintext using AES-256-GCM with the provided key.
// Returns base64 encoded ciphertext with "enc:" prefix.
func Encrypt(plaintext, key string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// Check if already encrypted
	if strings.HasPrefix(plaintext, EncryptedPrefix) {
		return plaintext, nil
	}

	if len(key) != 32 {
		return "", ErrInvalidKey
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	return EncryptedPrefix + encoded, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM with the provided key.
// Expects base64 encoded ciphertext with "enc:" prefix.
func Decrypt(ciphertext, key string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Check if it's encrypted
	if !strings.HasPrefix(ciphertext, EncryptedPrefix) {
		// Not encrypted, return as-is
		return ciphertext, nil
	}

	if len(key) != 32 {
		return "", ErrInvalidKey
	}

	// Remove prefix
	ciphertext = strings.TrimPrefix(ciphertext, EncryptedPrefix)

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", ErrInvalidCiphertext
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// IsEncrypted checks if a value is encrypted by looking for the prefix.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, EncryptedPrefix)
}
