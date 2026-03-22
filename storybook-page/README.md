# Storybook 前端

Storybook 前端是仓库内的 React 单页应用，对应后端的项目、故事、任务、缺陷、报表和技术负责人相关接口。

当前前端额外包含：

- 管理员 AI 配置页：`/admin/ai`
- 故事表单 AI 自动填表
- OpenAI 不可用时的规则草稿降级提示

## 环境要求

- Node.js 18+
- pnpm

## 开发命令

```bash
pnpm install
pnpm run dev
```

Rsbuild 开发服务器默认启动本地预览；具体端口以终端输出为准。

生产构建：

```bash
pnpm run build
```

构建并同步到后端嵌入目录：

```bash
pnpm run build:embed
```

## 技术栈

- React 19 + TypeScript
- Rsbuild
- Tailwind CSS 4
- Zustand
- `react-router-dom` 7.x
- Axios
- `@dnd-kit`
- Vitest + Testing Library

## 目录结构

```text
src/
├── components/   # 通用组件与业务组件
├── hooks/        # 自定义 hooks
├── pages/        # 页面级组件
├── router/       # 路由定义
├── services/     # API 请求封装
├── stores/       # Zustand store
├── test/         # 测试初始化与公共测试工具
├── types/        # TypeScript 类型
└── utils/        # 工具函数
```

## 质量检查

```bash
pnpm run lint
pnpm exec tsc --noEmit
pnpm run test
pnpm run test:coverage
pnpm run format
```

## 说明

- 生产静态资源会输出到 `dist/`。
- 后端嵌入目录是仓库根下的 `internal/webui/dist`。
- 如需单体部署，请优先使用 `pnpm run build:embed`，避免手工复制文件。
- 故事表单中的 AI 自动填表支持“仅补空白”和“覆盖填充”两种策略。
- 当后端未配置 OpenAI 或 OpenAI 调用失败时，前端会显示“规则草稿”来源标识。
