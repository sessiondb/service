package repository

import (
	"sessiondb/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QueryRepository struct {
	DB *gorm.DB
}

func NewQueryRepository(db *gorm.DB) *QueryRepository {
	return &QueryRepository{DB: db}
}

func (r *QueryRepository) SaveHistory(history *models.QueryHistory) error {
	return r.DB.Create(history).Error
}

func (r *QueryRepository) GetHistory(userID uuid.UUID) ([]models.QueryHistory, error) {
	var history []models.QueryHistory
	err := r.DB.Where("user_id = ?", userID).Order("executed_at desc").Limit(100).Find(&history).Error
	return history, err
}

func (r *QueryRepository) SaveScript(script *models.SavedScript) error {
	return r.DB.Create(script).Error
}

func (r *QueryRepository) GetScripts(userID uuid.UUID) ([]models.SavedScript, error) {
	var scripts []models.SavedScript
	err := r.DB.Where("user_id = ? OR is_public = ?", userID, true).Find(&scripts).Error
	return scripts, err
}
