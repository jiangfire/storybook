# 后续修复计划

## 5. 前端测试(缺失)

**问题**:`storybook-page/` 仅有有限的组件单元测试,无 API 集成测试、无路由守卫测试、无 E2E。

**实施方向**:
1. 引入 Vitest + React Testing Library 做组件/Hook 测试
2. 用 MSW(Mock Service Worker)拦截 API 请求做集成测试
3. 引入 Playwright 做至少 1 条核心 happy-path E2E(登录 → 创建项目 → 创建故事 → 看板查看)

## 8. 半实现功能补全(剩余项)

2026-05-15 已完成:Bug PUT/DELETE、Sprint close/cancel/delete、AC 单条 add/edit/remove。

**剩余缺失接口**:

| 资源 | 缺失接口 | 影响 |
|------|---------|------|
| 测试用例 | PUT /test-cases/:id、DELETE | 无法编辑或删除测试用例 |
| 缺陷 | comments | 无法在缺陷上留讨论 |
| Sprint | backlog reorder | 冲刺规划不能拖拽排序 |
| 报表 | cumulative flow、cycle time、lead time、throughput | 缺少标准敏捷指标 |
| 项目 | archive、export | 只能硬删 |
| AI | AC refinement、summary、translation、DoR check | 前端规划了更多能力 |
| 搜索 | date range、status array、assignee filter | 搜索能力弱 |

**实施方向**:按资源逐个补全 CRUD 接口,同步更新 OpenAPI 文档与前端 service 层。
