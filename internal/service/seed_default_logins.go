// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

import (
	"errors"
	"log"

	"sessiondb/internal/config"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"sessiondb/internal/utils"

	"gorm.io/gorm"
)

// SeedDefaultLogins creates platform users from config.DefaultLogins only when no users exist.
// If cfg.DefaultLogins is nil or empty, returns nil. If any user already exists, skips seeding.
// For each default login: finds user by email; if not found, finds role by role_key, hashes password,
// creates user with email, password hash, role_id, and a name. If role is not found, skips that entry and logs.
func SeedDefaultLogins(cfg *config.Config, userRepo *repository.UserRepository, roleRepo *repository.RoleRepository) error {
	if cfg == nil || cfg.DefaultLogins == nil || len(cfg.DefaultLogins) == 0 {
		return nil
	}
	count, err := userRepo.Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	for _, dl := range cfg.DefaultLogins {
		_, err := userRepo.FindByEmail(dl.Email)
		if err == nil {
			continue // user already exists
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		role, err := roleRepo.FindByKey(dl.RoleKey)
		if err != nil || role == nil {
			log.Printf("seed default logins: role with key %q not found, skipping user %s", dl.RoleKey, dl.Email)
			continue
		}
		hash, err := utils.HashPassword(dl.Password)
		if err != nil {
			return err
		}
		name := deriveNameFromEmail(dl.Email)
		user := &models.User{
			Name:         name,
			Email:        dl.Email,
			PasswordHash: hash,
			RoleID:       role.ID,
		}
		if err := userRepo.Create(user); err != nil {
			return err
		}
	}
	return nil
}

// deriveNameFromEmail returns a display name from email (e.g. "admin@example.com" -> "Default User (admin)").
func deriveNameFromEmail(email string) string {
	if email == "" {
		return "Default User"
	}
	for i := 0; i < len(email); i++ {
		if email[i] == '@' {
			if i == 0 {
				return "Default User"
			}
			return "Default User (" + email[:i] + ")"
		}
	}
	return "Default User"
}
