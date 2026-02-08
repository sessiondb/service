package main

import (
	"log"
	"sessiondb/internal/config"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"sessiondb/internal/utils"
)

func main() {
	// Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to Database
	repository.ConnectDB(cfg)

	email := "admin@example.com"
	newPassword := "Start123!"

	// Hash the new password
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Update the user
	// We use Model(&models.User{}) to ensure GORM knows which table to update
	if err := repository.DB.Model(&models.User{}).Where("email = ?", email).Update("password_hash", hashedPassword).Error; err != nil {
		log.Fatalf("Failed to update user password: %v", err)
	}

	log.Printf("Successfully reset password for %s to %s", email, newPassword)
}
