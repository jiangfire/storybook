# MCP 服务设计

> Model Context Protocol (MCP) 服务，让Claude Code深度参与开发过程，确保AC（验收标准）被系统化验证。

> 注意：本文中的 `cmd/mcp` 是独立的标准 MCP HTTP 服务；主应用进程里现有的 `/mcp/...` 业务 REST 路由不等同于标准 MCP transport。

---

## 一、MCP服务概述

### 1.1 设计目标

**问题**：传统开发中，AC往往只在测试阶段才被检查，开发过程中容易遗漏细节。

**解决方案**：通过MCP服务，让Claude Code在开发过程中：
1. 读取用户故事和AC
2. 验证代码实现是否满足AC
3. 生成AC验证模板
4. 检查AC覆盖度
5. 生成测试用例

### 1.2 架构位置

```
┌─────────────────────────────────────────────────────────┐
│                    Claude Code (客户端)                 │
│                         │                               │
│                    MCP Protocol                         │
│                         │                               │
└─────────────────────────┼───────────────────────────────┘
                          ▼
┌─────────────────────────────────────────────────────────┐
│                  Storybook MCP Server                    │
│  ┌──────────────────────────────────────────────────┐  │
│  │              MCP Transport Layer                  │  │
│  │              (HTTP MCP Transport)               │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │              MCP Tool Handlers                   │  │
│  │  - story_reader     - ac_validator              │  │
│  │  - ac_checker       - test_generator             │  │
│  │  - code_analyzer    - coverage_reporter          │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │              AC Verification Engine              │  │
│  │  - Parse AC from JSONB                           │  │
│  │  - Map AC to Code Tests                          │  │
│  │  - Generate Coverage Report                      │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                          │
                    Internal API
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│              Storybook Backend (Go + Gin)                │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐             │
│  │ Handler  │  │ Service  │  │   DB     │             │
│  └──────────┘  └──────────┘  └──────────┘             │
└─────────────────────────────────────────────────────────┘
```

---

## 二、MCP工具定义

### 2.1 工具列表

| 工具名 | 描述 | 输入 | 输出 |
|--------|------|------|------|
| `get_story` | 获取用户故事详情 | story_id | Story对象 |
| `list_stories` | 列出项目的故事 | project_id, filters | Story[] |
| `get_acceptance_criteria` | 获取故事的AC列表 | story_id | AcceptanceCriteria[] |
| `validate_ac` | 验证AC是否通过检查 | story_id, ac_id, evidence | ValidationResult |
| `check_ac_coverage` | 检查AC覆盖度 | story_id | CoverageReport |
| `generate_ac_tests` | 生成AC验证测试 | story_id | TestCode |
| `analyze_code_ac` | 分析代码与AC的对应关系 | story_id, file_path | CodeACMapping |
| `update_ac_status` | 更新AC状态 | story_id, ac_id, status | Success |

---

## 三、MCP工具详细设计

### 3.1 get_story - 获取用户故事

**用途**：Claude Code读取用户故事的完整信息，包括AC。

**输入**：
```json
{
  "story_id": 123
}
```

**输出**：
```json
{
  "id": 123,
  "project_id": 1,
  "title": "用户登录功能",
  "description": "作为已注册用户，我想要通过邮箱和密码登录...",
  "story_type": "feature",
  "status": "in_progress",
  "priority": 3,
  "story_points": 5,
  "assigned_to": {
    "id": 2,
    "email": "dev@example.com"
  },
  "acceptance_criteria": [
    {
      "id": "ac-1",
      "ref": "AC-1.2.1",
      "description": "输入正确的邮箱和密码可以登录",
      "status": "passed",
      "evidence": "internal/auth/handler_test.go:45",
      "order": 1
    },
    {
      "id": "ac-2",
      "ref": "AC-1.2.2",
      "description": "登录成功返回JWT Token",
      "status": "pending",
      "order": 2
    }
  ],
  "created_at": "2025-01-15T10:00:00Z",
  "updated_at": "2025-01-15T11:30:00Z"
}
```

---

### 3.2 get_acceptance_criteria - 获取AC列表

**用途**：获取某个用户故事的所有验收标准。

**输入**：
```json
{
  "story_id": 123
}
```

**输出**：
```json
{
  "story_id": 123,
  "acceptance_criteria": [
    {
      "id": "ac-1",
      "ref": "AC-1.2.1",
      "description": "输入正确的邮箱和密码可以登录",
      "status": "passed",
      "evidence": "internal/auth/handler_test.go:45",
      "verified_by": 2,
      "verified_at": "2025-01-15T14:30:00Z",
      "order": 1
    },
    {
      "id": "ac-2",
      "ref": "AC-1.2.2",
      "description": "登录成功返回JWT Token",
      "status": "pending",
      "order": 2
    }
  ],
  "metadata": {
    "total": 7,
    "passed": 5,
    "pending": 2,
    "failed": 0,
    "completion_percentage": 71.4
  }
}
```

---

### 3.3 validate_ac - 验证AC

**用途**：Claude Code完成开发后，标记某个AC为已通过，并提供证据。

**输入**：
```json
{
  "story_id": 123,
  "ac_id": "ac-2",
  "status": "passed",
  "evidence": "internal/auth/handler.go:89-95, internal/auth/handler_test.go:50-65"
}
```

**输出**：
```json
{
  "success": true,
  "message": "AC状态已更新",
  "data": {
    "ac_id": "ac-2",
    "status": "passed",
    "verified_at": "2025-01-15T15:00:00Z",
    "coverage_update": {
      "total": 7,
      "passed": 6,
      "pending": 1,
      "completion_percentage": 85.7
    }
  }
}
```

---

### 3.4 check_ac_coverage - 检查AC覆盖度

**用途**：在提交前检查是否所有AC都已通过。

**输入**：
```json
{
  "story_id": 123
}
```

**输出**：
```json
{
  "story_id": 123,
  "coverage": {
    "total_ac": 7,
    "passed_ac": 5,
    "pending_ac": 2,
    "failed_ac": 0,
    "completion_percentage": 71.4,
    "can_complete": false
  },
  "pending_items": [
    {
      "ac_id": "ac-6",
      "ref": "AC-1.2.5",
      "description": "错误的凭证返回401和明确错误提示",
      "status": "pending",
      "priority": "high"
    },
    {
      "ac_id": "ac-7",
      "ref": "AC-1.2.6",
      "description": "支持Refresh Token刷新Access Token",
      "status": "pending",
      "priority": "medium"
    }
  ],
  "recommendation": "建议在提交前完成所有高优先级的AC"
}
```

---

### 3.5 generate_ac_tests - 生成AC验证测试

**用途**：根据AC自动生成测试代码模板。

**输入**：
```json
{
  "story_id": 123,
  "test_framework": "gotest"
}
```

**输出**：
```json
{
  "story_id": 123,
  "test_file": "internal/auth/handler_test.go",
  "test_cases": [
    {
      "ac_ref": "AC-1.2.1",
      "test_name": "TestLoginWithCorrectCredentials",
      "description": "测试正确的邮箱和密码可以登录",
      "code": "func TestLoginWithCorrectCredentials(t *testing.T) {\n  // TODO: 实现测试逻辑\n}"
    },
    {
      "ac_ref": "AC-1.2.2",
      "test_name": "TestLoginReturnsJWTToken",
      "description": "测试登录成功返回JWT Token",
      "code": "func TestLoginReturnsJWTToken(t *testing.T) {\n  // TODO: 实现测试逻辑\n}"
    }
  ]
}
```

---

### 3.6 analyze_code_ac - 分析代码与AC的对应关系

**用途**：分析代码文件实现了哪些AC。

**输入**：
```json
{
  "story_id": 123,
  "file_path": "internal/auth/handler.go"
}
```

**输出**：
```json
{
  "story_id": 123,
  "file_path": "internal/auth/handler.go",
  "ac_mapping": [
    {
      "ac_ref": "AC-1.2.1",
      "description": "输入正确的邮箱和密码可以登录",
      "covered": true,
      "lines": [45, 89],
      "confidence": 0.95
    },
    {
      "ac_ref": "AC-1.2.2",
      "description": "登录成功返回JWT Token",
      "covered": true,
      "lines": [95, 102],
      "confidence": 0.90
    },
    {
      "ac_ref": "AC-1.2.5",
      "description": "错误的凭证返回401",
      "covered": false,
      "confidence": 0.0,
      "suggestion": "在handleLogin函数中添加错误凭证的处理"
    }
  ],
  "coverage_percentage": 66.7
}
```

---

### 3.7 update_ac_status - 更新AC状态

**用途**：批量更新AC状态。

**输入**：
```json
{
  "story_id": 123,
  "updates": [
    {
      "ac_id": "ac-1",
      "status": "passed",
      "evidence": "internal/auth/handler.go:89"
    },
    {
      "ac_id": "ac-2",
      "status": "passed",
      "evidence": "internal/auth/jwt.go:45"
    }
  ]
}
```

**输出**：
```json
{
  "success": true,
  "updated_count": 2,
  "new_coverage": {
    "total": 7,
    "passed": 7,
    "pending": 0,
    "completion_percentage": 100.0
  }
}
```

---

## 四、MCP服务实现

### 4.1 Go代码结构

```
pkg/mcp/
├── server.go              # MCP服务器
├── transport/             # 传输层
│   └── http.go           # HTTP MCP transport
├── handlers/              # 工具处理器
│   ├── story_reader.go   # 故事读取
│   ├── ac_validator.go   # AC验证
│   ├── ac_checker.go     # AC覆盖度检查
│   ├── test_generator.go # 测试生成
│   └── code_analyzer.go  # 代码分析
├── engine/                # 验证引擎
│   ├── ac_parser.go      # AC解析器
│   ├── coverage.go       # 覆盖度计算
│   └── mapper.go         # AC到代码映射
└── types.go               # 类型定义
```

---

### 4.2 核心类型定义

```go
// pkg/mcp/types.go
package mcp

import "context"

// Story 用户故事
type Story struct {
    ID                 uint                `json:"id"`
    ProjectID          uint                `json:"project_id"`
    Title              string              `json:"title"`
    Description        string              `json:"description"`
    StoryType          string              `json:"story_type"`
    Status             string              `json:"status"`
    Priority           int                 `json:"priority"`
    StoryPoints        int                 `json:"story_points,omitempty"`
    AssignedTo         *UserReference      `json:"assigned_to,omitempty"`
    AcceptanceCriteria []AcceptanceCriteria `json:"acceptance_criteria"`
    CreatedAt          time.Time           `json:"created_at"`
    UpdatedAt          time.Time           `json:"updated_at"`
}

// AcceptanceCriteria 验收标准
type AcceptanceCriteria struct {
    ID          string    `json:"id"`
    Ref         string    `json:"ref"`          // 如 "AC-1.2.1"
    Description string    `json:"description"`
    Status      string    `json:"status"`       // pending, passed, failed
    Evidence    string    `json:"evidence,omitempty"`
    VerifiedBy  *uint     `json:"verified_by,omitempty"`
    VerifiedAt  *time.Time `json:"verified_at,omitempty"`
    Order       int       `json:"order"`
    Notes       string    `json:"notes,omitempty"`
}

// UserReference 用户引用
type UserReference struct {
    ID    uint   `json:"id"`
    Email string `json:"email"`
}

// CoverageReport AC覆盖度报告
type CoverageReport struct {
    StoryID              uint              `json:"story_id"`
    Coverage             CoverageMetrics   `json:"coverage"`
    PendingItems         []PendingAC       `json:"pending_items,omitempty"`
    Recommendation       string            `json:"recommendation"`
}

// CoverageMetrics 覆盖度指标
type CoverageMetrics struct {
    TotalAC             int     `json:"total_ac"`
    PassedAC            int     `json:"passed_ac"`
    PendingAC           int     `json:"pending_ac"`
    FailedAC            int     `json:"failed_ac"`
    CompletionPercentage float64 `json:"completion_percentage"`
    CanComplete         bool    `json:"can_complete"`
}

// PendingAC 待完成的AC
type PendingAC struct {
    ACID       string `json:"ac_id"`
    Ref        string `json:"ref"`
    Description string `json:"description"`
    Status     string `json:"status"`
    Priority   string `json:"priority"`
}

// CodeACMapping 代码与AC的映射
type CodeACMapping struct {
    StoryID    uint         `json:"story_id"`
    FilePath   string       `json:"file_path"`
    ACMapping  []ACMapItem  `json:"ac_mapping"`
    CoveragePercentage float64 `json:"coverage_percentage"`
}

// ACMapItem AC映射项
type ACMapItem struct {
    Ref        string   `json:"ref"`
    Description string   `json:"description"`
    Covered    bool     `json:"covered"`
    Lines      []int    `json:"lines,omitempty"`
    Confidence float64  `json:"confidence"`
    Suggestion string   `json:"suggestion,omitempty"`
}
```

---

### 4.3 MCP服务器实现

当前仓库中的 `pkg/mcp/server.go` 已改为 **标准 MCP HTTP + JSON-RPC 2.0** 实现，不再使用早期的自定义 `{tool, arguments}` CLI 协议。

当前行为摘要：

- 独立入口为 `cmd/mcp`
- HTTP endpoint 固定挂载在 `/mcp`
- 默认只监听 `127.0.0.1:8081`
- 新会话通过 `initialize` 创建，服务端返回 `Mcp-Session-Id`
- 后续请求通过 `tools/list`、`tools/call` 访问工具
- `DELETE /mcp` + `Mcp-Session-Id` 可主动结束会话
- 当前实现使用纯 JSON 响应模式，不提供 SSE 流式返回

初始化示例：

```http
POST /mcp
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-03-26"
  }
}
```

返回示例：

```http
HTTP/1.1 200 OK
Mcp-Session-Id: 9d88d6...
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2025-03-26",
    "capabilities": {
      "tools": {
        "listChanged": false
      }
    },
    "serverInfo": {
      "name": "storybook-mcp",
      "version": "1.1.0"
    }
  }
}
```

工具调用示例：

```http
POST /mcp
Mcp-Session-Id: 9d88d6...
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "get_story",
    "arguments": {
      "story_id": 123
    }
  }
}
```

---

### 4.4 AC验证器实现

```go
// pkg/mcp/handlers/ac_validator.go
package handlers

import (
    "context"
    "fmt"
    "gorm.io/gorm"
)

type ACValidatorHandler struct {
    db *gorm.DB
}

func NewACValidatorHandler(db *gorm.DB) *ACValidatorHandler {
    return &ACValidatorHandler{db: db}
}

func (h *ACValidatorHandler) HandleValidateAC(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    storyID, ok := args["story_id"].(float64)
    if !ok {
        return nil, fmt.Errorf("invalid story_id")
    }

    acID, ok := args["ac_id"].(string)
    if !ok {
        return nil, fmt.Errorf("invalid ac_id")
    }

    status, ok := args["status"].(string)
    if !ok {
        return nil, fmt.Errorf("invalid status")
    }

    evidence, _ := args["evidence"].(string)

    // 获取故事
    var story model.UserStory
    if err := h.db.First(&story, uint(storyID)).Error; err != nil {
        return nil, err
    }

    // 解析AC
    var criteria []mcp.AcceptanceCriteria
    if err := json.Unmarshal(story.AcceptanceCriteria, &criteria); err != nil {
        return nil, err
    }

    // 更新AC状态
    userID := ctx.Value("user_id").(uint)
    now := time.Now()
    for i, ac := range criteria {
        if ac.ID == acID {
            criteria[i].Status = status
            criteria[i].Evidence = evidence
            criteria[i].VerifiedBy = &userID
            criteria[i].VerifiedAt = &now
            break
        }
    }

    // 保存
    updated, err := json.Marshal(criteria)
    if err != nil {
        return nil, err
    }

    if err := h.db.Model(&story).Update("acceptance_criteria", updated).Error; err != nil {
        return nil, err
    }

    // 计算新的覆盖度
    coverage := h.calculateCoverage(criteria)

    return map[string]interface{}{
        "success": true,
        "message": "AC状态已更新",
        "data": map[string]interface{}{
            "ac_id":      acID,
            "status":     status,
            "verified_at": now,
            "coverage_update": coverage,
        },
    }, nil
}

func (h *ACValidatorHandler) calculateCoverage(criteria []mcp.AcceptanceCriteria) map[string]interface{} {
    total := len(criteria)
    passed := 0
    pending := 0
    failed := 0

    for _, ac := range criteria {
        switch ac.Status {
        case "passed":
            passed++
        case "pending":
            pending++
        case "failed":
            failed++
        }
    }

    return map[string]interface{}{
        "total":                 total,
        "passed":                passed,
        "pending":               pending,
        "failed":                failed,
        "completion_percentage": float64(passed) / float64(total) * 100,
    }
}
```

---

## 五、Claude Code集成示例

### 5.1 使用Claude Code开发时的工作流

```bash
# 1. 开始开发一个用户故事
claude-code> "我开始实现 US-1.2: 用户登录"

# 2. Claude Code调用MCP获取故事和AC
MCP Tool: get_story(story_id=1)
=> 返回故事详情和7个AC

# 3. Claude Code生成代码框架
"我将创建 internal/auth/handler.go"

# 4. Claude Code实现代码

# 5. 实现完成后，Claude Code调用MCP验证AC
MCP Tool: validate_ac(
  story_id=1,
  ac_id="ac-1",
  status="passed",
  evidence="internal/auth/handler.go:45-50"
)

# 6. 检查所有AC是否完成
MCP Tool: check_ac_coverage(story_id=1)
=> 返回: 71.4% 完成，还有2个AC pending

# 7. Claude Code提醒用户
"已完成5/7个AC (71.4%)。待完成：
- AC-1.2.5: 错误凭证返回401
- AC-1.2.6: Refresh Token支持"

# 8. 继续实现剩余AC...

# 9. 所有AC完成后，建议提交
MCP Tool: check_ac_coverage(story_id=1)
=> 返回: 100% 完成，可以提交
"所有AC已通过 (7/7)，建议提交代码"
```

---

### 5.2 Claude Code生成的测试代码

```go
// internal/auth/handler_test.go
package auth_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

// AC-1.2.1: 输入正确的邮箱和密码可以登录
func TestLoginWithCorrectCredentials(t *testing.T) {
    // TODO: 实现测试逻辑
    req := LoginRequest{
        Email:    "test@example.com",
        Password: "password123",
    }

    result, err := authHandler.Login(req)

    assert.NoError(t, err)
    assert.NotNil(t, result.Token)
}

// AC-1.2.2: 登录成功返回JWT Token
func TestLoginReturnsJWTToken(t *testing.T) {
    // TODO: 实现测试逻辑
    req := LoginRequest{
        Email:    "test@example.com",
        Password: "password123",
    }

    result, err := authHandler.Login(req)

    assert.NoError(t, err)
    assert.NotEmpty(t, result.Token)
    assert.Contains(t, result.Token, "eyJ")
}

// AC-1.2.5: 错误的凭证返回401
func TestLoginWithWrongCredentials(t *testing.T) {
    req := LoginRequest{
        Email:    "test@example.com",
        Password: "wrongpassword",
    }

    result, err := authHandler.Login(req)

    assert.Error(t, err)
    assert.Equal(t, 401, err.(*HTTPError).StatusCode)
    assert.Equal(t, "邮箱或密码错误", err.(*HTTPError).Message)
}
```

---

## 六、MCP服务配置

> 当前仓库已实现 `cmd/mcp` 启动入口以及文中列出的核心工具；`config/mcp.yaml` 在本文中是示例配置，仓库默认未提交该文件，需要按环境自行创建。

### 6.1 配置文件

```yaml
# config/mcp.yaml
transport:
  type: http
  addr: 127.0.0.1:8081

database:
  host: localhost
  port: 5432
  name: storybook
  user: postgres
  password: postgres
  sslmode: disable
```

说明：

- 当前实现只识别 `transport.type=http`
- `transport.addr` 或 `server.addr` 可以覆盖监听地址
- `database.dsn` 与拆分字段二选一即可
- `tools`、`logging`、`server.name/version` 这些字段当前实现不会读取
- 如无配置文件，`cmd/mcp` 会回退到主应用默认数据库配置，并监听 `127.0.0.1:8081`

### 6.2 启动命令

```bash
# 使用默认配置启动
go run ./cmd/mcp

# 使用自定义配置启动
go run ./cmd/mcp -config config/mcp.yaml
```

### 6.3 客户端接入

```bash
# Claude Code
claude mcp add --transport http storybook http://127.0.0.1:8081/mcp

# Codex CLI
codex mcp add storybook --url http://127.0.0.1:8081/mcp
```

---

## 七、AC与代码的映射规则

### 7.1 映射策略

| 策略 | 描述 | 适用场景 |
|------|------|----------|
| **注释映射** | 通过代码注释中的AC引用 | 代码中有明确AC标记 |
| **函数名映射** | 函数名与AC描述相似 | 命名规范的项目 |
| **测试映射** | 测试用例名包含AC引用 | TDD项目 |
| **语义分析** | AI分析代码实现的功能 | 所有场景 |

### 7.2 注释映射示例

```go
// AC-1.2.1: 输入正确的邮箱和密码可以登录
func (h *AuthHandler) handleLogin(c *gin.Context) {
    // ...
}

// AC-1.2.2: 登录成功返回JWT Token
func generateJWTToken(user *User) (string, error) {
    // ...
}
```

---

## 八、AC验证最佳实践

### 8.1 开发流程

```
1. 领取故事 → 读取AC列表
2. 实现功能 → 对应AC编写代码
3. 编写测试 → 每个AC至少一个测试
4. 运行MCP → 检查AC覆盖度
5. 提交代码 → 确保所有AC通过
```

### 8.2 AC编写建议

- **可测试**: 每个AC都应该能通过自动化测试验证
- **具体**: 避免模糊描述，如"性能好"
- **独立**: AC之间尽量独立，减少依赖
- **可追踪**: 提供证据位置（文件:行号）

---

## 九、MCP服务API（内部）

MCP服务同时提供内部HTTP API供其他服务调用：

### 9.1 健康检查

```
GET /mcp/health
=> {"status": "ok", "version": "1.0.0"}
```

### 9.2 AC验证

```
POST /mcp/v1/stories/:id/validate
Content-Type: application/json

{
  "ac_id": "ac-1",
  "status": "passed",
  "evidence": "internal/auth/handler.go:45"
}
```

---

## 十、监控与日志

### 10.1 MCP调用日志

```json
{
  "timestamp": "2025-01-15T10:00:00Z",
  "tool": "validate_ac",
  "story_id": 123,
  "ac_id": "ac-1",
  "user_id": 2,
  "duration_ms": 15,
  "success": true
}
```

### 10.2 AC完成度统计

```
GET /mcp/v1/stats/ac-completion
=> {
  "total_stories": 50,
  "completed_stories": 20,
  "avg_completion_percentage": 75.5,
  "pending_high_priority_acs": 15
}
```
