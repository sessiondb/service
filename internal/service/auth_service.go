// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

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
	UserRepo     *repository.UserRepository
	Config       *config.Config
	TenantClient TenantClient
}

func NewAuthService(userRepo *repository.UserRepository, cfg *config.Config, tenantClient TenantClient) *AuthService {
	return &AuthService{
		UserRepo:     userRepo,
		Config:       cfg,
		TenantClient: tenantClient,
	}
}

func (s *AuthService) Login(email, password string) (*models.User, map[string]any, string, string, error) {
	user, err := s.UserRepo.FindByEmail(email)
	if err != nil {
		return nil, nil, "", "", errors.New("invalid credentials")
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return nil, nil, "", "", errors.New("invalid credentials")
	}

	// Fetch static RBAC permissions based on user role
	rbacPermissions := utils.GetPermissionsForRole(user.Role.Name)

	// Fetch Tenant Features from mock client
	// For production, we would need to pass a context tenant ID. For MVP, passing a dummy "tenant_id".
	tenantFeatures, err := s.TenantClient.GetFeaturesForTenant("default_tenant_123")
	if err != nil {
		return nil, nil, "", "", err
	}

	// Convert TenantFeatures map to map[string]any for JWT payload
	featureMap := make(map[string]any)
	for k, v := range tenantFeatures {
		featureMap[k] = v
	}

	// Attach directly to the user object for convenience
	user.RBACPermissions = rbacPermissions

	token, err := utils.GenerateToken(user.ID, user.Email, user.Role.Name, rbacPermissions, featureMap, s.Config)
	if err != nil {
		return nil, nil, "", "", err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, s.Config)
	if err != nil {
		return nil, nil, "", "", err
	}

	return user, featureMap, token, refreshToken, nil
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
