# Repository Guidelines

## Project Structure & Module Organization
仓库采用 Go 后端与 React 前端同仓结构。`cmd/` 放可执行入口（如 `cmd/server`、`cmd/embedui`、`cmd/mcp`）；`internal/` 放主要业务代码，按 `handler`、`service`、`repository`、`middleware`、`router`、`webui` 等分层；`pkg/mcp` 提供可复用 MCP 能力；`docs/` 保存设计与 API 文档。前端位于 `storybook-page/`，源码在 `storybook-page/src`，公共资源在 `storybook-page/public`。`internal/e2e` 是端到端测试，`internal/webui/dist` 是前端嵌入产物，不要手工修改。

## Build, Test, and Development Commands
后端本地启动：`go run ./cmd/server`。后端全量测试：`go test ./...`。前端开发：`cd storybook-page && pnpm install && pnpm run dev`。前端构建：`cd storybook-page && pnpm run build`。前端 lint 与测试：`pnpm run lint`、`pnpm run test`、`pnpm run test:coverage`。需要单体部署时，优先使用 `cd storybook-page && pnpm run build:embed`；如需保留嵌入目录中的跟踪文件，可改用 `pnpm run build` 后执行 `go run ./cmd/embedui`。

## Coding Style & Naming Conventions
Go 代码保持 `gofmt` 风格，包名使用短小小写名，测试文件以 `_test.go` 结尾。前端使用 TypeScript、ESLint 与 Prettier，默认 2 空格缩进；组件文件使用 PascalCase，如 `ProjectCard.tsx`，hooks 用 `useXxx`，store/service/utils 使用 camelCase 文件名。新增 API 时保持 `handler -> service -> repository/model` 分层，前端页面逻辑放 `pages/`，共享逻辑优先沉到 `services/`、`stores/` 或 `utils/`。

## Testing Guidelines
Go 测试与实现文件同目录放置，优先覆盖 handler、service 和嵌入路由。前端使用 Vitest 与 Testing Library，测试文件采用 `*.test.ts(x)` 或放在 `__tests__/` 下。当前仓库未见强制覆盖率阈值，但对新增逻辑与缺陷修复必须补测试；提交前至少运行受影响模块测试，跨层改动时补一次 `go test ./...` 与 `pnpm run test`。

## Commit & Pull Request Guidelines
提交信息遵循当前历史中的 Conventional Commits 风格，如 `feat:`、`fix:`、`chore:`、`docs:`，主题用简短中文说明变更目的。PR 应说明变更范围、影响模块、验证命令与结果；涉及 UI 时附截图，涉及接口或嵌入流程时注明是否更新 `docs/`、`internal/webui/dist` 或相关测试。

## Configuration & Security Tips
不要提交真实密钥、生产数据库或本地缓存。以源码目录为准修改前端，再通过构建同步嵌入产物；不要把 `storybook.db` 之类的本地数据当作代码变更提交，除非任务明确要求。
