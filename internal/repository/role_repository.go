package repository

import (
	"sessiondb/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoleRepository struct {
	DB *gorm.DB
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{DB: db}
}

func (r *RoleRepository) Create(role *models.Role) error {
	return r.DB.Create(role).Error
}

func (r *RoleRepository) FindAll() ([]models.Role, error) {
	var roles []models.Role
	err := r.DB.Preload("Permissions").Find(&roles).Error
	return roles, err
}

func (r *RoleRepository) FindByID(id uuid.UUID) (*models.Role, error) {
	var role models.Role
	err := r.DB.Preload("Permissions").First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) Update(role *models.Role) error {
	return r.DB.Save(role).Error
}

func (r *RoleRepository) FindByName(name string) (*models.Role, error) {
	var role models.Role
	err := r.DB.Where("name = ?", name).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) Delete(id uuid.UUID) error {
	return r.DB.Delete(&models.Role{}, id).Error
}
