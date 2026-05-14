# 后续修复计划

## 4. 通知系统（缺失）

**问题**：任务分配、状态变更、审批/驳回等关键事件没有任何通知机制。WebSocket 目前只广播 `story.status_changed` 和 `story.ac_updated`，范围极窄。

**实施方向**：
1. 设计 `Notification` 模型（站内通知）
2. 关键事件触发点接入通知写入（story claim/release、assign、status change、review approve/reject、sprint start/end）
3. 提供 `/api/notifications` 列表与已读接口
4. 扩展 WebSocket 事件类型至 task/bug/sprint/project
5. （可选）SMTP 配置与邮件模板

## 5. 前端测试（缺失）

**问题**：`storybook-page/` 仅有 16 个组件单元测试，无 API 集成测试、无路由守卫测试、无 E2E。

**实施方向**：
1. 引入 Vitest + React Testing Library 做组件/Hook 测试
2. 用 MSW（Mock Service Worker）拦截 API 请求做集成测试
3. 引入 Playwright 做至少 1 条核心 happy-path E2E（登录 → 创建项目 → 创建故事 → 看板查看）

## 6. Prometheus 指标（缺失）

**问题**：无 `/metrics` 端点，无法观测请求延迟、DB 慢查询、AI Token 消耗、缓存命中率等。

**实施方向**：
1. 引入 `github.com/prometheus/client_golang`
2. 暴露 `/metrics`（router 中注册）
3. 核心指标：HTTP 请求延迟直方图、DB 查询耗时、AI 调用次数/延迟/Token 数、WebSocket 连接数、向量索引队列深度

## 7. AI 端点独立限流（缺失）

**问题**：`/api/ai/generate-story` 等端点依赖通用 user rate limiter（100 req/min），无法防止单个用户刷爆 OpenAI Key。

**实施方向**：
1. 新增 AI 专用限流中间件（按 userID 维度，如 10 req/min）
2. 区分生成、拆分、INVEST 检查的配额
3. 超限时返回 429 并带 `Retry-After`

## 8. 半实现功能补全

**问题**：多个资源只有部分 CRUD，前端 UI 已规划但后端未提供完整接口。

| 资源 | 缺失接口 | 影响 |
|------|---------|------|
| 验收标准 (AC) | 单条 add/edit/remove | 前端只能全量替换 JSONB |
| 测试用例 | PUT /test-cases/:id、DELETE | 无法编辑或删除测试用例 |
| 缺陷 | PUT /bugs/:id、DELETE、comments | 无法编辑缺陷详情 |
| Sprint | close/cancel、delete、backlog reorder | Sprint 生命周期不完整 |
| 报表 | cumulative flow、cycle time、lead time、throughput | 缺少标准敏捷指标 |
| 项目 | archive、export | 只能硬删 |
| AI | AC refinement、summary、translation、DoR check | 前端规划了更多能力 |
| 搜索 | date range、status array、assignee filter | 搜索能力弱 |

**实施方向**：按资源逐个补全 CRUD 接口，同步更新 OpenAPI 文档与前端 service 层。
