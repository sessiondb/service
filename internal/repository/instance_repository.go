// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package repository

import (
	"sessiondb/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InstanceRepository struct {
	DB *gorm.DB
}

func NewInstanceRepository(db *gorm.DB) *InstanceRepository {
	return &InstanceRepository{DB: db}
}

func (r *InstanceRepository) Create(instance *models.DBInstance) error {
	return r.DB.Create(instance).Error
}

func (r *InstanceRepository) FindByID(id uuid.UUID) (*models.DBInstance, error) {
	var instance models.DBInstance
	err := r.DB.First(&instance, id).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *InstanceRepository) FindAll() ([]models.DBInstance, error) {
	var instances []models.DBInstance
	err := r.DB.Find(&instances).Error
	return instances, err
}

func (r *InstanceRepository) Update(instance *models.DBInstance) error {
	return r.DB.Save(instance).Error
}

func (r *InstanceRepository) Delete(id uuid.UUID) error {
	return r.DB.Delete(&models.DBInstance{}, id).Error
}
