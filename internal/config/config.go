package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
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
	}

	return config, nil
}
