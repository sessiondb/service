// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_DefaultLogins(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "config.toml")
	tomlContent := `
[auth]
[[auth.default_logins]]
email = "admin@example.com"
password = "admin123"
role_key = "super_admin"

[[auth.default_logins]]
email = "guest@example.com"
password = "guest123"
role_key = "analyst"
`
	if err := os.WriteFile(tomlPath, []byte(tomlContent), 0600); err != nil {
		t.Fatalf("write temp TOML: %v", err)
	}

	orig := os.Getenv("CONFIG_TOML_PATH")
	os.Setenv("CONFIG_TOML_PATH", tomlPath)
	defer func() {
		if orig == "" {
			os.Unsetenv("CONFIG_TOML_PATH")
		} else {
			os.Setenv("CONFIG_TOML_PATH", orig)
		}
	}()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.DefaultLogins == nil {
		t.Fatal("expected DefaultLogins to be populated, got nil")
	}
	if len(cfg.DefaultLogins) != 2 {
		t.Fatalf("expected 2 default logins, got %d", len(cfg.DefaultLogins))
	}
	if cfg.DefaultLogins[0].Email != "admin@example.com" || cfg.DefaultLogins[0].Password != "admin123" || cfg.DefaultLogins[0].RoleKey != "super_admin" {
		t.Errorf("first login: got %+v", cfg.DefaultLogins[0])
	}
	if cfg.DefaultLogins[1].Email != "guest@example.com" || cfg.DefaultLogins[1].Password != "guest123" || cfg.DefaultLogins[1].RoleKey != "analyst" {
		t.Errorf("second login: got %+v", cfg.DefaultLogins[1])
	}
}
