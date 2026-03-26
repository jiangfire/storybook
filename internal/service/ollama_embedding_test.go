package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOllamaEmbedding_EmbedText 测试 Ollama 单个文本向量化
func TestOllamaEmbedding_EmbedText(t *testing.T) {
	// Arrange - 需要本地运行 Ollama 服务
	svc := NewOllamaEmbedding("http://localhost:11434", "nomic-embed-text")
	ctx := context.Background()

	// Act
	embedding, err := svc.EmbedText(ctx, "测试文本")

	// Assert
	if err != nil {
		t.Skip("Ollama 服务不可用，跳过测试")
	}
	require.NoError(t, err)
	assert.NotEmpty(t, embedding)
	assert.Greater(t, len(embedding), 0)
}

// TestOllamaEmbedding_EmbedBatch 测试 Ollama 批量向量化
func TestOllamaEmbedding_EmbedBatch(t *testing.T) {
	// Arrange
	svc := NewOllamaEmbedding("http://localhost:11434", "nomic-embed-text")
	ctx := context.Background()
	texts := []string{"文本1", "文本2", "文本3"}

	// Act
	embeddings, err := svc.EmbedBatch(ctx, texts)

	// Assert
	if err != nil {
		t.Skip("Ollama 服务不可用，跳过测试")
	}
	require.NoError(t, err)
	assert.Len(t, embeddings, 3)
	for _, emb := range embeddings {
		assert.NotEmpty(t, emb)
	}
}

// TestOllamaEmbedding_EmptyText 测试空文本错误处理
func TestOllamaEmbedding_EmptyText(t *testing.T) {
	// Arrange
	svc := NewOllamaEmbedding("http://localhost:11434", "nomic-embed-text")
	ctx := context.Background()

	// Act & Assert
	_, err := svc.EmbedText(ctx, "")
	assert.Error(t, err)
	assert.Equal(t, ErrEmptyText, err)
}

// TestOllamaEmbedding_EmptyBatch 测试空批量错误处理
func TestOllamaEmbedding_EmptyBatch(t *testing.T) {
	// Arrange
	svc := NewOllamaEmbedding("http://localhost:11434", "nomic-embed-text")
	ctx := context.Background()

	// Act & Assert
	_, err := svc.EmbedBatch(ctx, []string{})
	assert.Error(t, err)
	assert.Equal(t, ErrEmptyText, err)
}

// TestOllamaEmbedding_GetDimension 测试获取向量维度
func TestOllamaEmbedding_GetDimension(t *testing.T) {
	// Arrange
	svc := NewOllamaEmbedding("http://localhost:11434", "nomic-embed-text")

	// Act & Assert
	// nomic-embed-text 默认维度是 768
	assert.Equal(t, 768, svc.GetDimension())
}

// TestOllamaEmbedding_CustomDimension 测试自定义维度
func TestOllamaEmbedding_CustomDimension(t *testing.T) {
	// Arrange
	svc := NewOllamaEmbeddingWithDimension("http://localhost:11434", "mxbai-embed-large", 1024)

	// Act & Assert
	assert.Equal(t, 1024, svc.GetDimension())
}

// TestEmbeddingService_OllamaContract 测试 Ollama 接口契约
func TestEmbeddingService_OllamaContract(t *testing.T) {
	// Arrange - 使用接口类型
	var svc EmbeddingService = NewOllamaEmbedding("http://localhost:11434", "nomic-embed-text")

	// Act
	dim := svc.GetDimension()

	// Assert
	assert.Equal(t, 768, dim)
}
