// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMigrateTokenAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "migrate-secret-123"
	old := os.Getenv("MIGRATE_TOKEN")
	defer func() { _ = os.Setenv("MIGRATE_TOKEN", old) }()

	tests := []struct {
		name           string
		setEnv         string
		header         string
		headerValue    string
		wantStatus     int
		wantNextCalled bool
	}{
		{"no token when env set", secret, "", "", http.StatusUnauthorized, false},
		{"wrong token", secret, "X-Migrate-Token", "wrong", http.StatusUnauthorized, false},
		{"valid X-Migrate-Token", secret, "X-Migrate-Token", secret, http.StatusOK, true},
		{"valid Authorization Bearer", secret, "Authorization", "Bearer " + secret, http.StatusOK, true},
		{"env unset returns 503", "", "X-Migrate-Token", secret, http.StatusServiceUnavailable, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("MIGRATE_TOKEN", tt.setEnv)
			r := gin.New()
			nextCalled := false
			r.POST("/migrate", MigrateTokenAuth(), func(c *gin.Context) {
				nextCalled = true
				c.Status(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodPost, "/migrate", nil)
			if tt.header != "" {
				req.Header.Set(tt.header, tt.headerValue)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if nextCalled != tt.wantNextCalled {
				t.Errorf("nextCalled = %v, want %v", nextCalled, tt.wantNextCalled)
			}
		})
	}
}
