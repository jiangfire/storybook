# GitHub Actions 工作流说明

本仓库已从 Gitea Actions 迁移到 GitHub Actions，当前使用两套工作流：

| 文件 | 触发时机 | 作用 |
|---|---|---|
| `.github/workflows/ci.yml` | 任意 `push`、`pull_request` | 后端测试、前端 `lint`、前端单测、嵌入式构建校验、单体构建校验 |
| `.github/workflows/release.yml` | 推送 `v*` tag | 在通过测试后打包嵌入式单体发布包，并上传为 GitHub Actions artifact |

## 当前 CI 门禁

已接入的检查：

- `go test ./...`
- `pnpm run lint`
- `pnpm run test`
- `pnpm run build:embed`
- `go build -o storybook-server ./cmd/server`

当前**没有**把 `pnpm exec tsc --noEmit` 纳入阻断门禁，原因是仓库现状下该命令本地会失败，主要是若干测试类型定义还未收口；如果后续把这些类型问题修完，再加回工作流更合理。

## 工作流实现说明

本次迁移采用 GitHub 官方 Actions 生态，不再保留 Gitea 时代的 runner 自举脚本：

- `actions/checkout@v4`
- `actions/setup-go@v5`
- `actions/setup-node@v4`
- `pnpm/action-setup@v4`
- `actions/upload-artifact@v4`

这样做的目的有两个：

1. 直接复用 GitHub Hosted Runner 的标准环境，减少仓库内维护自定义安装脚本的成本。
2. 使用官方 action 的缓存能力，降低 Go module 与 pnpm 依赖的重复下载开销。

## Node / pnpm / Go 版本来源

- Go 版本通过根目录 `go.mod` 中的 `go 1.25.0` 自动解析。
- Node.js 当前固定为 `20`。
- pnpm 当前固定为 `10`。

如果后续仓库升级这些版本，建议同步修改工作流，保证本地开发环境与 CI 环境一致。

## Release 产物

当你推送类似 `v1.0.0` 的 tag 时，`release.yml` 会生成一个 tarball，内容包括：

- `storybook-server`
- `.env.sample`
- `README.md`

前端静态资源会先通过 `pnpm run build:embed` 嵌入到 `internal/webui/dist`，然后再参与后端单体构建。

## 建议的启用顺序

1. 在 GitHub 仓库启用 Actions。
2. 推一个普通分支提交，确认 `ci.yml` 能通过。
3. 再推一个测试标签，例如 `v0.0.1-test`，确认 `release.yml` 能产出 artifact。

## 兼容性说明

当前工作流目标环境是 GitHub.com 原生 GitHub Actions，不再兼容之前为 Gitea / `act_runner` 做的约束收敛，例如：

- 不再保留 `.gitea/scripts/verify-runner-env.sh`
- 不再使用国内镜像地址作为工作流默认配置
- `upload-artifact` 已切换为 GitHub 官方推荐的 `v4`

如果后续需要再次兼容自建 Runner 或受限网络环境，建议在 GitHub Actions 基础上单独新增 Runner 适配层，而不是继续把平台特定逻辑写回通用工作流。
