package repository

import (
	"sessiondb/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) Create(user *models.User) error {
	return r.DB.Create(user).Error
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.DB.Preload("Role.Permissions").Preload("Permissions").Preload("SavedScripts").Preload("QueryTabs").Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.DB.Preload("Role.Permissions").Preload("Permissions").Preload("SavedScripts").Preload("QueryTabs").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindAll() ([]models.User, error) {
	var users []models.User
	err := r.DB.Preload("Role.Permissions").Preload("Permissions").Preload("SavedScripts").Preload("QueryTabs").Find(&users).Error
	return users, err
}

func (r *UserRepository) CountByRoleID(roleID uuid.UUID) (int64, error) {
	var count int64
	err := r.DB.Model(&models.User{}).Where("role_id = ?", roleID).Count(&count).Error
	return count, err
}

func (r *UserRepository) Update(user *models.User) error {
	return r.DB.Save(user).Error
}

func (r *UserRepository) Delete(id uuid.UUID) error {
	return r.DB.Delete(&models.User{}, id).Error
}

func (r *UserRepository) FindByName(name string) (*models.User, error) {
	var user models.User
	err := r.DB.Preload("Role.Permissions").Preload("Permissions").Preload("SavedScripts").Preload("QueryTabs").Where("name = ?", name).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) UpdateTabs(userID uuid.UUID, tabs []models.QueryTab) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		// Delete existing tabs
		if err := tx.Where("user_id = ?", userID).Delete(&models.QueryTab{}).Error; err != nil {
			return err
		}
		// Create new tabs
		if len(tabs) > 0 {
			if err := tx.Create(&tabs).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
