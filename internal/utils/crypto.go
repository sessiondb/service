// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
)

// GetEncryptionKey retrieves the encryption key from environment variable
func GetEncryptionKey() ([]byte, error) {
	key := os.Getenv("DB_CREDENTIAL_ENCRYPTION_KEY")
	if key == "" {
		return nil, errors.New("DB_CREDENTIAL_ENCRYPTION_KEY environment variable not set")
	}

	// Key must be 32 bytes for AES-256
	keyBytes := []byte(key)
	if len(keyBytes) != 32 {
		return nil, errors.New("encryption key must be exactly 32 bytes for AES-256")
	}

	return keyBytes, nil
}

// EncryptPassword encrypts a password using AES-256-GCM
func EncryptPassword(plainPassword string) (string, error) {
	key, err := GetEncryptionKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Create a nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt the password
	ciphertext := gcm.Seal(nonce, nonce, []byte(plainPassword), nil)

	// Encode to base64 for storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptPassword decrypts an encrypted password
func DecryptPassword(encryptedPassword string) (string, error) {
	key, err := GetEncryptionKey()
	if err != nil {
		return "", err
	}

	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedPassword)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
