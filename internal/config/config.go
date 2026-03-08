package config

import (
	"fmt"
	"os"
)

// Config 应用配置。
type Config struct {
	ServerAddr      string
	DBDriver        string
	DBDSN           string
	JWTSecret       string
	AccessTokenTTL  int // 小时
	RefreshTokenTTL int // 小时
}

func Load() (*Config, error) {
	cfg := &Config{
		ServerAddr:      getEnv("SERVER_ADDR", ":8080"),
		DBDriver:        getEnv("DB_DRIVER", "sqlite"),
		DBDSN:           getEnv("DB_DSN", "storybook.db"),
		JWTSecret:       getEnv("JWT_SECRET", "change-me-in-production"),
		AccessTokenTTL:  24,
		RefreshTokenTTL: 24 * 7,
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET must not be empty")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
