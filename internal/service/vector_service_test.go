package service

import (
	"context"
	"testing"

	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupTestDB 创建测试数据库（使用内存 SQLite）
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 注意：SQLite 不支持 pgvector，所以这里只创建表结构
	// 实际的向量搜索测试需要使用 PostgreSQL
	err = db.AutoMigrate(&model.UserStory{})
	require.NoError(t, err)

	return db
}

// TestVectorService_SearchSimilarStories_Empty 测试空结果
func TestVectorService_SearchSimilarStories_Empty(t *testing.T) {
	t.Skip("跳过需要 PostgreSQL + pgvector 的测试")
	// Arrange
	db := setupTestDB(t)
	mockEmbedding := NewMockEmbedding()
	svc := NewVectorService(db, mockEmbedding)
	ctx := context.Background()

	// Act
	results, err := svc.SearchSimilarStories(ctx, "测试查询", []uint{1}, 10)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, results)
}

// TestVectorService_SearchSimilarStories_WithArchived 测试归档故事过滤
func TestVectorService_SearchSimilarStories_WithArchived(t *testing.T) {
	t.Skip("跳过需要 PostgreSQL + pgvector 的测试")
	// Arrange
	db := setupTestDB(t)
	mockEmbedding := NewMockEmbedding()
	svc := NewVectorService(db, mockEmbedding)
	ctx := context.Background()

	// 创建测试数据（归档的故事不应出现在结果中）
	archivedStory := &model.UserStory{
		ID:        1,
		ProjectID: 1,
		Title:     "已归档的功能",
		Archived:  true,
	}
	db.Create(archivedStory)

	// Act
	results, err := svc.SearchSimilarStories(ctx, "已归档的功能", []uint{1}, 10)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, results) // 归档的故事不应出现在搜索结果中
}

// TestVectorService_PrepareStoryContent 测试内容准备
func TestVectorService_PrepareStoryContent(t *testing.T) {
	// Arrange
	story := &model.UserStory{
		Title:       "用户登录",
		Description: "实现用户登录功能",
	}

	// Act
	svc := NewVectorService(nil, nil)
	content := svc.PrepareStoryContent(story)

	// Assert
	assert.Contains(t, content, "标题：用户登录")
	assert.Contains(t, content, "描述：实现用户登录功能")
}

// TestVectorService_PrepareStoryContent_WithAC 测试包含验收标准的内容准备
func TestVectorService_PrepareStoryContent_WithAC(t *testing.T) {
	// Arrange
	story := &model.UserStory{
		Title:       "用户注册",
		Description: "邮箱注册功能",
		AcceptanceCriteria: model.MarshalJSON([]model.AcceptanceCriterion{
			{ID: "1", Description: "支持邮箱输入", Status: "pending", Order: 1},
			{ID: "2", Description: "发送验证邮件", Status: "pending", Order: 2},
		}),
	}

	// Act
	svc := NewVectorService(nil, nil)
	content := svc.PrepareStoryContent(story)

	// Assert
	assert.Contains(t, content, "标题：用户注册")
	assert.Contains(t, content, "验收：支持邮箱输入")
	assert.Contains(t, content, "验收：发送验证邮件")
}

// TestVectorService_PrepareStoryContent_WithTags 测试包含标签的内容准备
func TestVectorService_PrepareStoryContent_WithTags(t *testing.T) {
	// Arrange
	story := &model.UserStory{
		Title: "支付功能",
		Tags:  model.MarshalJSON([]string{"支付", "订单"}),
	}

	// Act
	svc := NewVectorService(nil, nil)
	content := svc.PrepareStoryContent(story)

	// Assert
	assert.Contains(t, content, "标签：支付 订单")
}

// TestVectorService_ToSimilarStories 测试结果转换
func TestVectorService_ToSimilarStories(t *testing.T) {
	// Arrange
	mockEmbedding := NewMockEmbedding()
	svc := NewVectorService(nil, mockEmbedding).(*vectorService)

	// 使用匿名结构体（与实现中的类型一致）
	results := []struct {
		model.UserStory
		Similarity float64
	}{
		{
			UserStory:  model.UserStory{ID: 1, Title: "故事1"},
			Similarity: 0.95,
		},
		{
			UserStory:  model.UserStory{ID: 2, Title: "故事2"},
			Similarity: 0.87,
		},
	}

	// Act
	similarStories := svc.toSimilarStories(results)

	// Assert
	assert.Len(t, similarStories, 2)
	assert.Equal(t, uint(1), similarStories[0].Story.(model.UserStory).ID)
	assert.Equal(t, 0.95, similarStories[0].Similarity)
	assert.Equal(t, uint(2), similarStories[1].Story.(model.UserStory).ID)
	assert.Equal(t, 0.87, similarStories[1].Similarity)
}

// TestCosineSimilarity 测试余弦相似度计算
func TestCosineSimilarity(t *testing.T) {
	// Arrange
	v1 := []float32{1, 0, 0}
	v2 := []float32{1, 0, 0}
	v3 := []float32{0, 1, 0}
	v4 := []float32{-1, 0, 0}

	// Act & Assert
	// 相同向量，相似度为 1
	got, err := cosineSimilarity(v1, v2)
	require.NoError(t, err)
	assert.InDelta(t, 1.0, got, 0.001)
	// 正交向量，相似度为 0
	got, err = cosineSimilarity(v1, v3)
	require.NoError(t, err)
	assert.InDelta(t, 0.0, got, 0.001)
	// 相反向量，相似度为 -1
	got, err = cosineSimilarity(v1, v4)
	require.NoError(t, err)
	assert.InDelta(t, -1.0, got, 0.001)
}

// TestCosineSimilarity_ZeroVector 测试零向量返回 0（避免除零）
func TestCosineSimilarity_ZeroVector(t *testing.T) {
	zero := []float32{0, 0, 0}
	v := []float32{1, 2, 3}

	got, err := cosineSimilarity(zero, v)
	require.NoError(t, err)
	assert.Equal(t, 0.0, got)

	got, err = cosineSimilarity(v, zero)
	require.NoError(t, err)
	assert.Equal(t, 0.0, got)
}

// TestCosineSimilarity_DifferentLength 测试不同长度向量返回错误而非 panic
func TestCosineSimilarity_DifferentLength(t *testing.T) {
	// Arrange
	v1 := []float32{1, 2, 3}
	v2 := []float32{1, 2} // 不同长度

	// Act
	got, err := cosineSimilarity(v1, v2)

	// Assert - 应该返回错误，不应该 panic
	require.Error(t, err)
	assert.Equal(t, 0.0, got)
	assert.Contains(t, err.Error(), "dimension mismatch")
}

// TestBatchIndexStories_ContinuesOnError 测试批量索引时单个失败不应中断整个批次
func TestBatchIndexStories_ContinuesOnError(t *testing.T) {
	// 这个测试验证：当某条故事的索引失败时，
	// 其他故事应该继续被索引，而不是整个批次都失败
	t.Skip("需要实现错误隔离机制后启用")

	// Arrange
	db := setupTestDB(t)

	// 创建测试故事
	story1 := &model.UserStory{ID: 1, ProjectID: 1, Title: "故事1"}
	story2 := &model.UserStory{ID: 2, ProjectID: 1, Title: "故事2"}
	story3 := &model.UserStory{ID: 3, ProjectID: 1, Title: "故事3"}
	db.Create(story1)
	db.Create(story2)
	db.Create(story3)

	stories := []model.UserStory{*story1, *story2, *story3}
	mockEmbedding := NewMockEmbedding()
	svc := NewVectorService(db, mockEmbedding)
	ctx := context.Background()

	// Act - 索引多个故事
	err := svc.BatchIndexStories(ctx, stories)

	// Assert - 即使某个失败，其他也应该成功
	// 当前实现会返回错误，但理想情况是：
	// 1. 记录失败的story
	// 2. 继续处理其他story
	// 3. 返回部分成功的结果
	require.NoError(t, err)
}

// TestIndexBatch_PanicRecovery 测试panic恢复机制
func TestIndexBatch_PanicRecovery(t *testing.T) {
	// 验证即使发生panic，服务也不应崩溃
	// 而应该记录错误并继续
	t.Skip("需要实现panic恢复后启用")

	// Arrange
	_ = setupTestDB(t)
	_ = NewMockEmbedding()

	// Act & Assert
	// 当前实现会panic并中断服务
	// 期望：捕获panic，记录日志，继续处理
	t.Log("✅ 应该捕获panic而不中断服务")
}
