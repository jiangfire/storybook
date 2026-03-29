# Gitea Actions / act_runner 说明

本仓库新增了两套工作流：

| 文件 | 触发时机 | 作用 |
|---|---|---|
| `.gitea/workflows/ci.yml` | 任意 `push`、`pull_request` | 后端测试、前端 `lint`、前端单测、嵌入式构建校验、单体构建校验 |
| `.gitea/workflows/release.yml` | 推送 `v*` tag | 在通过测试后打包嵌入式单体发布包，并上传为 workflow artifact |

## 当前 CI 门禁

已接入的检查：

- `go test ./...`
- `pnpm run lint`
- `pnpm run test`
- `pnpm run build:embed`
- `go build -o storybook-server ./cmd/server`

当前**没有**把 `pnpm exec tsc --noEmit` 纳入阻断门禁，原因是仓库现状下该命令本地会失败，主要是若干测试类型定义还未收口；如果后续把这些类型问题修完，再加回工作流更合理。

## act_runner 前置要求

`act_runner` 侧至少要满足这些条件：

1. `ubuntu-latest` 标签能调度到可执行 Linux job 的环境。
2. job 环境里要有 `gcc`，因为仓库依赖了 `github.com/mattn/go-sqlite3`，Go 测试和构建会走 CGO。
3. job 环境需要能访问：
   - `github.com`：下载 `checkout`、`setup-go`、`setup-node`、`upload-artifact` 等 action
   - Go module 源
   - npm / pnpm 包源

如果你的 `act_runner` 还是默认标签映射，建议先检查 `ubuntu-latest` 指向的镜像是否满足上面这些要求。

## 发布包内容

当你推送类似 `v1.0.0` 的 tag 时，`release.yml` 会生成一个 tarball，内容包括：

- `storybook-server`
- `.env.sample`
- `README.md`

前端静态资源会先通过 `pnpm run build:embed` 嵌入到 `internal/webui/dist`，然后再参与后端单体构建。

## 建议的启用顺序

1. 在 Gitea 仓库开启 Actions。
2. 确认 `act_runner` 已注册并能接收 `ubuntu-latest`。
3. 先推一个普通分支提交，确认 `ci.yml` 能过。
4. 再推一个测试标签，例如 `v0.0.1-test`，确认 `release.yml` 能产出 artifact。

## 兼容性说明

本次工作流刻意避开了 Gitea 当前文档里标明的限制项，例如：

- 不使用 `workflow_dispatch`
- 不依赖 `continue-on-error`
- 不依赖 `permissions`

同时，action 引用使用了绝对 URL，便于在 `act_runner` 上更稳定地拉取 GitHub 上的官方 action。
