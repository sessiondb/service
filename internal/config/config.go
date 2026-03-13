// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package config

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

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

func LoadConfig() (*Config, error) {
	// Check if .env file exists
	// Actually, easier to use viper's config search paths

	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("SERVER_MODE", "debug")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("JWT_EXPIRY_HOURS", 24)
	viper.SetDefault("JWT_REFRESH_EXPIRY", 720) // 30 days
	viper.SetDefault("MAIL_ENABLED", false)
	viper.SetDefault("MAIL_SMTP_PORT", 587)

	if err := viper.ReadInConfig(); err != nil {
		// Just log error and continue if file not found, as we rely on env vars too
		log.Printf("Could not load .env file: %v. Using environment variables.", err)
	}

	config := &Config{
		Server: ServerConfig{
			Port: viper.GetString("SERVER_PORT"),
			Mode: viper.GetString("SERVER_MODE"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetString("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			Name:     viper.GetString("DB_NAME"),
			SSLMode:  viper.GetString("DB_SSLMODE"),
		},
		Redis: RedisConfig{
			Addr:     viper.GetString("REDIS_ADDR"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		JWT: JWTConfig{
			Secret:        viper.GetString("JWT_SECRET"),
			ExpiryHours:   viper.GetInt("JWT_EXPIRY_HOURS"),
			RefreshExpiry: viper.GetInt("JWT_REFRESH_EXPIRY"),
		},
		Mail: MailConfig{
			Enabled:  viper.GetBool("MAIL_ENABLED"),
			From:     viper.GetString("MAIL_FROM"),
			AppURL:   viper.GetString("APP_URL"),
			SMTPHost: viper.GetString("MAIL_SMTP_HOST"),
			SMTPPort: viper.GetInt("MAIL_SMTP_PORT"),
			SMTPUser: viper.GetString("MAIL_SMTP_USER"),
			SMTPPass: viper.GetString("MAIL_SMTP_PASS"),
		},
	}

	// Optionally load [auth] default_logins from TOML (env/server remain from .env above).
	tomlPath := os.Getenv("CONFIG_TOML_PATH")
	if tomlPath == "" {
		tomlPath = "config.default.toml"
	}
	if _, err := os.Stat(tomlPath); err == nil {
		v := viper.New()
		v.SetConfigFile(tomlPath)
		if err := v.ReadInConfig(); err != nil {
			log.Printf("Could not load TOML config %s: %v. DefaultLogins left nil.", tomlPath, err)
		} else {
			var authSection struct {
				DefaultLogins []DefaultLogin `mapstructure:"default_logins"`
			}
			if err := v.UnmarshalKey("auth", &authSection); err != nil {
				log.Printf("Could not unmarshal auth from %s: %v. DefaultLogins left nil.", tomlPath, err)
			} else {
				config.DefaultLogins = authSection.DefaultLogins
			}
		}
	}

	return config, nil
}
