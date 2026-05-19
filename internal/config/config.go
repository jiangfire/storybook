package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config 应用配置。
type Config struct {
	ServerAddr      string
	DBDriver        string
	DBDSN           string
	JWTSecret       string
	LogLevel        string
	LogFormat       string
	AccessTokenTTL  int // 小时
	RefreshTokenTTL int // 小时

	// 数据库连接池
	DBMaxOpenConns           int  // 最大打开连接数
	DBMaxIdleConns           int  // 最大空闲连接数
	DBConnMaxLifetimeMinutes int  // 连接最大生存时间（分钟）
	DBAutoMigrate            bool // 是否在启动时执行 AutoMigrate（生产建议关闭，使用独立迁移工具）

	// Vector / Embedding (P3.3)
	EmbeddingProvider string // EMBEDDING_PROVIDER：openai / ollama / 空（禁用）
	OpenAIAPIKey      string // OPENAI_API_KEY：embedding service 使用（与 AIConfig.APIKey 分离）
	OllamaURL         string // OLLAMA_URL：Ollama HTTP endpoint
	OllamaModel       string // OLLAMA_MODEL：embedding 模型名
	OllamaDimension   int    // OLLAMA_DIMENSION：embedding 向量维度（<=0 表示未设置）

	// AI 限流
	AIUserRateLimitPerMin int // AI_USER_RATE_LIMIT_PER_MIN：每用户每分钟 AI 调用次数

	// Metrics
	MetricsUser string // /metrics Basic Auth 用户名（为空则不注册 /metrics）
	MetricsPass string // /metrics Basic Auth 密码
}

func Load() (*Config, error) {
	return load(true)
}

func LoadForBootstrap() (*Config, error) {
	return load(false)
}

func LoadDotEnv() {
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		_ = godotenv.Load(filepath.Join(exeDir, ".env"))
	}
	_ = godotenv.Load()
}

func load(requireJWT bool) (*Config, error) {
	maxOpen, err := getEnvInt("DB_MAX_OPEN_CONNS", 50)
	if err != nil {
		return nil, err
	}
	maxIdle, err := getEnvInt("DB_MAX_IDLE_CONNS", 10)
	if err != nil {
		return nil, err
	}
	connLifetime, err := getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 30)
	if err != nil {
		return nil, err
	}
	autoMigrate, err := getEnvBool("DB_AUTO_MIGRATE", true)
	if err != nil {
		return nil, err
	}

	accessTTL, err := getEnvInt("ACCESS_TOKEN_TTL_HOURS", 24)
	if err != nil {
		return nil, err
	}
	refreshTTL, err := getEnvInt("REFRESH_TOKEN_TTL_HOURS", 24*7)
	if err != nil {
		return nil, err
	}

	ollamaDim, err := getEnvInt("OLLAMA_DIMENSION", 0)
	if err != nil {
		return nil, err
	}
	aiUserLimit, err := getEnvInt("AI_USER_RATE_LIMIT_PER_MIN", 10)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		ServerAddr:               normalizeServerAddr(getEnv("SERVER_ADDR", ":8080")),
		DBDriver:                 getEnv("DB_DRIVER", "sqlite"),
		DBDSN:                    getEnv("DB_DSN", "storybook.db"),
		JWTSecret:                strings.TrimSpace(os.Getenv("JWT_SECRET")),
		LogLevel:                 getEnv("LOG_LEVEL", "info"),
		LogFormat:                getEnv("LOG_FORMAT", "text"),
		AccessTokenTTL:           accessTTL,
		RefreshTokenTTL:          refreshTTL,
		DBMaxOpenConns:           maxOpen,
		DBMaxIdleConns:           maxIdle,
		DBConnMaxLifetimeMinutes: connLifetime,
		DBAutoMigrate:            autoMigrate,
		EmbeddingProvider:        strings.TrimSpace(os.Getenv("EMBEDDING_PROVIDER")),
		OpenAIAPIKey:             strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		OllamaURL:                strings.TrimSpace(os.Getenv("OLLAMA_URL")),
		OllamaModel:              strings.TrimSpace(os.Getenv("OLLAMA_MODEL")),
		OllamaDimension:          ollamaDim,
		AIUserRateLimitPerMin:    aiUserLimit,
		MetricsUser:              strings.TrimSpace(os.Getenv("METRICS_USER")),
		MetricsPass:              strings.TrimSpace(os.Getenv("METRICS_PASS")),
	}

	if requireJWT {
		if cfg.JWTSecret == "" {
			return nil, fmt.Errorf("JWT_SECRET must be set")
		}
		if len(cfg.JWTSecret) < 32 {
			return nil, fmt.Errorf("JWT_SECRET must be at least 32 bytes")
		}
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %q is not a valid integer", key, v)
	}
	return n, nil
}

func getEnvBool(key string, fallback bool) (bool, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("invalid %s: %q is not a valid boolean (use true/false/1/0)", key, v)
	}
	return b, nil
}

func normalizeServerAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ":8080"
	}
	if _, err := strconv.Atoi(addr); err == nil {
		return ":" + addr
	}
	return addr
}
