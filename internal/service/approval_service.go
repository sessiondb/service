// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// DBUserProvisioner is used to provision DB users and grant permissions (e.g. DBUserProvisioningService).
type DBUserProvisioner interface {
	ProvisionDBUser(user *models.User, instance *models.DBInstance) (*models.DBUserCredential, error)
	GrantPermissions(cred *models.DBUserCredential, permissions []models.Permission) error
}

type ApprovalService struct {
	ApprovalRepo   *repository.ApprovalRepository
	PermRepo       *repository.PermissionRepository
	Provisioning   DBUserProvisioner
	InstanceRepo   *repository.InstanceRepository
	UserRepo       *repository.UserRepository
}

// NewApprovalService constructs an ApprovalService with the given repositories and provisioning service.
func NewApprovalService(
	approvalRepo *repository.ApprovalRepository,
	permRepo *repository.PermissionRepository,
	provisioningService DBUserProvisioner,
	instanceRepo *repository.InstanceRepository,
	userRepo *repository.UserRepository,
) *ApprovalService {
	return &ApprovalService{
		ApprovalRepo: approvalRepo,
		PermRepo:     permRepo,
		Provisioning: provisioningService,
		InstanceRepo: instanceRepo,
		UserRepo:     userRepo,
	}
}

// ApplyApprovalSideEffects creates Permission records and provisions the DB user with grants for the approved request.
// Expects request.RequestedItems to be valid JSON array of RequestedItem. On any failure returns an error (caller should rollback status).
func (s *ApprovalService) ApplyApprovalSideEffects(request *models.ApprovalRequest) error {
	if len(request.RequestedItems) == 0 {
		return errors.New("requested items is empty")
	}
	var items []models.RequestedItem
	if err := json.Unmarshal(request.RequestedItems, &items); err != nil {
		return fmt.Errorf("invalid requested items JSON: %w", err)
	}
	if len(items) == 0 {
		return errors.New("requested items is empty")
	}

	requester, err := s.UserRepo.FindByID(request.RequesterID)
	if err != nil {
		return fmt.Errorf("load requester: %w", err)
	}

	grantedBy := uuid.Nil
	if request.ReviewedBy != nil {
		grantedBy = *request.ReviewedBy
	}

	for _, item := range items {
		instance, err := s.InstanceRepo.FindByID(item.InstanceID)
		if err != nil {
			return fmt.Errorf("load instance %s: %w", item.InstanceID, err)
		}

		perm := &models.Permission{
			UserID:     &requester.ID,
			InstanceID: &item.InstanceID,
			Database:   item.Database,
			Schema:     "public",
			Table:      item.Table,
			Privileges: pq.StringArray(item.Privileges),
			Type:       "permanent",
			GrantedBy:  grantedBy,
		}
		if err := s.PermRepo.Create(perm); err != nil {
			return fmt.Errorf("create permission: %w", err)
		}

		cred, err := s.Provisioning.ProvisionDBUser(requester, instance)
		if err != nil {
			return fmt.Errorf("provision DB user: %w", err)
		}

		perms := []models.Permission{*perm}
		if err := s.Provisioning.GrantPermissions(cred, perms); err != nil {
			return fmt.Errorf("grant permissions: %w", err)
		}
	}
	return nil
}

// CreateRequest creates an approval request with the given metadata, permissions JSON, and requested items JSON.
// requestedItemsJSON is stored in ApprovalRequest.RequestedItems ([]RequestedItem as JSON).
func (s *ApprovalService) CreateRequest(requesterID uuid.UUID, reqType, description, justification string, permissions []byte, requestedItemsJSON []byte) (*models.ApprovalRequest, error) {
	request := &models.ApprovalRequest{
		Type:                 reqType,
		RequesterID:          requesterID,
		Description:          description,
		Justification:        justification,
		RequestedPermissions: permissions,
		RequestedItems:       requestedItemsJSON,
		Status:               "pending",
		ExpiresAt:            time.Now().Add(24 * time.Hour), // Default 24h expiry
	}

	if err := s.ApprovalRepo.Create(request); err != nil {
		return nil, err
	}
	return request, nil
}

func (s *ApprovalService) ApproveRequest(requestID, reviewerID uuid.UUID) (*models.ApprovalRequest, error) {
	request, err := s.ApprovalRepo.FindByID(requestID)
	if err != nil {
		return nil, err
	}

	if request.Status != "pending" {
		return nil, errors.New("request is not pending")
	}

	now := time.Now()
	request.Status = "approved"
	request.ReviewedBy = &reviewerID
	request.ReviewedAt = &now

	if err := s.ApprovalRepo.Update(request); err != nil {
		return nil, err
	}

	if err := s.ApplyApprovalSideEffects(request); err != nil {
		request.Status = "pending"
		request.ReviewedBy = nil
		request.ReviewedAt = nil
		_ = s.ApprovalRepo.Update(request)
		return nil, err
	}

	return request, nil
}

func (s *ApprovalService) RejectRequest(requestID, reviewerID uuid.UUID, reason string) (*models.ApprovalRequest, error) {
	request, err := s.ApprovalRepo.FindByID(requestID)
	if err != nil {
		return nil, err
	}

	if request.Status != "pending" {
		return nil, errors.New("request is not pending")
	}

	now := time.Now()
	request.Status = "rejected"
	request.ReviewedBy = &reviewerID
	request.ReviewedAt = &now
	request.RejectionReason = reason

	if err := s.ApprovalRepo.Update(request); err != nil {
		return nil, err
	}

	return request, nil
}

func (s *ApprovalService) GetPendingRequests() ([]models.ApprovalRequest, error) {
	return s.ApprovalRepo.FindByStatus("pending")
}

func (s *ApprovalService) GetAllRequests() ([]models.ApprovalRequest, error) {
	return s.ApprovalRepo.FindAll()
}
