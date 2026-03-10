// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package repository

import (
	"sessiondb/internal/models"

	"gorm.io/gorm"
)

// FeatureNotifyRepository persists "Notify me when this is ready" sign-ups for roadmap features.
type FeatureNotifyRepository struct {
	DB *gorm.DB
}

// NewFeatureNotifyRepository returns a new FeatureNotifyRepository.
func NewFeatureNotifyRepository(db *gorm.DB) *FeatureNotifyRepository {
	return &FeatureNotifyRepository{DB: db}
}

// Create inserts a new FeatureNotifyRequest. Idempotent: same email+featureKey may be stored once
// (caller can use ExistsByEmailAndFeature to avoid duplicates).
func (r *FeatureNotifyRepository) Create(req *models.FeatureNotifyRequest) error {
	return r.DB.Create(req).Error
}

// ExistsByEmailAndFeature returns true if a request already exists for this email and feature key.
func (r *FeatureNotifyRepository) ExistsByEmailAndFeature(email, featureKey string) (bool, error) {
	var count int64
	err := r.DB.Model(&models.FeatureNotifyRequest{}).
		Where("email = ? AND feature_key = ?", email, featureKey).
		Count(&count).Error
	return count > 0, err
}
