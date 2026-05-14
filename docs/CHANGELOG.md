# Changelog

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
