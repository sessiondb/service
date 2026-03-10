//go:build pro

// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sessiondb/internal/models"
)

// CredentialSessionRepository handles persistence for CredentialSession (Phase 4 – Session Engine).
type CredentialSessionRepository struct {
	DB *gorm.DB
}

// NewCredentialSessionRepository returns a new CredentialSessionRepository.
func NewCredentialSessionRepository(db *gorm.DB) *CredentialSessionRepository {
	return &CredentialSessionRepository{DB: db}
}

// Create persists a CredentialSession.
func (r *CredentialSessionRepository) Create(s *models.CredentialSession) error {
	return r.DB.Create(s).Error
}

// FindByID loads a CredentialSession by ID.
func (r *CredentialSessionRepository) FindByID(id uuid.UUID) (*models.CredentialSession, error) {
	var s models.CredentialSession
	if err := r.DB.First(&s, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// FindActiveByUserAndInstance returns the active CredentialSession for the user on the instance, if any.
// Returns (nil, nil) when no active session exists.
func (r *CredentialSessionRepository) FindActiveByUserAndInstance(userID, instanceID uuid.UUID) (*models.CredentialSession, error) {
	var s models.CredentialSession
	err := r.DB.Where("user_id = ? AND instance_id = ? AND status = ?", userID, instanceID, "active").
		Order("created_at DESC").First(&s).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// Update saves changes to a CredentialSession.
func (r *CredentialSessionRepository) Update(s *models.CredentialSession) error {
	return r.DB.Save(s).Error
}
