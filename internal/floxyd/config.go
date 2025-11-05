package floxyd

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	Workers        int
	WorkerInterval time.Duration
}

func LoadConfig() (*Config, error) {
	config := &Config{
		DBHost:         getEnvOrDefault("FLOXY_DB_HOST", ""),
		DBPort:         getEnvOrDefault("FLOXY_DB_PORT", ""),
		DBUser:         getEnvOrDefault("FLOXY_DB_USER", ""),
		DBPassword:     os.Getenv("FLOXY_DB_PASSWORD"),
		DBName:         getEnvOrDefault("FLOXY_DB_NAME", ""),
		Workers:        getIntEnvOrDefault("FLOXY_WORKERS", 3),
		WorkerInterval: getDurationEnvOrDefault("FLOXY_WORKER_INTERVAL", 100*time.Millisecond),
	}

	if config.DBHost == "" {
		return nil, fmt.Errorf("FLOXY_DB_HOST environment variable is required")
	}
	if config.DBPort == "" {
		return nil, fmt.Errorf("FLOXY_DB_PORT environment variable is required")
	}
	if config.DBUser == "" {
		return nil, fmt.Errorf("FLOXY_DB_USER environment variable is required")
	}
	if config.DBName == "" {
		return nil, fmt.Errorf("FLOXY_DB_NAME environment variable is required")
	}

	return config, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnvOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationEnvOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
