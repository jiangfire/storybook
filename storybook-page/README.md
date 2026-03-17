# Storybook 前端

Storybook 前端是仓库内的 React 单页应用，对应后端的项目、故事、任务、缺陷、报表和技术负责人相关接口。

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
pnpm run test
pnpm run test:coverage
pnpm run format
```

## 说明

- 生产静态资源会输出到 `dist/`。
- 后端嵌入目录是仓库根下的 `internal/webui/dist`。
- 如需单体部署，请优先使用 `pnpm run build:embed`，避免手工复制文件。
