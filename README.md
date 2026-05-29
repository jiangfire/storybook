# Storybook

Storybook 是一个围绕用户故事（User Story）的敏捷协作系统，支持产品、开发、测试、技术负责人、管理员多角色协同。

## 功能概览

- 用户认证与权限控制（JWT）
- 项目管理（创建、成员、技术负责人、统计）
- 故事全生命周期管理（状态流转、审批、活动历史）
- 看板拖拽（Kanban）
- 子任务、测试用例、缺陷管理
- 速度/质量/燃尽报表
- 全局搜索（项目/故事/缺陷）+ 可选语义搜索
- WebSocket 实时更新
- MCP 接口（AC 校验与 AI 辅助能力）
- AI 用户故事辅助：故事表单已拆分"模型辅助（OpenAI）"与"规则辅助（兜底草稿）"

## 技术栈

- 后端：Go 1.26 + Gin + GORM + SQLite/PostgreSQL
- 前端：React 19 + TypeScript + Rsbuild + Tailwind CSS 4 + Zustand + React Router 7

## 快速开始

### 构建与启动

```bash
# 构建前端并嵌入后端
cd storybook-page && pnpm install && pnpm run build:embed && cd ..

# 构建后端
go build -o storybook-server ./cmd/server

# 启动（无参数默认启动服务器）
./storybook-server
```

默认配置：

- `SERVER_ADDR=:8080`
- `DB_DRIVER=sqlite`
- `DB_DSN=storybook.db`
- `JWT_SECRET=change-me-in-production`

### 前端开发模式

如需单独开发前端（热更新），在另一个终端：

```bash
cd storybook-page
pnpm install
pnpm run dev
```

前端开发服务器将在 `http://localhost:3000` 启动，需要后端已在 `http://localhost:8080` 运行。

### 初始化管理员

```bash
./storybook-server bootstrap-admin --email admin@example.com --password Admin1234
```

可选指定用户名：

```bash
./storybook-server bootstrap-admin --email admin@example.com --username admin --password Admin1234
```

- 如果该邮箱已存在，会把该用户提升为 `admin`；同时传入 `--password` 会重置密码。
- `bootstrap-admin` 只依赖数据库配置，不要求预先设置 `JWT_SECRET`。
- 正常启动服务时仍然要求设置 `JWT_SECRET`。

### 命令行参考

| 子命令 | 说明 |
|---|---|
| `server` | 启动 HTTP 服务器（默认，无参数时自动执行） |
| `bootstrap-admin` | 创建或提升管理员账户 |
| `version` / `-v` | 显示版本号 |
| `help` / `-h` | 显示帮助信息 |

使用 `<command> -h` 查看子命令的详细用法。

## AI 配置

1. 使用管理员账号登录，进入前端管理页 `/admin/ai`。
2. 配置 OpenAI API Key、模型、temperature、max tokens，并执行"测试连接"。
3. 故事创建/编辑表单会显示"模型辅助"和"规则辅助"两个区域，其中模型辅助支持：
   - 仅补空白：只补尚未填写的字段
   - 覆盖填充：用 AI 草稿整体覆盖当前表单

如果需要启用管理员 AI 配置页中的 OpenAI 持久化配置，还需要额外设置：

- `AI_CONFIG_ENCRYPTION_KEY`：Base64 编码的 32 字节密钥

```bash
powershell -Command "$bytes = New-Object byte[] 32; [System.Security.Cryptography.RandomNumberGenerator]::Fill($bytes); [Convert]::ToBase64String($bytes)"
```

如果本地只想体验故事表单的模型辅助，不配置 OpenAI 也可以正常使用，系统会自动回退到内置规则草稿。

## 搜索

- 搜索框默认同时执行关键词搜索与语义搜索候选请求。
- 搜索框右侧的小图标表示语义搜索状态：绿色表示已启用，灰色带斜杠表示未启用。
- 如果没有配置向量搜索，界面会自动隐藏语义结果分区，不会影响普通关键词搜索。

## 质量检查

```bash
go test ./...
golangci-lint run
powershell -ExecutionPolicy Bypass -File .\scripts\run-gosec.ps1
```

- `scripts/run-gosec.ps1` 已固化仓库内确认过的 gosec 路径级排除规则。
- 如果直接执行 `gosec ./...`，你仍会看到部分 `G304` / `G117` 规则型告警。

## 文档

- [系统概述](docs/01-系统概述.md)
- [用户故事拆解](docs/02-用户故事拆解.md)
- [数据库设计](docs/03-数据库设计.md)
- [API 设计](docs/04-API设计.md)
- [MCP 服务设计](docs/05-MCP服务设计.md)
- [OpenAPI](docs/06-openapi.yaml)
- [前端实施计划](docs/07-前端实施计划.md)
- [技术负责人实施计划](docs/08-技术负责人角色实施计划.md)
- [GitHub Actions 工作流说明](docs/github-actions.md)
- [向量索引与语义搜索使用指南](docs/vector-index-usage.md)
