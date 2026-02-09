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

// Methods for QueryHistory removed as we now use AuditLog

func (r *QueryRepository) SaveScript(script *models.SavedScript) error {
	return r.DB.Create(script).Error
}

func (r *QueryRepository) GetScripts(userID uuid.UUID) ([]models.SavedScript, error) {
	var scripts []models.SavedScript
	err := r.DB.Where("user_id = ? OR is_public = ?", userID, true).Find(&scripts).Error
	return scripts, err
}
