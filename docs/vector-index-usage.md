# 向量索引与语义搜索使用指南

## 适用范围

本文对应当前仓库中的语义搜索实现，包括：

- `scripts/migrate_vector.sql`
- `cmd/index-vector/main.go`
- `internal/service/vector_service.go`
- `internal/router/router.go`

## 启用条件

语义搜索只有在以下条件同时满足时才会启用：

1. 数据库为 PostgreSQL。
2. 已安装 `pgvector` 扩展并执行 `scripts/migrate_vector.sql`。
3. 设置了 `EMBEDDING_PROVIDER`。
4. `user_stories.embedding` 列维度与当前 Embedding 提供商维度一致。

后端启动时会做维度校验；不满足时不会报错退出，而是记录 `vector search disabled` 日志并回退为普通关键词搜索。

## 维度约束

| 提供商 | 默认模型 | 默认维度 | 是否与默认迁移脚本兼容 |
|---|---|---:|---|
| `mock` | - | 1536 | 是 |
| `openai` | `text-embedding-3-small` | 1536 | 是 |
| `ollama` | `nomic-embed-text` | 768 | 否 |
| `ollama` | `mxbai-embed-large` | 1024 | 否 |

`scripts/migrate_vector.sql` 当前创建的是 `vector(1536)`，因此默认只直接兼容 `mock` 和 `openai`。

如果你要用 Ollama，必须先把数据库列维度改成对应模型的维度，再重新索引。

## 数据库准备

### 1. 安装 pgvector 扩展

```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

### 2. 执行迁移脚本

```bash
psql -h 127.0.0.1 -p 5432 -U postgres -d storybook -f scripts/migrate_vector.sql
```

### 3. 如果切换到非 1536 维模型

下面以 `nomic-embed-text` 的 768 维为例：

```sql
DROP INDEX IF EXISTS idx_user_stories_embedding;
ALTER TABLE user_stories DROP COLUMN IF EXISTS embedding;
ALTER TABLE user_stories ADD COLUMN embedding vector(768);
CREATE INDEX IF NOT EXISTS idx_user_stories_embedding
ON user_stories
USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);
```

如果你也要给 `projects.embedding` 建同维度向量列，需要同步修改 `projects` 表。

## 批量索引命令

### Mock

```bash
go run ./cmd/index-vector -provider mock
```

### OpenAI

```bash
$env:OPENAI_API_KEY="YOUR_API_KEY"
go run ./cmd/index-vector -provider openai -batch 50
```

### Ollama

```bash
go run ./cmd/index-vector -provider ollama -ollama-model nomic-embed-text -ollama-dimension 768
```

## 命令行参数

| 参数 | 默认值 | 说明 |
|---|---|---|
| `-provider` | `mock` | `mock` / `openai` / `ollama` |
| `-api-key` | 空 | OpenAI API Key；也可用 `OPENAI_API_KEY` |
| `-ollama-url` | `http://localhost:11434` | Ollama 服务地址 |
| `-ollama-model` | `nomic-embed-text` | Ollama 模型名 |
| `-ollama-dimension` | `0` | Ollama 模型维度；未设置时按已知模型推断 |
| `-batch` | `10` | 批处理大小 |
| `-force` | `false` | 是否强制重建所有故事向量 |

补充说明：

- `-batch` 现在会真实传入 `VectorService`，不再是无效参数。
- 索引命令启动前会先做数据库列维度校验，维度不匹配会直接失败。

## 运行时行为

### 自动增量索引

当语义搜索已启用时，下面这些操作会自动重建对应故事的 embedding：

- 创建故事
- 编辑标题
- 编辑描述
- 编辑故事类型
- 编辑标签
- 编辑验收标准（Acceptance Criteria，验收标准）

### 不自动重建的场景

以下操作不会单独触发重建，因为当前实现里不会影响检索文本，或原有向量仍可复用：

- 归档 / 恢复
- 分配负责人
- 仅修改状态

## 前端探测与接口

前端通过下面的能力接口探测语义搜索是否启用：

```http
GET /api/search/capabilities
```

启用后可使用：

```http
GET /api/search/semantic?q=用户登录&limit=10
GET /api/search/projects?q=支付系统&limit=5
POST /api/stories/similar
POST /api/tags/suggest
```

## 手工 smoke test

### 1. 检查后端是否识别向量能力

启动服务后查看日志，应出现：

```text
vector search enabled
```

### 2. 检查能力接口

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" http://127.0.0.1:8080/api/search/capabilities
```

期望返回：

```json
{
  "code": 0,
  "data": {
    "semantic_enabled": true
  }
}
```

### 3. 检查索引是否已写入

```sql
SELECT COUNT(*) AS indexed_count
FROM user_stories
WHERE archived = false AND embedding IS NOT NULL;
```

## 常见问题

### 1. 启动日志里出现 `vector search disabled`

优先检查：

- 是否是 PostgreSQL
- 是否执行了 `scripts/migrate_vector.sql`
- `EMBEDDING_PROVIDER` 是否设置
- provider 维度是否与 `embedding` 列一致

### 2. 切换 provider 后搜索失效

通常是维度不匹配。`-force` 只能重建数据，不能自动修改数据库列类型；需要先调整 schema，再运行全量重建。

### 3. SQLite 下为什么没有语义搜索

SQLite 不支持 `pgvector`，当前仓库也没有为 SQLite 提供兼容层，因此会自动回退为普通搜索。
