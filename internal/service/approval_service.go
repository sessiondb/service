// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

import (
	"errors"
	"sessiondb/internal/models"
	"sessiondb/internal/repository"
	"time"

	"github.com/google/uuid"
)

type ApprovalService struct {
	ApprovalRepo *repository.ApprovalRepository
}

func NewApprovalService(approvalRepo *repository.ApprovalRepository) *ApprovalService {
	return &ApprovalService{ApprovalRepo: approvalRepo}
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

	// TODO: Trigger side effects (grant permissions, create temp user, etc.)

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
