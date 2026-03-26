package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var knownOllamaEmbeddingDimensions = map[string]int{
	"mxbai-embed-large": 1024,
	"nomic-embed-text":  768,
}

// OllamaEmbedding Ollama 向量化实现（纯 HTTP，支持本地模型）
type OllamaEmbedding struct {
	baseURL   string
	model     string
	dimension int
	client    *http.Client
}

// NewOllamaEmbedding 创建 Ollama Embedding 服务
func NewOllamaEmbedding(baseURL, model string) *OllamaEmbedding {
	dimension := knownOllamaEmbeddingDimensions[model]
	if dimension <= 0 {
		dimension = 768
	}
	return &OllamaEmbedding{
		baseURL:   baseURL,
		model:     model,
		dimension: dimension,
		client: &http.Client{
			Timeout: 30 * time.Second, // 设置 30 秒超时
		},
	}
}

// NewOllamaEmbeddingWithDimension 创建自定义维度的 Ollama Embedding 服务
func NewOllamaEmbeddingWithDimension(baseURL, model string, dimension int) *OllamaEmbedding {
	return &OllamaEmbedding{
		baseURL:   baseURL,
		model:     model,
		dimension: dimension,
		client: &http.Client{
			Timeout: 30 * time.Second, // 设置 30 秒超时
		},
	}
}

// EmbedText 将单个文本转换为向量
func (s *OllamaEmbedding) EmbedText(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, ErrEmptyText
	}

	// 构建请求
	reqBody := map[string]any{
		"model":  s.model,
		"prompt": text,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 发送 HTTP 请求
	url := fmt.Sprintf("%s/api/embeddings", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama api: status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("ollama: empty response")
	}

	return result.Embedding, nil
}

// EmbedBatch 批量将文本转换为向量
func (s *OllamaEmbedding) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyText
	}

	// Ollama 不支持原生批量，需要逐个调用（KISS：保持简单）
	result := make([][]float32, len(texts))
	for i, text := range texts {
		if text == "" {
			return nil, ErrEmptyText
		}

		embedding, err := s.EmbedText(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("ollama api batch[%d]: %w", i, err)
		}
		result[i] = embedding
	}

	return result, nil
}

// GetDimension 返回向量维度
func (s *OllamaEmbedding) GetDimension() int {
	return s.dimension
}
