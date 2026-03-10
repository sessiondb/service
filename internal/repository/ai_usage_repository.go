// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package repository

import (
	"sessiondb/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AIUsageRepository persists AITokenUsage for dashboard.
type AIUsageRepository struct {
	DB *gorm.DB
}

// NewAIUsageRepository returns a new AIUsageRepository.
func NewAIUsageRepository(db *gorm.DB) *AIUsageRepository {
	return &AIUsageRepository{DB: db}
}

// RecordUsage inserts one usage record.
func (r *AIUsageRepository) RecordUsage(usage *models.AITokenUsage) error {
	return r.DB.Create(usage).Error
}

// GetUsageByUser returns usage records for a user in the time range.
func (r *AIUsageRepository) GetUsageByUser(userID uuid.UUID, from, to time.Time) ([]models.AITokenUsage, error) {
	var list []models.AITokenUsage
	err := r.DB.Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, from, to).
		Order("created_at DESC").Find(&list).Error
	return list, err
}

// GetUsageSummaryByUser returns per-user aggregate counts for the time range (for admin dashboard).
func (r *AIUsageRepository) GetUsageSummaryByUser(from, to time.Time) ([]struct {
	UserID  uuid.UUID
	Count   int64
	Input   int64
	Output  int64
}, error) {
	var out []struct {
		UserID  uuid.UUID
		Count   int64
		Input   int64
		Output  int64
	}
	err := r.DB.Model(&models.AITokenUsage{}).
		Select("user_id, count(*) as count, coalesce(sum(input_tokens),0) as input, coalesce(sum(output_tokens),0) as output").
		Where("created_at >= ? AND created_at <= ?", from, to).
		Group("user_id").
		Scan(&out).Error
	return out, err
}

// GetGlobalUsageTotal returns total request count and token sums in the time range.
func (r *AIUsageRepository) GetGlobalUsageTotal(from, to time.Time) (count int64, inputTokens, outputTokens int64, err error) {
	var res struct {
		Count  int64
		Input  int64
		Output int64
	}
	err = r.DB.Model(&models.AITokenUsage{}).
		Select("count(*) as count, coalesce(sum(input_tokens),0) as input, coalesce(sum(output_tokens),0) as output").
		Where("created_at >= ? AND created_at <= ?", from, to).
		Scan(&res).Error
	return res.Count, res.Input, res.Output, err
}
