-- 向量数据库迁移脚本
-- 为 Storybook 项目添加 pgvector 支持

-- 1. 安装 pgvector 扩展
CREATE EXTENSION IF NOT EXISTS vector;

-- 2. 为 user_stories 表添加向量列
-- 1536 维对应 OpenAI text-embedding-3-small 模型
ALTER TABLE user_stories
ADD COLUMN IF NOT EXISTS embedding vector(1536);

-- 3. 为 user_stories 创建 HNSW 索引（高性能近似搜索）
-- 使用余弦相似度（vector_cosine_ops）
CREATE INDEX IF NOT EXISTS idx_user_stories_embedding
ON user_stories
USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- 4. 为 projects 表添加向量列
ALTER TABLE projects
ADD COLUMN IF NOT EXISTS embedding vector(1536);

-- 5. 为 projects 创建 HNSW 索引
CREATE INDEX IF NOT EXISTS idx_projects_embedding
ON projects
USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- 6. 添加注释
COMMENT ON COLUMN user_stories.embedding IS '文本向量嵌入（1536维），用于语义搜索';
COMMENT ON COLUMN projects.embedding IS '项目描述向量嵌入（1536维），用于语义搜索';
