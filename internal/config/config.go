// Package config provides configuration management via environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration settings.
type Config struct {
	// PostgreSQL настройки
	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string

	// S3 настройки
	S3Bucket    string
	S3Region    string
	S3AccessKey string
	S3SecretKey string
	S3Endpoint  string // Опционально, для совместимых с S3 хранилищ

	// Настройки бэкапа
	BackupPrefix string // Префикс для файлов в S3
	Compress     bool   // Сжимать ли бэкап
}

// Load загружает конфигурацию из переменных окружения
func Load() (*Config, error) {
	// Пытаемся загрузить .env файл (опционально)
	_ = godotenv.Load()

	cfg := &Config{
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnvInt("DB_PORT", 5432),
		DBName:       getEnv("DB_NAME", ""),
		DBUser:       getEnv("DB_USER", "postgres"),
		DBPassword:   getEnv("DB_PASSWORD", ""),
		S3Bucket:     getEnv("S3_BUCKET", ""),
		S3Region:     getEnv("S3_REGION", "us-east-1"),
		S3AccessKey:  getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:  getEnv("S3_SECRET_KEY", ""),
		S3Endpoint:   getEnv("S3_ENDPOINT", ""),
		BackupPrefix: getEnv("BACKUP_PREFIX", "backups"),
		Compress:     getEnvBool("COMPRESS", true),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate проверяет обязательные параметры
func (c *Config) Validate() error {
	if c.DBName == "" {
		return fmt.Errorf("DB_NAME обязателен")
	}
	if c.S3Bucket == "" {
		return fmt.Errorf("S3_BUCKET обязателен")
	}
	if c.S3AccessKey == "" {
		return fmt.Errorf("S3_ACCESS_KEY обязателен")
	}
	if c.S3SecretKey == "" {
		return fmt.Errorf("S3_SECRET_KEY обязателен")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
