# Gitea Actions / act_runner 说明

本仓库新增了两套工作流：

| 文件 | 触发时机 | 作用 |
|---|---|---|
| `.gitea/workflows/ci.yml` | 任意 `push`、`pull_request` | 后端测试、前端 `lint`、前端单测、嵌入式构建校验、单体构建校验 |
| `.gitea/workflows/release.yml` | 推送 `v*` tag | 在通过测试后打包嵌入式单体发布包，并上传为 workflow artifact |
| `.gitea/scripts/verify-runner-env.sh` | 被工作流调用 | 预检并补齐 runner 的 `go`、`node`、`pnpm`，优先复用已有工具，缺失时走国内镜像安装 |

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
2. runner 机器上至少需要：
   - `git`
   - `bash`
   - `curl` 或 `wget`
3. job 环境需要能访问：
   - 你的 Gitea 实例
   - Go 下载源 `https://golang.google.cn`
   - Go module 源（默认已配置 `https://goproxy.cn,direct`）
   - npm / pnpm 包源（默认已配置 `https://registry.npmmirror.com`）

如果你的 `act_runner` 还是默认标签映射，建议先检查 `ubuntu-latest` 指向的镜像是否满足上面这些要求。

## CGO 说明

仓库当前默认在 `CGO_ENABLED=0` 模式下执行后端测试与构建：

- SQLite 驱动已切换为 `github.com/glebarez/sqlite`
- CI / Release 工作流已显式设置 `CGO_ENABLED=0`
- `act_runner` 不再需要为本仓库额外准备 `gcc` 仅用于 Go 构建

如果后续需要启用 `go test -race`，那是单独的可选检查，必须在支持 CGO 的环境下以 `CGO_ENABLED=1` 运行。

## 单体交付约定

当前系统以 `cmd/server` 作为唯一对外交付的后端进程：

- CI / Release 只校验和打包 `storybook-server`
- 管理员初始化通过 `storybook-server bootstrap-admin ...` 完成
- 已删除独立入口 `cmd/mcp`、`cmd/index-vector`
- 后端测试直接覆盖当前仓库内全部 Go 包

## 中国网络环境建议

当前工作流已针对中国网络做了两类收敛：

- 不再使用 `setup-go`、`setup-node`、`corepack` 这类运行期下载工具链的 action
- 如果 runner 镜像里缺少 `go` / `node` / `pnpm`，会自动从国内更稳的下载源安装到用户目录
- 默认注入国内更稳定的镜像：
  - `GO download=https://golang.google.cn`
  - `GOPROXY=https://goproxy.cn,direct`
  - `GOSUMDB=sum.golang.google.cn`
  - `NPM_CONFIG_REGISTRY=https://registry.npmmirror.com`

另外，工作流里的 `uses:` 已改成简写形式，例如 `actions/checkout@v4`，不再把地址硬编码到 `https://github.com/...`。这样你可以在 Gitea 服务器侧配置 action 拉取来源，避免每次都强制连 GitHub。

如果你的 Gitea 是自建，建议再做两件事：

1. 在 Gitea 侧把 Actions 默认源改为 `self` 或你自己的镜像源。
2. 在同一 Gitea 实例里提前镜像这些 action 仓库：
   - `actions/checkout`
   - `actions/upload-artifact`

这样 runner 在执行工作流时，action 代码就能直接从你的 Gitea 实例获取，而不是临时去 GitHub 拉取。

## Artifact 兼容性说明

当前 `release.yml` 固定使用 `actions/upload-artifact@v3`，不使用 `v4`，原因是：

- `upload-artifact@v4` 依赖的 `@actions/artifact v2+` 在 GHES / Gitea 兼容链路下会直接报 `GHESNotSupportedError`
- 失败日志里常见的 `(node) [DEP0040] The 'punycode' module is deprecated` 通常会和这一步同时出现；在当前报错场景下，优先先处理 `v4` 的兼容性问题
- 对当前仓库来说，`v3` 已足够满足“上传单个 tarball 作为 workflow artifact”的需求，兼容性风险更低

如果后续运行环境切换到 GitHub.com 原生 Actions，再评估是否升级到 `v4`。

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
