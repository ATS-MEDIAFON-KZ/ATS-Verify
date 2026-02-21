package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Kazpost  KazpostConfig
	CDEK     CDEKConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port string
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// JWTConfig holds JWT token settings.
type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

// KazpostConfig holds Kazpost API credentials.
type KazpostConfig struct {
	APIKey string
}

// CDEKConfig holds CDEK API credentials.
type CDEKConfig struct {
	ClientID     string
	ClientSecret string
}

// DSN returns the PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	// Attempt to load .env file; it's okay if it doesn't exist (e.g. in Docker)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	dbPort, err := strconv.Atoi(getEnv("POSTGRES_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid POSTGRES_PORT: %w", err)
	}

	jwtHours, err := strconv.Atoi(getEnv("JWT_EXPIRATION_HOURS", "24"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRATION_HOURS: %w", err)
	}

	return &Config{
		Server: ServerConfig{
			Port: getEnv("APP_PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     dbPort,
			User:     getEnv("POSTGRES_USER", "ats_admin"),
			Password: getEnv("POSTGRES_PASSWORD", ""),
			DBName:   getEnv("POSTGRES_DB", "ats_verify"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", ""),
			Expiration: time.Duration(jwtHours) * time.Hour,
		},
		Kazpost: KazpostConfig{
			APIKey: getEnv("KAZPOST_API_KEY", ""),
		},
		CDEK: CDEKConfig{
			ClientID:     getEnv("CDEK_CLIENT_ID", ""),
			ClientSecret: getEnv("CDEK_CLIENT_SECRET", ""),
		},
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
