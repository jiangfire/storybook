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

## 技术栈

- 后端：Go 1.25 + Gin + GORM + SQLite/PostgreSQL
- 前端：React 19 + TypeScript + Rsbuild + Tailwind CSS 4 + Zustand

## 本地开发

1. 启动后端（仓库根目录）：

```bash
go run ./cmd/server
```

2. 启动前端（新终端）：

```bash
cd storybook-page
pnpm install
pnpm run dev
```

## 单体部署（前端嵌入后端）

在仓库根目录执行：

```bash
cd storybook-page
pnpm run build
cd ..
go run ./cmd/embedui
go build -o storybook-server ./cmd/server
```

启动：

```bash
./storybook-server
```

启动后直接访问后端地址（默认 `http://localhost:8080`），无需单独部署前端静态站点。

## 文档

- [系统概述](docs/01-系统概述.md)
- [API 设计](docs/04-API设计.md)
- [前端实施计划](docs/07-前端实施计划.md)
- [技术负责人实施计划](docs/08-技术负责人角色实施计划.md)
