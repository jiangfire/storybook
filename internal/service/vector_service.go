package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"git.neolidy.top/neo/storybook/internal/model"
	"gorm.io/gorm"
)

// VectorService 向量搜索服务接口
type VectorService interface {
	// SearchSimilarStories 搜索相似故事
	SearchSimilarStories(ctx context.Context, query string, projectIDs []uint, limit int) ([]SimilarStory, error)

	// IndexStory 为单个故事生成并存储向量
	IndexStory(ctx context.Context, story *model.UserStory) error

	// BatchIndexStories 批量为故事生成并存储向量
	BatchIndexStories(ctx context.Context, stories []model.UserStory) error

	// PrepareStoryContent 准备用于向量化的文本内容
	PrepareStoryContent(story *model.UserStory) string
}

// vectorService 向量搜索服务实现
type vectorService struct {
	db           *gorm.DB
	embeddingSvc EmbeddingService
	batchSize    int
}

// NewVectorService 创建向量搜索服务（依赖注入）
func NewVectorService(db *gorm.DB, embeddingSvc EmbeddingService) VectorService {
	return NewVectorServiceWithBatchSize(db, embeddingSvc, 10)
}

// NewVectorServiceWithBatchSize 创建带自定义批处理大小的向量搜索服务。
func NewVectorServiceWithBatchSize(db *gorm.DB, embeddingSvc EmbeddingService, batchSize int) VectorService {
	if batchSize <= 0 {
		batchSize = 10
	}
	return &vectorService{
		db:           db,
		embeddingSvc: embeddingSvc,
		batchSize:    batchSize,
	}
}

// SearchSimilarStories 搜索相似故事
func (s *vectorService) SearchSimilarStories(ctx context.Context, query string, projectIDs []uint, limit int) ([]SimilarStory, error) {
	if s.db.Dialector.Name() != "postgres" {
		return nil, fmt.Errorf("vector search requires postgres with pgvector")
	}
	// 安全检查：空 projectIDs
	if len(projectIDs) == 0 {
		return []SimilarStory{}, nil
	}

	// 1. 生成查询向量
	embedding, err := s.embeddingSvc.EmbedText(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("generate query embedding: %w", err)
	}
	if err := s.validateEmbeddingDimension(embedding); err != nil {
		return nil, fmt.Errorf("generate query embedding: %w", err)
	}

	// 2. 将向量转换为字符串格式 "[0.1, 0.2, ...]"
	vectorStr := float32ArrayToString(embedding)

	// 3. 执行向量相似度搜索（使用参数化查询防止 SQL 注入）
	var results []struct {
		model.UserStory
		Similarity float64
	}

	// 使用 GORM 的 Raw SQL + 参数（防止 SQL 注入）
	sqlQuery := `
		SELECT id, project_id, title, description, story_type, status, priority,
		       points, assigned_to, created_by, position, acceptance_criteria, tags,
		       1 - (embedding <=> $1::vector) as similarity
		FROM user_stories
		WHERE project_id = ANY($2::bigint[])
		  AND archived = false
		  AND embedding IS NOT NULL
		ORDER BY embedding <=> $1::vector
		LIMIT $3
	`

	// 转换 projectIDs 为 PostgreSQL 数组格式
	projectIDsArray := fmt.Sprintf("{%s}", uintSliceToString(projectIDs))

	err = s.db.Raw(sqlQuery, vectorStr, projectIDsArray, limit).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	// 4. 转换结果
	return s.toSimilarStories(results), nil
}

// IndexStory 为单个故事生成并存储向量
func (s *vectorService) IndexStory(ctx context.Context, story *model.UserStory) error {
	if s.db.Dialector.Name() != "postgres" {
		return fmt.Errorf("vector indexing requires postgres with pgvector")
	}
	// 1. 准备文本内容
	content := s.PrepareStoryContent(story)

	// 2. 生成向量
	embedding, err := s.embeddingSvc.EmbedText(ctx, content)
	if err != nil {
		return fmt.Errorf("generate embedding: %w", err)
	}
	if err := s.validateEmbeddingDimension(embedding); err != nil {
		return fmt.Errorf("generate embedding: %w", err)
	}

	// 3. 更新数据库（使用参数化查询）
	vectorStr := float32ArrayToString(embedding)
	sqlQuery := "UPDATE user_stories SET embedding = $1::vector WHERE id = $2"
	err = s.db.Exec(sqlQuery, vectorStr, story.ID).Error
	if err != nil {
		return fmt.Errorf("update story embedding: %w", err)
	}

	return nil
}

// BatchIndexStories 批量为故事生成并存储向量
func (s *vectorService) BatchIndexStories(ctx context.Context, stories []model.UserStory) error {
	for i := 0; i < len(stories); i += s.batchSize {
		end := min(i+s.batchSize, len(stories))
		batch := stories[i:end]

		if err := s.indexBatch(ctx, batch); err != nil {
			return fmt.Errorf("batch %d: %w", i/s.batchSize, err)
		}
	}

	return nil
}

// indexBatch 批量索引一批故事（单一职责原则）
func (s *vectorService) indexBatch(ctx context.Context, stories []model.UserStory) error {
	// 1. 批量生成向量
	texts := make([]string, len(stories))
	for i, story := range stories {
		texts[i] = s.PrepareStoryContent(&story)
	}

	embeddings, err := s.embeddingSvc.EmbedBatch(ctx, texts)
	if err != nil {
		return err
	}
	for i, embedding := range embeddings {
		if err := s.validateEmbeddingDimension(embedding); err != nil {
			return fmt.Errorf("embedding %d: %w", i, err)
		}
	}

	// 2. 批量更新数据库（使用事务）
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			// 记录panic但不重新抛出，避免中断整个服务
			slog.Error("panic during batch index, recovered",
				"batch_size", len(stories),
				"panic", r,
			)
		}
	}()

	for i, story := range stories {
		vectorStr := float32ArrayToString(embeddings[i])
		// 使用参数化查询防止 SQL 注入
		sqlQuery := "UPDATE user_stories SET embedding = $1::vector WHERE id = $2"
		if err := tx.Exec(sqlQuery, vectorStr, story.ID).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("update story %d: %w", story.ID, err)
		}
	}

	// 检查 Commit 错误
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// PrepareStoryContent 准备用于向量化的文本内容
func (s *vectorService) PrepareStoryContent(story *model.UserStory) string {
	var parts []string

	// 标题权重最高
	parts = append(parts, fmt.Sprintf("标题：%s", story.Title))

	if story.StoryType != "" {
		parts = append(parts, fmt.Sprintf("类型：%s", story.StoryType))
	}

	// 描述
	if story.Description != "" {
		parts = append(parts, fmt.Sprintf("描述：%s", story.Description))
	}

	// 验收标准
	acs, err := model.ParseAcceptanceCriteria(story.AcceptanceCriteria)
	if err != nil {
		// 记录警告但继续执行（验收标准解析失败不影响索引）
		// 可以考虑使用日志库记录
	}
	for _, ac := range acs {
		parts = append(parts, fmt.Sprintf("验收：%s", ac.Description))
	}

	// 标签
	var tags []string
	if story.Tags != nil {
		// datatypes.JSON 需要 Unmarshal 才能获取实际数据
		if err := json.Unmarshal(story.Tags, &tags); err == nil {
			if len(tags) > 0 {
				parts = append(parts, fmt.Sprintf("标签：%s", strings.Join(tags, " ")))
			}
		}
	}

	return safeJoin(parts, "\n")
}

func (s *vectorService) validateEmbeddingDimension(embedding []float32) error {
	if len(embedding) == 0 {
		return ErrInvalidVector
	}
	expected := s.embeddingSvc.GetDimension()
	if expected <= 0 {
		return fmt.Errorf("embedding service returned invalid dimension %d", expected)
	}
	if len(embedding) != expected {
		return fmt.Errorf("%w: expected %d, got %d", ErrInvalidVector, expected, len(embedding))
	}
	return nil
}

// toSimilarStories 转换搜索结果为相似故事列表
func (s *vectorService) toSimilarStories(results []struct {
	model.UserStory
	Similarity float64
}) []SimilarStory {
	stories := make([]SimilarStory, len(results))
	for i, r := range results {
		stories[i] = SimilarStory{
			Story:      r.UserStory,
			Similarity: r.Similarity,
		}
	}
	return stories
}

// cosineSimilarity 计算两个向量的余弦相似度。
// 维度不匹配时返回错误而非 panic，避免脏数据导致进程崩溃。
func cosineSimilarity(a, b []float32) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector dimension mismatch: %d vs %d", len(a), len(b))
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0, nil
	}

	return float64(dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))), nil
}

// float32ArrayToString 将 float32 数组转换为 pgvector 格式字符串
func float32ArrayToString(vec []float32) string {
	if len(vec) == 0 {
		return "[]"
	}

	// 使用 strings.Builder 提高效率
	var sb strings.Builder
	sb.WriteString("[")
	for i, v := range vec {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%f", v))
	}
	sb.WriteString("]")
	return sb.String()
}

// safeJoin 安全地连接字符串
func safeJoin(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, part := range parts {
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(part)
	}
	return sb.String()
}

// uintSliceToString 将 uint 切片转换为 PostgreSQL 数组字符串（防止 SQL 注入）
func uintSliceToString(ids []uint) string {
	if len(ids) == 0 {
		return "{}"
	}

	// 使用 strings.Builder 提高效率
	var sb strings.Builder
	sb.WriteString("{")
	for i, id := range ids {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%d", id))
	}
	sb.WriteString("}")
	return sb.String()
}
