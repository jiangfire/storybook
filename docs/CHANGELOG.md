# Changelog

## 2026-05-15 §5 前端测试 + §8 剩余补全

### §8.1 测试用例 PUT/DELETE

- **`PUT /test-cases/:id`** — 选择性更新 title/description/steps/expected_result/status,走 `UpdateWithVersion` 乐观锁,version 冲突 409。权限:creator / tester / admin。
- **`DELETE /test-cases/:id`** — 软删除,权限 creator / admin。均写 ActivityLog。

### §8.3 Sprint backlog reorder

- **`POST /sprints/:id/reorder`** — 批量更新 sprint 内 story 的 position。前置校验:所有 story_id 必须属于该 sprint(计数比对),防跨 sprint 写。
- 事务内逐条 UPDATE,日志 + WS 广播 `sprint.reordered`。

### §8.4 敏捷报表指标

- **`GET /projects/:id/reports/cumulative-flow`** — 按天回放 `status_changed` ActivityLog,生成每个状态在每个 end-of-day 的存量分布。
- **`GET /projects/:id/reports/cycle-time`** — 平均从首次 `in_progress` 到首次 `done` 的时长,含 per-story 明细。
- **`GET /projects/:id/reports/lead-time`** — 平均从 `created_at` 到首次 `done` 的时长。
- **`GET /projects/:id/reports/throughput`** — 每个 interval(week/day) 内首次进入 `done` 的故事数。
- 统一 `parseReportWindow` 支持 `from`/`to` 日期范围,纯日期自动滚到 end-of-day。

### §8.5 项目归档/导出

- **`Project.Archived` + `ArchivedAt`** 字段新增,`ListProjects` 默认过滤 archived=false,支持 `?include_archived=true` / `?archived_only=true`。
- **`POST /projects/:id/archive`** + **`POST /projects/:id/unarchive`** — owner/admin 可归档/恢复,幂等。
- **`GET /projects/:id/export`** — 返回项目快照(JSON),预加载 members/stories/sprints/bugs/tasks/test_cases。

### §8.6 AI 辅助(AC 优化/摘要/翻译/DoR)

- **`AIService.Chat(ctx, systemPrompt, userPrompt)`** 新增,OpenAI 与 heuristic 双实现均落地,带 metrics 埋点。
- **`POST /ai/stories/:id/refine-ac`** — 传入 feedback,LLM 返回优化后的 AC 列表。
- **`GET /ai/stories/:id/summary`** — 2-3 句干系人摘要。
- **`POST /ai/stories/:id/translate`** — 中英互译,返回结构化 title/description/AC。
- **`GET /ai/stories/:id/dor-check`** — 纯确定性 DoR 打分(title/description/AC/points/assignee/review_status 六维)。

### §8.7 搜索高级筛选

- **`parseSearchFilters`** 支持 `created_from`/`created_to`(RFC3339 或 YYYY-MM-DD,后者自动 inclusive)、`status`(数组或 CSV)、`assignee`(uint)。
- 应用于 `searchStories` / `searchBugs` / `searchProjects`,`Capabilities` 同步广告新 filter 能力。

### §8.2 Bug 评论

- **`BugComment` 模型** + `bug_comment_repo.go`(ListByBug / CountByBug) + `bug_comment_handler.go`(Create/List/Update/Delete)。
- 路由 `/bugs/:id/comments` + `/bugs/:id/comments/:commentID`。权限:project member 可创建/查看;author/admin 可编辑/删除。
- 含 ActivityLog + WS 广播(`bug.comment.created`/`updated`/`deleted`)。

### §5 前端测试体系

- **MSW 集成测试** — 安装 `msw@2.14.6`,新增 `src/test/server.ts` + `handlers.ts`,`setup.ts` 注入 MSW 生命周期。
  - `AuthFlow.integration.test.tsx` — 注册/登录/邮箱格式验证(3 个,全通过)。
  - `ProjectFlow.integration.test.tsx` — 创建项目/表单校验(2 个,全通过)。
- **Playwright E2E** — 安装 `@playwright/test`,新增 `playwright.config.ts` + `e2e/happy-path.spec.ts`(注册→登录→创建项目→创建故事→看板)。浏览器下载因网络限制未完成,测试代码已就绪。

---

## 2026-05-15 PLAN.md §4 / §6 / §7 / §8 推进

### §6 Prometheus 指标

- **引入 `github.com/prometheus/client_golang`**，新增 `internal/metrics/metrics.go` 单例 Registry，注册 HTTP 请求直方图/计数器、AI 调用计数/延迟/Token、WebSocket 连接 Gauge、AI 缓存命中/未命中等核心指标。
- **`/metrics` 端点带 Basic Auth 保护** — 凭据从 `METRICS_USER` / `METRICS_PASS` 读取，任一未配置则**不注册**端点(避免误把数据暴露在公网)。新增 `internal/middleware/basicauth.go` 用 `subtle.ConstantTimeCompare` 防止时序攻击。
- **HTTP/AI/WS 埋点落地** — RequestLogger 在 `c.Next()` 后写直方图,`route` 使用 `c.FullPath()` 避免高基数;`ai_service.go` 与 `ai_retry.go` 在调用前后计时并读取 `resp.Usage` 拆 prompt/completion token;Hub 在连接 Add/Remove 时增减 Gauge。

### §7 AI 端点独立限流

- **新增 `NewNamedUserRateLimiter("ai", n)`** — 复用 user_ratelimit 的 windowCounter,key 形如 `ai:<userID>`,与现有 100/min/user 限流叠加生效(双层独立计数)。
- **三个 AI 路由叠加 `aiLimiter.Middleware()`**(`router.go:188-190`):`/api/ai/generate-story`、`/api/ai/stories/:id/split`、`/api/ai/stories/:id/invest-check`。
- **配额可配置** — `AI_USER_RATE_LIMIT_PER_MIN` 默认 10。
- **429 返回 `Retry-After` 头** — IP 限流与 user 限流统一在拒绝路径写入剩余窗口秒数,方便客户端退避。

### §4 站内通知系统

- **新增 `Notification` 模型**(`internal/model/models.go`)+ `internal/repository/notification_repo.go`(BulkCreate / ListByUser / MarkRead / MarkAllRead / UnreadCount)。
- **`internal/service/notification_service.go`** — 写入 DB 后通过 `EventPublisher.BroadcastUser` 推送 `notification.new` 给该用户的所有 WS 连接。`Hub.BroadcastUser` 新增,`EventPublisher` 接口同步扩展。
- **接入触发点** — story claim/release/review、task assign、bug assign、sprint started/completed。Notifier 失败仅记日志不阻塞主流程。
- **HTTP API** — `GET /api/notifications`、`GET /api/notifications/unread-count`、`POST /api/notifications/:id/read`、`POST /api/notifications/mark-all-read`。
- **前端铃铛** — `NotificationBell.tsx` + Zustand `notificationStore`,`useWebSocket` 增加 `notification.new` 分支,新通知触发 toast + 红点。
- **明确不做 SMTP/邮件**(用户决策)。

### §8 CRUD 补全(本轮三项)

- **Bug `PUT /api/bugs/:id` + `DELETE`** — Update 走 `bugRepo.UpdateWithVersion` 乐观锁,字段 title/description/severity 可选更新,version 冲突返回 409;Delete 走软删除。权限:reporter / product / admin(Delete 仅 reporter / admin)。两者均写 ActivityLog 并 WS 广播 `bug.updated` / `bug.deleted`。
- **Sprint close / cancel / delete**
  - `POST /api/sprints/:id/close` — 仅 active 可关闭;事务内置状态为 completed 并把仍未完成的 story `sprint_id` 退回 NULL。
  - `POST /api/sprints/:id/cancel` — planned 或 active 可取消;事务内置状态为 cancelled 并把**所有**关联 story 退回 backlog(不论进度)。
  - `DELETE /api/sprints/:id` — 仅 planned 可删除,事务内先把 story `sprint_id` 置 NULL,再软删除 sprint;active/completed 必须先 close/cancel。
  - 三者均广播 `sprint.closed` / `sprint.cancelled` / `sprint.deleted` WS 事件并发送项目级通知。
  - `model.SprintStatusCancelled` 新增,`workflow.go` 状态机将 cancelled 标记为终态。
- **AC 单条 add / edit / remove**
  - `POST /api/stories/:id/ac` — 服务端按 `nextACID` 生成 `ac-<N>` ID(扫描现有最大后缀),order = len+1。
  - `PUT /api/stories/:id/ac/:acID` — 选择性更新 description / ref / notes / order;status 仍走 `PATCH /acceptance-criteria/:acID` 保留 verified 审计字段。
  - `DELETE /api/stories/:id/ac/:acID` — 从数组中过滤掉,保留其他元素的 order。
  - 三者沿用 `UpdateACStatus` 的解析→变更→保存 JSONB + ActivityLog + 广播 `story.ac_added` / `story.ac_edited` / `story.ac_removed` 的模式;权限沿用 `denyTechLeadStoryMutation`。

### 测试

- `internal/handler/__tests__/Header.test.tsx` 增加 useWebSocket / useToast / notificationStore / NotificationBell 四个 vi.mock 适配新铃铛接入,7 个 Header 测试 + 全套 292 个前端测试通过。
- Go 全套测试通过(`internal/{handler,router,service,e2e}` 等)。

## 2026-05-14 安全与稳定性修复

### 严重缺陷修复

- **go.mod 版本修正** — 将无效的 `go 1.25.4` 修正为 `go 1.25.0`，`go mod tidy` 通过。
- **DB_AUTO_MIGRATE 默认关闭** — `internal/config/config.go` 默认改为 `false`，防止生产环境启动时意外执行破坏性迁移。
- **优雅关闭** — `cmd/server/main.go` 增加 `SIGTERM`/`SIGINT` 信号监听，使用 `http.Server.Shutdown` 实现 10 秒超时优雅关闭。
- **Claim/Release 并发竞态修复** — `StoryService` 与 `TaskService` 的 `Claim`/`Release` 方法改为事务 + `SELECT ... FOR UPDATE` 行锁，防止并发下两人同时领取成功。
- **Story 创建事务保护** — `StoryService.Create` 将 `INSERT` 与活动日志写入包在 `gorm.DB.Transaction` 中，避免部分成功。
- **登录失败原子递增** — `AuthHandler.Login` 使用 `gorm.Expr("failed_login_attempts + ?", 1)` 原子递增，消除并发暴力破解绕过账号锁定的风险。
- **JWT Secret 最小长度校验** — 启动时强制 `JWT_SECRET` ≥ 32 字节，防止短密钥被离线爆破。

### 高优先级修复

- **`errors.Is` 统一替换** — 13 个 handler 文件中的 `err == gorm.ErrRecordNotFound` 全部替换为 `errors.Is(err, gorm.ErrRecordNotFound)`，避免 GORM 包一层错误后判断失效。
- **删除死代码 `IsTechLeadOrAdmin`** — `internal/service/access.go` 中逻辑错误的死桩函数已删除。
- **向量搜索防御性检查** — `vector_service.go` 的 `SearchSimilarStories` 与 `IndexStory` 入口增加 postgres driver 校验，防止在非 postgres 环境下误调用 crash。
- **JULIANDAY 跨数据库兼容** — `user_management_handler.go` 的平均完成时间 SQL 按驱动自动切换：PostgreSQL 用 `EXTRACT(EPOCH FROM ...)`，SQLite 保留 `JULIANDAY`。
- **WebSocket 可靠性增强**
  - `Hub.HandleWS` 的读消息 goroutine 增加 `recover()`，防止单条异常消息炸掉 goroutine 导致泄露。
  - 增加 `SetReadLimit(64KiB)` 与 buffer size 限制，防止恶意大帧耗尽内存。
- **AI Batch 并发限制** — `ai_service.go` 的 `BatchGenerate` 增加 channel semaphore（上限 2），保护连接池不被并发 OpenAI 调用打满。
- **Refresh 端点限流** — `/api/auth/refresh` 纳入 IP limiter，防止刷新令牌被循环刷取。
- **ACCompletionStats 跨项目泄漏修复** — `MCPHandler.ACCompletionStats` 限定统计范围为当前用户可访问的项目，防止 product/admin 角色查看全库数据。
- **Admin 创建用户 email 规范化** — `UserManagementHandler.CreateUser` 统一 `strings.ToLower` 与 `TrimSpace`，消除大小写导致的重复注册漏洞。
- **JWT/Refresh TTL 环境变量化** — 新增 `ACCESS_TOKEN_TTL_HOURS` 与 `REFRESH_TOKEN_TTL_HOURS` 环境变量支持。

### Repository 层重构

- **引入泛型 BaseRepository** — `internal/repository/base.go` 提供 `FindByID`/`Create`/`Save`/`Delete`/`HardDelete`/`Count`/`UpdateWithVersion`/`Paginate`，所有具体 repository 组合使用。
- **10 个资源 repository 落地** — `project_repo.go`/`story_repo.go`/`task_repo.go`/`bug_repo.go`/`sprint_repo.go`/`user_repo.go`/`testcase_repo.go`/`activity_repo.go`，覆盖项目、故事、任务、缺陷、冲刺、用户、测试用例、活动日志。
- **Handler 构造函数统一注入 repository** — 13 个 handler 全部改为通过构造函数接收 repository 依赖，不再内部临时 `New`。
- **Handler DB 调用迁移** — 将 128 处直接 `h.db.` 调用削减至 65 处（约 50%），主要完成：
  - `auth_handler.go` 全部迁移（注册/登录/刷新）
  - `mcp_handler.go` 全部迁移
  - `testcase_handler.go` 全部迁移
  - `me_handler.go` 全部迁移
  - `bug_handler.go` CRUD 迁移
  - `sprint_handler.go` CRUD 迁移
  - `project_handler.go` 创建/更新/删除/成员管理迁移
  - `techlead_handler.go` 技术负责人增删查迁移
  - `user_management_handler.go` 用户 CRUD 迁移
  - `report_handler.go`  sprint/story 查询迁移
  - `ai_handler.go` story 查询迁移
  - `story_assignment.go` 分配/保存迁移
- **乐观锁与软删除字段已就位** — 所有 model 已增加 `Version int` 与 `DeletedAt gorm.DeletedAt`，repository `UpdateWithVersion` 已可用。

### 测试适配

- `cmd/server/main_test.go` 补充 `DB_AUTO_MIGRATE=true`，适配默认值变更。
