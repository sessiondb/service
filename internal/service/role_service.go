// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

import (
	"errors"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"sessiondb/internal/utils"

	"github.com/google/uuid"
)

type RoleService struct {
	RoleRepo *repository.RoleRepository
	UserRepo *repository.UserRepository
}

func NewRoleService(roleRepo *repository.RoleRepository, userRepo *repository.UserRepository) *RoleService {
	return &RoleService{
		RoleRepo: roleRepo,
		UserRepo: userRepo,
	}
}

func (s *RoleService) CreateRole(name, description string, permissions []models.Permission) (*models.Role, error) {
	role := &models.Role{
		Name:        name,
		Key:         utils.ToSnakeCase(name),
		Description: description,
		Permissions: permissions,
	}
	if err := s.RoleRepo.Create(role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *RoleService) GetAllRoles() ([]models.Role, error) {
	roles, err := s.RoleRepo.FindAll()
	if err != nil {
		return nil, err
	}

	for i := range roles {
		count, _ := s.UserRepo.CountByRoleID(roles[i].ID)
		roles[i].UserCount = int(count)
	}
	return roles, nil
}

func (s *RoleService) GetRoleByID(id uuid.UUID) (*models.Role, error) {
	role, err := s.RoleRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	count, _ := s.UserRepo.CountByRoleID(role.ID)
	role.UserCount = int(count)
	return role, nil
}

func (s *RoleService) GetRoleByName(name string) (*models.Role, error) {
	return s.RoleRepo.FindByName(name)
}

func (s *RoleService) UpdateRole(id uuid.UUID, name, description string) (*models.Role, error) {
	role, err := s.RoleRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if role.IsSystemRole {
		return nil, errors.New("cannot modify system role")
	}

	role.Name = name
	role.Key = utils.ToSnakeCase(name)
	role.Description = description

	if err := s.RoleRepo.Update(role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *RoleService) DeleteRole(id uuid.UUID) error {
	role, err := s.RoleRepo.FindByID(id)
	if err != nil {
		return err
	}

	if role.IsSystemRole {
		return errors.New("cannot delete system role")
	}

	return s.RoleRepo.Delete(id)
}
