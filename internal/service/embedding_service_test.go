package service

import (
	"context"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewOpenAIEmbedding_DefaultConfig 测试默认配置初始化
func TestNewOpenAIEmbedding_DefaultConfig(t *testing.T) {
	// Arrange
	svc := NewOpenAIEmbedding("test-api-key")

	// Assert
	require.NotNil(t, svc.client)
	assert.Equal(t, string(openai.SmallEmbedding3), svc.model)
	assert.Equal(t, 1536, svc.dimension)
}

// TestNewOpenAIEmbeddingWithModel_CustomConfig 测试自定义配置初始化
func TestNewOpenAIEmbeddingWithModel_CustomConfig(t *testing.T) {
	// Arrange
	svc := NewOpenAIEmbeddingWithModel("test-api-key", "text-embedding-3-large", 3072)

	// Assert
	require.NotNil(t, svc.client)
	assert.Equal(t, "text-embedding-3-large", svc.model)
	assert.Equal(t, 3072, svc.dimension)
}

// TestOpenAIEmbedding_EmptyText 测试空文本错误处理
func TestOpenAIEmbedding_EmptyText(t *testing.T) {
	// Arrange
	svc := NewOpenAIEmbedding("test-api-key")
	ctx := context.Background()

	// Act & Assert
	_, err := svc.EmbedText(ctx, "")
	assert.Error(t, err)
	assert.Equal(t, ErrEmptyText, err)
}

// TestOpenAIEmbedding_EmptyBatch 测试空批量错误处理
func TestOpenAIEmbedding_EmptyBatch(t *testing.T) {
	// Arrange
	svc := NewOpenAIEmbedding("test-api-key")
	ctx := context.Background()

	// Act & Assert
	_, err := svc.EmbedBatch(ctx, []string{})
	assert.Error(t, err)
	assert.Equal(t, ErrEmptyText, err)
}

// TestOpenAIEmbedding_GetDimension 测试获取向量维度
func TestOpenAIEmbedding_GetDimension(t *testing.T) {
	// Arrange
	svc := NewOpenAIEmbedding("test-api-key")

	// Act & Assert
	assert.Equal(t, 1536, svc.GetDimension())
}

// TestMockEmbedding_EmbedText 测试 Mock 向量化
func TestMockEmbedding_EmbedText(t *testing.T) {
	// Arrange
	svc := NewMockEmbedding()
	ctx := context.Background()

	// Act
	embedding, err := svc.EmbedText(ctx, "任意文本")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1536, len(embedding))
	// Mock 应该返回确定性的结果
	assert.Equal(t, float32(0.1), embedding[0])
}

// TestMockEmbedding_EmbedBatch 测试 Mock 批量向量化
func TestMockEmbedding_EmbedBatch(t *testing.T) {
	// Arrange
	svc := NewMockEmbedding()
	ctx := context.Background()
	texts := []string{"文本1", "文本2"}

	// Act
	embeddings, err := svc.EmbedBatch(ctx, texts)

	// Assert
	require.NoError(t, err)
	assert.Len(t, embeddings, 2)
	for _, emb := range embeddings {
		assert.Equal(t, 1536, len(emb))
	}
}

// TestMockEmbedding_GetDimension 测试 Mock 维度
func TestMockEmbedding_GetDimension(t *testing.T) {
	// Arrange
	svc := NewMockEmbedding()

	// Act & Assert
	assert.Equal(t, 1536, svc.GetDimension())
}

// TestEmbeddingService_Contract 测试接口契约（依赖倒置原则）
func TestEmbeddingService_Contract(t *testing.T) {
	// Arrange - 使用接口类型
	var svc EmbeddingService = NewMockEmbedding()
	ctx := context.Background()

	// Act
	embedding, err := svc.EmbedText(ctx, "测试")
	dim := svc.GetDimension()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1536, len(embedding))
	assert.Equal(t, 1536, dim)
}
