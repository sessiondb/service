// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package repository

import (
	"errors"
	"sessiondb/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PermissionRepository handles persistence for data-level Permission records.
type PermissionRepository struct {
	DB *gorm.DB
}

// NewPermissionRepository returns a new PermissionRepository.
func NewPermissionRepository(db *gorm.DB) *PermissionRepository {
	return &PermissionRepository{DB: db}
}

// Create creates a permission.
func (r *PermissionRepository) Create(p *models.Permission) error {
	return r.DB.Create(p).Error
}

// FindByUserAndInstance returns all permissions that apply to the user for the given instance:
// direct user permissions (user_id = userID) and role permissions (role_id = user's role).
// Only permissions with instance_id = instanceID are returned; records without instance_id are ignored.
// Returns empty slice if user is not found (no permissions).
func (r *PermissionRepository) FindByUserAndInstance(userID, instanceID uuid.UUID) ([]models.Permission, error) {
	user, err := NewUserRepository(r.DB).FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	var perms []models.Permission
	if err := r.DB.Where("user_id = ? AND instance_id = ?", userID, instanceID).Find(&perms).Error; err != nil {
		return nil, err
	}
	if user.RoleID != uuid.Nil {
		var rolePerms []models.Permission
		if err := r.DB.Where("role_id = ? AND instance_id = ?", user.RoleID, instanceID).Find(&rolePerms).Error; err != nil {
			return nil, err
		}
		perms = append(perms, rolePerms...)
	}
	return perms, nil
}

// FindByUserID returns all direct permissions for a user (for admin UI).
func (r *PermissionRepository) FindByUserID(userID uuid.UUID) ([]models.Permission, error) {
	var perms []models.Permission
	err := r.DB.Where("user_id = ?", userID).Find(&perms).Error
	return perms, err
}

// Delete deletes a permission by ID.
func (r *PermissionRepository) Delete(id uuid.UUID) error {
	return r.DB.Delete(&models.Permission{}, id).Error
}

// Update updates a permission.
func (r *PermissionRepository) Update(p *models.Permission) error {
	return r.DB.Save(p).Error
}
