// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package utils

import (
	"crypto/rand"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

const (
	// Character sets for password generation
	lowerChars   = "abcdefghijklmnopqrstuvwxyz"
	upperChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitChars   = "0123456789"
	specialChars = "!@#$%^&*()-_=+[]{}|;:,.<>?"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateSecurePassword generates a cryptographically secure random password
func GenerateSecurePassword(length int) (string, error) {
	if length < 8 {
		length = 24 // Default to 24 characters
	}

	// Combine all character sets
	allChars := lowerChars + upperChars + digitChars + specialChars

	password := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(allChars))))
		if err != nil {
			return "", err
		}
		password[i] = allChars[num.Int64()]
	}

	return string(password), nil
}
