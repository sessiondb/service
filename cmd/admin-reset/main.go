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
	name := "admin"
	password := "Admin123!"

	// Hash the password
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	var user models.User
	result := repository.DB.Where("email = ? OR name = ?", email, name).First(&user)

	if result.Error == nil {
		// User exists, reset password
		user.PasswordHash = hashedPassword
		user.Name = name // Ensure name is "admin"
		if err := repository.DB.Save(&user).Error; err != nil {
			log.Fatalf("Failed to reset password: %v", err)
		}
		log.Printf("Successfully reset password for user %s (%s) to %s", user.Name, user.Email, password)
	} else {
		// User does not exist, create new admin
		var role models.Role
		if err := repository.DB.Where("name = ?", "Super Admin").First(&role).Error; err != nil {
			log.Fatalf("Failed to find Super Admin role: %v", err)
		}

		user = models.User{
			Name:         name,
			Email:        email,
			PasswordHash: hashedPassword,
			RoleID:       role.ID,
			Status:       "active",
		}

		if err := repository.DB.Create(&user).Error; err != nil {
			log.Fatalf("Failed to create admin user: %v", err)
		}
		log.Printf("Successfully created new admin user: %s (%s) with password: %s", name, email, password)
	}
}
