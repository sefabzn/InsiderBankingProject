// Package config handles environment configuration loading.
package config

import (
	"os"
	"strconv"
)

// Config holds all configuration values for the application.
type Config struct {
	Port           string
	Environment    string
	DBUrl          string
	JWTSecret      string
	AllowedOrigins string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:           getEnv("PORT", "8080"),
		Environment:    getEnv("ENV", "dev"),
		DBUrl:          getEnv("DB_URL", ""),
		JWTSecret:      getEnv("JWT_SECRET", ""),
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "*"),
	}
}

// getEnv reads an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetPortInt returns the port as an integer.
func (c *Config) GetPortInt() int {
	port, err := strconv.Atoi(c.Port)
	if err != nil {
		return 8080
	}
	return port
}

// GetAddr returns the full address string for the server.
func (c *Config) GetAddr() string {
	return ":" + c.Port
}
