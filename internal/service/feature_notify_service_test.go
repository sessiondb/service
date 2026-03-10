// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

import (
	"sessiondb/internal/repository"
	"testing"
)

func TestFeatureNotifyService_RegisterRequest_EmptyInput(t *testing.T) {
	// Nil repo is never used when email or featureKey is empty (early return)
	svc := NewFeatureNotifyService(repository.NewFeatureNotifyRepository(nil))

	created, already, err := svc.RegisterRequest("", "sessions", nil)
	if err != nil {
		t.Fatalf("expected nil error for empty email, got %v", err)
	}
	if created != nil || already {
		t.Fatal("expected nil created and not already for empty email")
	}

	created, already, err = svc.RegisterRequest("u@test.com", "", nil)
	if err != nil {
		t.Fatalf("expected nil error for empty featureKey, got %v", err)
	}
	if created != nil || already {
		t.Fatal("expected nil created and not already for empty featureKey")
	}
}
