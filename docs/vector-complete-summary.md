# 向量检索改造总结

## 当前状态

本仓库的向量检索链路已经形成完整闭环，但它是“**可选能力**”，不会默认强制启用。

已完成能力：

- PostgreSQL + `pgvector` schema 迁移
- Embedding provider 抽象：`mock` / `openai` / `ollama`
- 批量索引 CLI：`cmd/index-vector`
- 语义搜索接口与前端能力探测
- 故事创建/编辑时的增量索引
- 启动阶段的 schema 与 provider 维度校验
- 语义搜索的项目权限收口

## 本次修复的关键问题

### 1. 项目权限绕过

之前 `/api/search/semantic` 允许调用方直接传任意 `project_ids`，存在越权搜索风险。

现在改为：

- 先取当前用户的 `AccessibleProjectIDs`
- 如果客户端显式传了 `project_ids`，会与可访问项目集合做交集
- 未授权项目会被自动过滤

### 2. Embedding 维度不一致

之前数据库迁移固定为 `vector(1536)`，但 Ollama 默认模型通常不是 1536 维，容易出现运行期失败。

现在改为：

- 启动时检测 `user_stories.embedding` 实际维度
- 对比当前 provider 的 `GetDimension()`
- 不匹配时直接禁用语义搜索
- `index-vector` 运行前也会做同样校验

### 3. 增量索引缺失

之前只有离线批量索引，故事新增或更新后向量可能长期过期。

现在改为：

- `StoryService.Create`
- `StoryService.Update`

在影响搜索文本的字段发生变化时，会自动触发单故事重建。

### 4. `-batch` 参数无效

之前 CLI 接收了 `-batch`，但服务层内部仍硬编码为 `10`。

现在改为：

- `VectorService` 支持自定义 batch size
- CLI 参数会真实传递到服务层

## 架构入口

### 主要文件

| 路径 | 作用 |
|---|---|
| `scripts/migrate_vector.sql` | 建立向量列与 HNSW 索引 |
| `internal/service/embedding_service.go` | OpenAI / Mock embedding |
| `internal/service/ollama_embedding.go` | Ollama embedding |
| `internal/service/embedding_factory.go` | provider 工厂 |
| `internal/service/vector_schema.go` | 向量列维度探测与校验 |
| `internal/service/vector_service.go` | 搜索与索引 |
| `internal/service/story_service.go` | 故事写入后的增量索引 |
| `internal/handler/search_handler.go` | 语义搜索 API |
| `internal/router/router.go` | 运行时能力接入 |
| `cmd/index-vector/main.go` | 批量索引 CLI |

## API 能力

| 接口 | 说明 |
|---|---|
| `GET /api/search/capabilities` | 返回是否启用语义搜索 |
| `GET /api/search/semantic` | 故事语义搜索 |
| `GET /api/search/projects` | 跨项目语义搜索 |
| `POST /api/stories/similar` | 相似故事推荐 |
| `POST /api/tags/suggest` | 基于相似故事做标签建议 |

## 测试现状

自动化已覆盖：

- `EmbeddingService` 单元测试
- `OllamaEmbedding` 单元测试
- `VectorService` 单元测试
- `StoryService` 增量索引测试
- `SearchHandler` 权限与默认参数测试

已执行：

```bash
go test ./...
```

当前仍属于手工 smoke test 的部分：

- 真实 PostgreSQL + `pgvector` 查询链路
- 真实 OpenAI API 调用
- 真实 Ollama 服务调用

## 使用建议

- 如果你要快速验证功能，优先使用 `mock` 或 `openai`
- 如果你要切换到 Ollama，先确认数据库列维度已经改成对应模型维度
- 如果你只想保留传统搜索，不设置 `EMBEDDING_PROVIDER` 即可

## 相关文档

- [向量索引与语义搜索使用指南](./vector-index-usage.md)
- [向量能力阶段总结](./vector-phase1-2-summary.md)
