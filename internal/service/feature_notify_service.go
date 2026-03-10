// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

import (
	"sessiondb/internal/models"
	"sessiondb/internal/repository"

	"github.com/google/uuid"
)

// FeatureNotifyService handles "Notify me when this is ready" sign-ups for roadmap features.
type FeatureNotifyService struct {
	Repo *repository.FeatureNotifyRepository
}

// NewFeatureNotifyService returns a new FeatureNotifyService.
func NewFeatureNotifyService(repo *repository.FeatureNotifyRepository) *FeatureNotifyService {
	return &FeatureNotifyService{Repo: repo}
}

// RegisterRequest creates a new feature notify request. If the same email+featureKey already exists,
// it returns (nil, nil) so the handler can respond with 200 "Already registered".
func (s *FeatureNotifyService) RegisterRequest(email, featureKey string, userID *uuid.UUID) (*models.FeatureNotifyRequest, bool, error) {
	if email == "" || featureKey == "" {
		return nil, false, nil
	}
	exists, err := s.Repo.ExistsByEmailAndFeature(email, featureKey)
	if err != nil {
		return nil, false, err
	}
	if exists {
		return nil, true, nil
	}
	req := &models.FeatureNotifyRequest{
		Email:      email,
		FeatureKey: featureKey,
		UserID:     userID,
	}
	if err := s.Repo.Create(req); err != nil {
		return nil, false, err
	}
	return req, false, nil
}
