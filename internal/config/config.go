// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func getwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return "<error>"
	}
	return dir
}

// resolveConfigTOMLPath returns the path to the single config.toml (scli init standard).
// Order: CONFIG_TOML_PATH env, then SESSIONDB_CONFIG_DIR/config.toml, then cwd config.default.toml.
func resolveConfigTOMLPath() string {
	if p := os.Getenv("CONFIG_TOML_PATH"); p != "" {
		return p
	}
	if d := os.Getenv("SESSIONDB_CONFIG_DIR"); d != "" {
		return filepath.Join(d, "config.toml")
	}
	return ""
}

// DefaultLogin represents a single default login entry from TOML auth.default_logins.
type DefaultLogin struct {
	Email    string `mapstructure:"email"`
	Password string `mapstructure:"password"`
	RoleKey  string `mapstructure:"role_key"`
}

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	JWT         JWTConfig
	Mail        MailConfig
	DefaultLogins []DefaultLogin
}

// MailConfig for sending credentials email on user create. When disabled, SendCredentialsEmail is a no-op.
type MailConfig struct {
	Enabled  bool
	From     string
	AppURL   string // Base URL for login link (e.g. https://app.sessiondb.io)
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
}

type ServerConfig struct {
	Port string
	Mode string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret        string
	ExpiryHours   int
	RefreshExpiry int
}

// getStr returns the first non-empty value from viper for the given keys (env key first, then TOML key).
func getStr(v *viper.Viper, keys ...string) string {
	for _, k := range keys {
		if s := v.GetString(k); s != "" {
			return s
		}
	}
	return ""
}

// getInt returns the first set value from viper for the given keys (so 0 is valid for redis.db).
func getInt(v *viper.Viper, keys ...string) int {
	for _, k := range keys {
		if v.IsSet(k) {
			return v.GetInt(k)
		}
	}
	return 0
}

func LoadConfig() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()

	// Defaults
	v.SetDefault("SERVER_PORT", "8080")
	v.SetDefault("SERVER_MODE", "debug")
	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", "5432")
	v.SetDefault("DB_SSLMODE", "disable")
	v.SetDefault("REDIS_ADDR", "localhost:6379")
	v.SetDefault("JWT_EXPIRY_HOURS", 24)
	v.SetDefault("JWT_REFRESH_EXPIRY", 720)
	v.SetDefault("MAIL_ENABLED", false)
	v.SetDefault("MAIL_SMTP_PORT", 587)

	// Prefer single config.toml (scli init standard); fallback to .env
	tomlPath := resolveConfigTOMLPath()
	if tomlPath != "" {
		if _, err := os.Stat(tomlPath); err == nil {
			v.SetConfigFile(tomlPath)
			if err := v.ReadInConfig(); err != nil {
				log.Printf("Could not load config.toml %s: %v. Falling back to .env / env.", tomlPath, err)
			} else {
				log.Printf("Loaded config from %s", tomlPath)
			}
		}
	}
	if !v.IsSet("server.port") && !v.IsSet("SERVER_PORT") {
		v.SetConfigFile(".env")
		_ = v.ReadInConfig()
	}

	config := &Config{
		Server: ServerConfig{
			Port: getStr(v, "SERVER_PORT", "server.port"),
			Mode: getStr(v, "SERVER_MODE", "server.mode"),
		},
		Database: DatabaseConfig{
			Host:     getStr(v, "DB_HOST", "database.host"),
			Port:     getStr(v, "DB_PORT", "database.port"),
			User:     getStr(v, "DB_USER", "database.user"),
			Password: getStr(v, "DB_PASSWORD", "database.password"),
			Name:     getStr(v, "DB_NAME", "database.name"),
			SSLMode:  getStr(v, "DB_SSLMODE", "database.ssl_mode"),
		},
		Redis: RedisConfig{
			Addr:     getStr(v, "REDIS_ADDR", "redis.addr"),
			Password: getStr(v, "REDIS_PASSWORD", "redis.password"),
			DB:       getInt(v, "REDIS_DB", "redis.db"),
		},
		JWT: JWTConfig{
			Secret:        getStr(v, "JWT_SECRET", "jwt.secret"),
			ExpiryHours:   getInt(v, "JWT_EXPIRY_HOURS", "jwt.expiry_hours"),
			RefreshExpiry: getInt(v, "JWT_REFRESH_EXPIRY", "jwt.refresh_expiry"),
		},
		Mail: MailConfig{
			Enabled:  v.GetBool("MAIL_ENABLED"),
			From:     getStr(v, "MAIL_FROM", "mail.from"),
			AppURL:   getStr(v, "APP_URL", "mail.app_url"),
			SMTPHost: getStr(v, "MAIL_SMTP_HOST", "mail.smtp_host"),
			SMTPPort: getInt(v, "MAIL_SMTP_PORT", "mail.smtp_port"),
			SMTPUser: getStr(v, "MAIL_SMTP_USER", "mail.smtp_user"),
			SMTPPass: getStr(v, "MAIL_SMTP_PASS", "mail.smtp_pass"),
		},
	}
	if config.Server.Port == "" {
		config.Server.Port = "8080"
	}
	if config.Server.Mode == "" {
		config.Server.Mode = "debug"
	}
	if config.JWT.ExpiryHours == 0 {
		config.JWT.ExpiryHours = 24
	}
	if config.JWT.RefreshExpiry == 0 {
		config.JWT.RefreshExpiry = 720
	}

	// [auth] default_logins from same config.toml or legacy config.default.toml
	if tomlPath != "" {
		var authSection struct {
			DefaultLogins []DefaultLogin `mapstructure:"default_logins"`
		}
		if err := v.UnmarshalKey("auth", &authSection); err == nil && len(authSection.DefaultLogins) > 0 {
			config.DefaultLogins = authSection.DefaultLogins
			log.Printf("Loaded %d default login(s) from %s", len(config.DefaultLogins), tomlPath)
		} else {
			_ = v.UnmarshalKey("auth.default_logins", &config.DefaultLogins)
			if len(config.DefaultLogins) > 0 {
				log.Printf("Loaded %d default login(s) from %s", len(config.DefaultLogins), tomlPath)
			}
		}
	}
	if len(config.DefaultLogins) == 0 {
		legacyPath := "config.default.toml"
		if tomlPath == "" && os.Getenv("SESSIONDB_CONFIG_DIR") != "" {
			legacyPath = filepath.Join(os.Getenv("SESSIONDB_CONFIG_DIR"), "config.default.toml")
		}
		if _, err := os.Stat(legacyPath); err == nil {
			v2 := viper.New()
			v2.SetConfigFile(legacyPath)
			if err := v2.ReadInConfig(); err == nil {
				_ = v2.UnmarshalKey("auth.default_logins", &config.DefaultLogins)
				if len(config.DefaultLogins) > 0 {
					log.Printf("Loaded %d default login(s) from %s", len(config.DefaultLogins), legacyPath)
				}
			}
		}
	}

	return config, nil
}
