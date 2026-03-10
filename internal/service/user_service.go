// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

import (
	"errors"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"sessiondb/internal/utils"

	"github.com/google/uuid"
)

type UserService struct {
	UserRepo            *repository.UserRepository
	ProvisioningService *DBUserProvisioningService
	InstanceRepo        *repository.InstanceRepository
}

func NewUserService(
	userRepo *repository.UserRepository,
	provisioningService *DBUserProvisioningService,
	instanceRepo *repository.InstanceRepository,
) *UserService {
	return &UserService{
		UserRepo:            userRepo,
		ProvisioningService: provisioningService,
		InstanceRepo:        instanceRepo,
	}
}

// Create creates a platform user and auto-provisions DB users on all instances.
// "Existing user" means a platform user with the same email already exists; in that case we return "email already in use".
func (s *UserService) Create(user *models.User, password string) (*models.User, error) {
	if _, err := s.UserRepo.FindByEmail(user.Email); err == nil {
		return nil, errors.New("email already in use")
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = hashedPassword

	if err := s.UserRepo.Create(user); err != nil {
		return nil, err
	}

	// Auto-provision DB users on all instances: create a DB user when none exists for that user–instance pair (no catalog crawl).
	instances, err := s.InstanceRepo.FindAll()
	if err == nil && len(instances) > 0 {
		for _, instance := range instances {
			cred, err := s.ProvisioningService.ProvisionDBUser(user, &instance)
			if err != nil {
				// Log error but don't fail user creation
				// TODO: Add proper logging
				continue
			}

			// Grant permissions if user has any
			if len(user.Permissions) > 0 {
				if err := s.ProvisioningService.GrantPermissions(cred, user.Permissions); err != nil {
					// Log error but continue
					// TODO: Add proper logging
				}
			}
		}
	}

	return user, nil
}

func (s *UserService) Update(user *models.User) error {
	// Implement Update in Repo first, assume it exists or use DB.Save
	return s.UserRepo.Update(user)
}

func (s *UserService) Delete(id uuid.UUID) error {
	return s.UserRepo.Delete(id)
}

func (s *UserService) SyncTabs(userID uuid.UUID, tabs []models.QueryTab) error {
	// Ensure all tabs have the correct UserID
	for i := range tabs {
		tabs[i].UserID = userID
	}
	return s.UserRepo.UpdateTabs(userID, tabs)
}

func (s *UserService) GetUserByID(id uuid.UUID) (*models.User, error) {
	return s.UserRepo.FindByID(id)
}

func (s *UserService) GetAllUsers() ([]models.User, error) {
	return s.UserRepo.FindAll()
}
