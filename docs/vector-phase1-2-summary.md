# 向量能力阶段总结

## Phase 1：数据库与基础设施

已完成：

- `scripts/migrate_vector.sql`
  - `user_stories.embedding`
  - `projects.embedding`
  - HNSW 索引
- `EmbeddingService` 抽象
  - `mock`
  - `openai`
  - `ollama`
- `VectorService`
  - 单故事索引
  - 批量索引
  - 相似故事搜索

本阶段新增的一个关键保护是：**运行时会主动检查数据库向量列维度是否与当前 provider 匹配**。

## Phase 2：索引工具与运行时接入

已完成：

- `cmd/index-vector`
  - 支持 `-provider`
  - 支持 `-batch`
  - 支持 `-force`
  - 支持 `-ollama-dimension`
- `internal/router/router.go`
  - 只有满足前置条件时才启用语义搜索
- `internal/service/story_service.go`
  - 故事创建/更新时自动触发增量索引

## 当前实现边界

### 自动启用前提

必须同时满足：

1. `DB_DRIVER=postgres`
2. 已安装 `pgvector`
3. 已执行 `scripts/migrate_vector.sql`
4. `EMBEDDING_PROVIDER` 已设置
5. provider 维度与 `embedding` 列维度一致

### 不满足前提时

系统行为：

- 普通关键词搜索继续可用
- 语义搜索接口不会注册真实能力
- `/api/search/capabilities` 会返回 `semantic_enabled=false`

## 推荐落地顺序

1. 先执行数据库迁移
2. 选定 provider
3. 校验维度是否一致
4. 跑一次 `cmd/index-vector`
5. 启动服务并检查 `/api/search/capabilities`

## 参考

- [向量索引与语义搜索使用指南](./vector-index-usage.md)
- [向量检索改造总结](./vector-complete-summary.md)
