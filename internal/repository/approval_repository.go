package repository

import (
	"sessiondb/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ApprovalRepository struct {
	DB *gorm.DB
}

func NewApprovalRepository(db *gorm.DB) *ApprovalRepository {
	return &ApprovalRepository{DB: db}
}

func (r *ApprovalRepository) Create(request *models.ApprovalRequest) error {
	return r.DB.Create(request).Error
}

func (r *ApprovalRepository) FindByID(id uuid.UUID) (*models.ApprovalRequest, error) {
	var request models.ApprovalRequest
	err := r.DB.Preload("Requester").Preload("TargetUser").First(&request, id).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *ApprovalRepository) FindByStatus(status string) ([]models.ApprovalRequest, error) {
	var requests = make([]models.ApprovalRequest, 0)
	err := r.DB.Preload("Requester").Where("status = ?", status).Find(&requests).Error
	return requests, err
}

func (r *ApprovalRepository) FindAll() ([]models.ApprovalRequest, error) {
	var requests = make([]models.ApprovalRequest, 0)
	err := r.DB.Preload("Requester").Order("created_at desc").Find(&requests).Error
	return requests, err
}

func (r *ApprovalRepository) Update(request *models.ApprovalRequest) error {
	return r.DB.Save(request).Error
}
