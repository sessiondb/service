// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package api

import (
	"sessiondb/internal/engine"
	"sessiondb/internal/repository"
	"sessiondb/internal/service"

	"gorm.io/gorm"
)

// PremiumDeps holds dependencies required by premium (pro) routes.
// Passed from main so provider_pro can wire Session, Alert, and Report engines without main being build-tag aware.
type PremiumDeps struct {
	PermRepo      *repository.PermissionRepository
	InstanceRepo  *repository.InstanceRepository
	AccessEngine  engine.AccessEngine
	DB            *gorm.DB
	QueryService  *service.QueryService // optional: for registering post-query alert hook
}
