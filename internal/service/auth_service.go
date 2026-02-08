package service

import (
	"errors"
	"log"
	"sessiondb/internal/config"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"sessiondb/internal/utils"
)

type AuthService struct {
	UserRepo *repository.UserRepository
	Config   *config.Config
}

func NewAuthService(userRepo *repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		UserRepo: userRepo,
		Config:   cfg,
	}
}

func (s *AuthService) Login(email, password string) (*models.User, string, string, error) {
	user, err := s.UserRepo.FindByEmail(email)
	if err != nil {
		return nil, "", "", errors.New("invalid credentials")
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return nil, "", "", errors.New("invalid credentials")
	}

	token, err := utils.GenerateToken(user.ID, user.Email, user.Role.Name, s.Config)
	if err != nil {
		return nil, "", "", err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, s.Config)
	if err != nil {
		return nil, "", "", err
	}

	return user, token, refreshToken, nil
}

func (s *AuthService) Register(name, email, password string) (*models.User, error) {
	// Check if user exists
	if _, err := s.UserRepo.FindByEmail(email); err == nil {
		return nil, errors.New("email already in use")
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Find default role (Developer)
	var role models.Role
	if err := s.UserRepo.DB.Where("name = ?", "Developer").First(&role).Error; err != nil {
		// Fallback or error if roles not seeded
		return nil, errors.New("default role 'Developer' not found")
	}
	log.Printf("Found role: %s, ID: %s", role.Name, role.ID)

	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: hashedPassword,
		Status:       "active",
		RoleID:       role.ID,
		Role:         role, // Populate full role for response
	}

	if err := s.UserRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}
