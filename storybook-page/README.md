# Storybook 前端

一个支持敏捷项目管理的 Web 应用，以用户故事为中心的需求管理工具。

## 🚀 快速开始

### 环境要求

- Node.js 18+
- pnpm

### 安装依赖

```bash
pnpm install
```

### 配置环境变量

复制 `.env.example` 为 `.env`：

```bash
cp .env.example .env
```

根据实际情况修改 `.env` 文件中的配置。

### 启动开发服务器

```bash
pnpm run dev
```

访问 http://localhost:3000

### 构建生产版本

```bash
pnpm run build
```

构建产物将输出到 `dist/` 目录。

## 📦 技术栈

- **框架**: React 19 + TypeScript
- **构建工具**: Rsbuild
- **样式**: Tailwind CSS 4
- **状态管理**: Zustand
- **路由**: React Router v6
- **拖拽**: @dnd-kit
- **HTTP 客户端**: Axios
- **日期处理**: date-fns

## 🏗️ 项目结构

```
src/
├── components/          # 可复用组件
│   ├── ui/             # 基础UI组件
│   ├── board/          # 看板组件
│   ├── story/          # 故事组件
│   └── layout/         # 布局组件
├── pages/              # 页面组件
│   ├── Auth/           # 认证页面
│   ├── Projects/       # 项目页面
│   ├── Stories/        # 故事页面
│   └── Dashboard/      # 工作台
├── stores/             # Zustand状态管理
├── services/           # API服务
├── hooks/              # 自定义Hooks
├── types/              # TypeScript类型
├── utils/              # 工具函数
└── router/             # 路由配置
```

## 🎨 设计理念

**"Refined Professionalism"（精致专业主义）**

- **色调**: 深海蓝 + 暖灰白 + 薄荷绿点缀
- **字体**: Playfair Display（标题）+ Inter（正文）
- **风格**: 现代专业、克制优雅、信息密度适中
- **动画**: 流畅微动效（200-300ms ease-out）

## 🔑 核心功能

- ✅ 用户认证与权限管理
- ✅ 项目管理（列表、详情、创建）
- ✅ 看板视图（拖拽交互）
- ✅ 用户故事管理（CRUD、详情）
- ✅ 验收标准管理
- ✅ 活动历史追踪
- 🚧 个人工作台
- 🚧 统计报表

## 📝 开发指南

### 代码规范

- 使用 TypeScript 类型
- 组件拆分：单文件 <300 行
- 状态管理：Zustand store
- 样式：Tailwind CSS + 自定义主题

### 提交规范

```bash
feat: 新功能
fix: 修复bug
refactor: 重构
docs: 文档更新
style: 代码格式调整
test: 测试相关
chore: 构建/工具链相关
```

## 🧪 测试

```bash
# 运行测试
pnpm test

# 类型检查
pnpm run type-check

# 代码检查
pnpm run lint
```

## 📄 License

MIT
