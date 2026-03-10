// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package repository

import (
	"sessiondb/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AIConfigRepository handles UserAIConfig and AIExecutionPolicy persistence.
type AIConfigRepository struct {
	DB *gorm.DB
}

// NewAIConfigRepository returns a new AIConfigRepository.
func NewAIConfigRepository(db *gorm.DB) *AIConfigRepository {
	return &AIConfigRepository{DB: db}
}

// GetUserAIConfig returns the AI config for the user (BYOK). One config per user for now.
func (r *AIConfigRepository) GetUserAIConfig(userID uuid.UUID) (*models.UserAIConfig, error) {
	var cfg models.UserAIConfig
	err := r.DB.Where("user_id = ?", userID).First(&cfg).Error
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveUserAIConfig creates or updates the user's AI config.
func (r *AIConfigRepository) SaveUserAIConfig(cfg *models.UserAIConfig) error {
	return r.DB.Save(cfg).Error
}

// GetAIExecutionPolicies returns all policies for an instance.
func (r *AIConfigRepository) GetAIExecutionPolicies(instanceID uuid.UUID) ([]models.AIExecutionPolicy, error) {
	var list []models.AIExecutionPolicy
	err := r.DB.Where("instance_id = ?", instanceID).Find(&list).Error
	return list, err
}
