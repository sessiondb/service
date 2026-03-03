// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package service

import (
	"sync"
)

type ConfigService struct {
	authType string
	mu       sync.RWMutex
}

func NewConfigService() *ConfigService {
	return &ConfigService{
		authType: "password", // Default
	}
}

func (s *ConfigService) GetAuthConfig() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]string{"type": s.authType}
}

func (s *ConfigService) UpdateAuthConfig(authType string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authType = authType
}
