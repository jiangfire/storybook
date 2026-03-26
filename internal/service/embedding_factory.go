package service

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// EmbeddingConfig Embedding 服务配置
type EmbeddingConfig struct {
	Provider        string // "mock", "openai", "ollama"
	OpenAIKey       string // OpenAI API Key
	OllamaURL       string // Ollama 服务地址
	OllamaModel     string // Ollama 模型名称
	OllamaDimension int    // Ollama 模型维度
}

// NewEmbeddingServiceFromConfig 根据配置创建 Embedding 服务（工厂模式）
func NewEmbeddingServiceFromConfig(config EmbeddingConfig) (EmbeddingService, error) {
	switch config.Provider {
	case "openai":
		if config.OpenAIKey == "" {
			config.OpenAIKey = os.Getenv("OPENAI_API_KEY")
		}
		if config.OpenAIKey == "" {
			return nil, fmt.Errorf("OpenAI API Key 未提供")
		}
		return NewOpenAIEmbedding(config.OpenAIKey), nil

	case "ollama":
		if config.OllamaURL == "" {
			config.OllamaURL = "http://localhost:11434"
		}
		if config.OllamaModel == "" {
			config.OllamaModel = "nomic-embed-text"
		}
		if config.OllamaDimension > 0 {
			return NewOllamaEmbeddingWithDimension(config.OllamaURL, config.OllamaModel, config.OllamaDimension), nil
		}
		return NewOllamaEmbedding(config.OllamaURL, config.OllamaModel), nil

	case "mock":
		return NewMockEmbedding(), nil

	default:
		return nil, fmt.Errorf("不支持的 provider: %s（支持: mock, openai, ollama）", config.Provider)
	}
}

// NewEmbeddingServiceFromEnv 从环境变量创建 Embedding 服务
func NewEmbeddingServiceFromEnv() (EmbeddingService, error) {
	config := EmbeddingConfig{
		Provider:        os.Getenv("EMBEDDING_PROVIDER"),
		OpenAIKey:       os.Getenv("OPENAI_API_KEY"),
		OllamaURL:       os.Getenv("OLLAMA_URL"),
		OllamaModel:     os.Getenv("OLLAMA_MODEL"),
		OllamaDimension: ReadPositiveIntEnv("OLLAMA_DIMENSION"),
	}

	if config.Provider == "" {
		config.Provider = "mock" // 默认使用 mock
	}

	return NewEmbeddingServiceFromConfig(config)
}

func ReadPositiveIntEnv(key string) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0
	}
	return value
}
