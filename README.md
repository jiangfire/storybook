# Storybook

Storybook 是一个围绕用户故事（User Story）的敏捷协作系统，支持产品、开发、测试、技术负责人、管理员多角色协同。

## 功能概览

- 用户认证与权限控制（JWT）
- 项目管理（创建、成员、技术负责人、统计）
- 故事全生命周期管理（状态流转、审批、活动历史）
- 看板拖拽（Kanban）
- 子任务、测试用例、缺陷管理
- 速度/质量/燃尽报表
- 全局搜索（项目/故事/缺陷）
- WebSocket 实时更新
- MCP 接口（AC 校验与 AI 辅助能力）
- AI 用户故事辅助：管理员配置 OpenAI，故事表单支持“仅补空白 / 覆盖填充”，调用失败自动回退规则草稿

## 技术栈

- 后端：Go 1.25 + Gin + GORM + SQLite/PostgreSQL
- 前端：React 19 + TypeScript + Rsbuild + Tailwind CSS 4 + Zustand + React Router 7

## 本地开发

1. 启动后端（仓库根目录）：

```bash
go run ./cmd/server
```

默认配置：

- `SERVER_ADDR=:8080`
- `DB_DRIVER=sqlite`
- `DB_DSN=storybook.db`
- `JWT_SECRET=change-me-in-production`

如果需要启用管理员 AI 配置页中的 OpenAI 持久化配置，还需要额外设置：

- `AI_CONFIG_ENCRYPTION_KEY`

该值需要是一个 Base64 编码后的 32 字节密钥，可用下面的方式生成：

```bash
powershell -Command "$bytes = New-Object byte[] 32; [System.Security.Cryptography.RandomNumberGenerator]::Fill($bytes); [Convert]::ToBase64String($bytes)"
```

如果本地只想体验故事表单的 AI 自动填表，不配置 OpenAI 也可以正常使用，系统会自动回退到内置规则草稿。

2. 启动前端（新终端）：

```bash
cd storybook-page
pnpm install
pnpm run dev
```

## 初始化管理员

首次启动没有 `admin` 账号时，可直接执行：

```bash
go run ./cmd/bootstrap-admin --email admin@example.com --password Admin1234
```

可选指定用户名：

```bash
go run ./cmd/bootstrap-admin --email admin@example.com --username admin --password Admin1234
```

如果该邮箱已存在，命令会把该用户提升为 `admin`；如果同时传入 `--password`，还会重置密码。

## 单体部署（前端嵌入后端）

推荐直接使用前端脚本同步构建产物：

```bash
cd storybook-page
pnpm run build:embed
cd ..
go build -o storybook-server ./cmd/server
```

如需保留 `internal/webui/dist/.gitignore` 等文件，也可以改用：

```bash
cd storybook-page
pnpm run build
cd ..
go run ./cmd/embedui
```

启动：

```bash
./storybook-server
```

启动后直接访问后端地址（默认 `http://localhost:8080`），无需单独部署前端静态站点。

## AI 配置说明

1. 使用管理员账号登录。
2. 进入前端管理页 `/admin/ai`。
3. 配置 OpenAI API Key、模型、temperature、max tokens，并执行“测试连接”。
4. 故事创建表单会显示 AI 自动填表区域，支持：
   - 仅补空白：只补尚未填写的字段
   - 覆盖填充：用 AI 草稿整体覆盖当前表单

补充说明：

- 如果 OpenAI 未配置或被手动禁用，故事表单仍可使用规则草稿。
- 如果 OpenAI 已配置但调用失败，后端会自动降级到规则草稿，并在前端返回 warning。

## 质量检查与安全扫描

常用检查命令：

```bash
go test ./...
golangci-lint run
powershell -ExecutionPolicy Bypass -File .\scripts\run-gosec.ps1
```

说明：

- `scripts/run-gosec.ps1` 已固化仓库内确认过的 gosec 路径级排除规则，用于屏蔽当前项目中已确认的误报。
- 如果直接执行 `gosec ./...`，你仍会看到部分 `G304` / `G117` 规则型告警。

## 文档

- [系统概述](docs/01-系统概述.md)
- [用户故事拆解](docs/02-用户故事拆解.md)
- [数据库设计](docs/03-数据库设计.md)
- [API 设计](docs/04-API设计.md)
- [OpenAPI](docs/06-openapi.yaml)
- [前端实施计划](docs/07-前端实施计划.md)
- [技术负责人实施计划](docs/08-技术负责人角色实施计划.md)
