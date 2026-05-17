package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jiangfire/storybook/internal/metrics"
	"github.com/sashabaranov/go-openai"
)

// EmbeddingService 定义文本向量化接口（依赖倒置原则）
type EmbeddingService interface {
	// EmbedText 将文本转换为向量
	EmbedText(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch 批量转换文本
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// GetDimension 返回向量维度
	GetDimension() int
}

// OpenAIEmbedding OpenAI 向量化实现
type OpenAIEmbedding struct {
	client    *openai.Client
	model     string
	dimension int
}

// NewOpenAIEmbedding 创建 OpenAI Embedding 服务
func NewOpenAIEmbedding(apiKey string) *OpenAIEmbedding {
	return &OpenAIEmbedding{
		client:    openai.NewClient(apiKey),
		model:     string(openai.SmallEmbedding3), // text-embedding-3-small
		dimension: 1536,
	}
}

// NewOpenAIEmbeddingWithModel 创建自定义模型的 OpenAI Embedding 服务
func NewOpenAIEmbeddingWithModel(apiKey, model string, dimension int) *OpenAIEmbedding {
	return &OpenAIEmbedding{
		client:    openai.NewClient(apiKey),
		model:     model,
		dimension: dimension,
	}
}

// EmbedText 将单个文本转换为向量
func (s *OpenAIEmbedding) EmbedText(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, ErrEmptyText
	}

	start := time.Now()
	resp, err := s.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(s.model),
	})
	metrics.AICallDuration.WithLabelValues("embed_text", s.model).Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.AICallsTotal.WithLabelValues("embed_text", s.model, "error").Inc()
		return nil, fmt.Errorf("openai embeddings: %w", err)
	}
	metrics.AICallsTotal.WithLabelValues("embed_text", s.model, "success").Inc()
	metrics.AITokensTotal.WithLabelValues("embed_text", s.model, "prompt").Add(float64(resp.Usage.PromptTokens))

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("openai: empty response")
	}

	return resp.Data[0].Embedding, nil
}

// EmbedBatch 批量将文本转换为向量
func (s *OpenAIEmbedding) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyText
	}

	start := time.Now()
	resp, err := s.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: texts,
		Model: openai.EmbeddingModel(s.model),
	})
	metrics.AICallDuration.WithLabelValues("embed_batch", s.model).Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.AICallsTotal.WithLabelValues("embed_batch", s.model, "error").Inc()
		return nil, fmt.Errorf("openai embeddings batch: %w", err)
	}
	metrics.AICallsTotal.WithLabelValues("embed_batch", s.model, "success").Inc()
	metrics.AITokensTotal.WithLabelValues("embed_batch", s.model, "prompt").Add(float64(resp.Usage.PromptTokens))

	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("openai: expected %d embeddings, got %d", len(texts), len(resp.Data))
	}

	result := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		result[i] = data.Embedding
	}
	return result, nil
}

// GetDimension 返回向量维度
func (s *OpenAIEmbedding) GetDimension() int {
	return s.dimension
}

// MockEmbedding 测试用 Mock 实现
type MockEmbedding struct {
	dimension int
	value     float32 // 用于返回确定性的值
}

// NewMockEmbedding 创建 Mock Embedding 服务
func NewMockEmbedding() *MockEmbedding {
	return &MockEmbedding{
		dimension: 1536,
		value:     0.1,
	}
}

// NewMockEmbeddingWithDimension 创建自定义维度的 Mock Embedding 服务
func NewMockEmbeddingWithDimension(dimension int, value float32) *MockEmbedding {
	return &MockEmbedding{
		dimension: dimension,
		value:     value,
	}
}

// EmbedText 将文本转换为 Mock 向量
func (m *MockEmbedding) EmbedText(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, ErrEmptyText
	}

	// 返回确定性的向量用于测试
	result := make([]float32, m.dimension)
	for i := range result {
		result[i] = m.value
	}
	return result, nil
}

// EmbedBatch 批量将文本转换为 Mock 向量
func (m *MockEmbedding) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyText
	}

	result := make([][]float32, len(texts))
	for i := range texts {
		result[i], _ = m.EmbedText(ctx, texts[i])
	}
	return result, nil
}

// GetDimension 返回向量维度
func (m *MockEmbedding) GetDimension() int {
	return m.dimension
}
