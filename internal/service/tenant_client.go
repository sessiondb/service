// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

// Feature represents a single tenant feature flag
type Feature struct {
	Enabled     bool   `json:"enabled"`
	Reason      string `json:"reason,omitempty"`
	MinimumPlan string `json:"minimumPlan,omitempty"`
}

// TenantFeatures models the features payload response
type TenantFeatures map[string]Feature

// TenantClient defines the interface for fetching tenant information
type TenantClient interface {
	GetFeaturesForTenant(tenantID string) (TenantFeatures, error)
}

// MockTenantClient is a placeholder implementation that returns static feature flags
type MockTenantClient struct{}

// NewMockTenantClient creates a new mock tenant client
func NewMockTenantClient() *MockTenantClient {
	return &MockTenantClient{}
}

// GetFeaturesForTenant simulates fetching feature flags for a tenant
func (m *MockTenantClient) GetFeaturesForTenant(tenantID string) (TenantFeatures, error) {
	// In the future, this will make an HTTP/gRPC call to the actual Tenant Service
	// For now, return a static mock payload matching the backend requirements
	return TenantFeatures{
		"audit_logs": {
			Enabled: true, // Community: view/create audit logs
		},
		"audit_logs_export": {
			Enabled:     false,
			Reason:      "plan_upgrade_required",
			MinimumPlan: "Pro",
		},
		"custom_db_roles": {
			Enabled: true,
		},
		"advanced_approvals": {
			Enabled:     false,
			MinimumPlan: "Enterprise",
		},
	}, nil
}
