// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package repository

import (
	"sessiondb/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DBUserCredentialRepository struct {
	DB *gorm.DB
}

func NewDBUserCredentialRepository(db *gorm.DB) *DBUserCredentialRepository {
	return &DBUserCredentialRepository{DB: db}
}

// Create creates a new DB user credential
func (r *DBUserCredentialRepository) Create(cred *models.DBUserCredential) error {
	return r.DB.Create(cred).Error
}

// FindByID finds a credential by ID
func (r *DBUserCredentialRepository) FindByID(id uuid.UUID) (*models.DBUserCredential, error) {
	var cred models.DBUserCredential
	err := r.DB.Preload("User").Preload("Instance").First(&cred, id).Error
	if err != nil {
		return nil, err
	}
	return &cred, nil
}

// FindByUserAndInstance finds a credential for a specific user and instance
func (r *DBUserCredentialRepository) FindByUserAndInstance(userID, instanceID uuid.UUID) (*models.DBUserCredential, error) {
	var cred models.DBUserCredential
	err := r.DB.Where("user_id = ? AND instance_id = ? AND status = ?", userID, instanceID, "active").
		Preload("User").
		Preload("Instance").
		First(&cred).Error
	if err != nil {
		return nil, err
	}
	return &cred, nil
}

// FindByUser finds all credentials for a user
func (r *DBUserCredentialRepository) FindByUser(userID uuid.UUID) ([]models.DBUserCredential, error) {
	var creds []models.DBUserCredential
	err := r.DB.Where("user_id = ?", userID).
		Preload("Instance").
		Order("created_at desc").
		Find(&creds).Error
	return creds, err
}

// FindExpired finds all expired credentials
func (r *DBUserCredentialRepository) FindExpired() ([]models.DBUserCredential, error) {
	var creds []models.DBUserCredential
	err := r.DB.Where("expires_at IS NOT NULL AND expires_at < NOW() AND status = ?", "active").
		Preload("User").
		Preload("Instance").
		Find(&creds).Error
	return creds, err
}

// UpdateStatus updates the status of a credential
func (r *DBUserCredentialRepository) UpdateStatus(id uuid.UUID, status string) error {
	return r.DB.Model(&models.DBUserCredential{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// Delete deletes a credential
func (r *DBUserCredentialRepository) Delete(id uuid.UUID) error {
	return r.DB.Delete(&models.DBUserCredential{}, id).Error
}

// UpdateRole updates the role of a credential
func (r *DBUserCredentialRepository) UpdateRole(id uuid.UUID, role string) error {
	return r.DB.Model(&models.DBUserCredential{}).
		Where("id = ?", id).
		Update("role", role).Error
}

// FindAll finds all credentials (admin use)
func (r *DBUserCredentialRepository) FindAll() ([]models.DBUserCredential, error) {
	var creds []models.DBUserCredential
	err := r.DB.Preload("User").
		Preload("Instance").
		Order("created_at desc").
		Find(&creds).Error
	return creds, err
}

// FindByInstance finds all credentials for an instance
func (r *DBUserCredentialRepository) FindByInstance(instanceID uuid.UUID) ([]models.DBUserCredential, error) {
	var creds []models.DBUserCredential
	err := r.DB.Where("instance_id = ?", instanceID).
		Preload("User").
		Order("created_at desc").
		Find(&creds).Error
	return creds, err
}
