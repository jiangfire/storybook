package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/storybook/internal/model"
)

// nonTechLeadRoles 是"除技术负责人外的全部角色"，是 handler 侧
// denyTechLeadStoryMutation 否定式守卫的允许集等价形式。
var nonTechLeadRoles = []string{
	model.RoleProduct, model.RoleDeveloper, model.RoleTester, model.RoleAdmin,
}

// RouteRoleMatrix 是路由级角色声明表：路由组/端点 → 允许访问的平台角色。
// 新增带角色约束的端点时优先在此登记并用 RequireRouteRoles 挂载，
// handler 内联的角色判断保留为纵深防御（两者口径必须一致，
// 由 TestRouteRoleMatrixMatchesHandlerInlineGuards 台账测试约束）。
var RouteRoleMatrix = map[string][]string{
	// —— 组级 ——
	"admin.ai":    {model.RoleAdmin},
	"admin.users": {model.RoleAdmin},
	"techlead":    {model.RoleTechLead, model.RoleAdmin},

	// —— AI（product/admin，与前端 canUseStoryAI 面板口径一致）——
	"ai.generate-story": {model.RoleProduct, model.RoleAdmin},
	"ai.story-chat":     {model.RoleProduct, model.RoleAdmin},
	"ai.split-story":    {model.RoleProduct, model.RoleAdmin},
	"ai.invest-check":   {model.RoleProduct, model.RoleAdmin},
	"ai.refine-ac":      {model.RoleProduct, model.RoleAdmin},
	"ai.summary":        {model.RoleProduct, model.RoleAdmin},
	"ai.translate":      {model.RoleProduct, model.RoleAdmin},
	"ai.dor-check":      {model.RoleProduct, model.RoleAdmin},

	// —— 用户故事 ——
	"stories.create":    {model.RoleProduct, model.RoleAdmin},
	"stories.update":    nonTechLeadRoles,
	"stories.delete":    nonTechLeadRoles,
	"stories.status":    nonTechLeadRoles,
	"stories.claim":     {model.RoleDeveloper, model.RoleAdmin},
	"stories.release":   nonTechLeadRoles,
	"stories.ac-status": nonTechLeadRoles,
	// AC 内容增/改/删是审批对象内容，按 P3 决策记录仅 PM/admin（与前端 canEditACContent 一致）
	"stories.ac-add":    {model.RoleProduct, model.RoleAdmin},
	"stories.ac-update": {model.RoleProduct, model.RoleAdmin},
	"stories.ac-delete": {model.RoleProduct, model.RoleAdmin},
	"stories.archive":   nonTechLeadRoles,
	"stories.restore":   nonTechLeadRoles,
	"stories.assign":    {model.RoleProduct, model.RoleTechLead, model.RoleAdmin},
	"stories.review":    {model.RoleTechLead, model.RoleAdmin},
	"stories.code-refs": {model.RoleDeveloper, model.RoleAdmin},

	// —— 子任务 ——
	"tasks.create":        {model.RoleProduct, model.RoleDeveloper, model.RoleAdmin},
	"tasks.split-from-ac": {model.RoleProduct, model.RoleAdmin},
	"tasks.claim":         {model.RoleDeveloper, model.RoleAdmin},
	"tasks.code-refs":     {model.RoleDeveloper, model.RoleAdmin},

	// —— 测试用例 ——
	"testcases.create": {model.RoleTester, model.RoleAdmin},
	"testcases.status": {model.RoleTester, model.RoleAdmin},

	// —— 冲刺（product/admin）——
	"sprints.create":       {model.RoleProduct, model.RoleAdmin},
	"sprints.status":       {model.RoleProduct, model.RoleAdmin},
	"sprints.assign-story": {model.RoleProduct, model.RoleAdmin},
	"sprints.close":        {model.RoleProduct, model.RoleAdmin},
	"sprints.cancel":       {model.RoleProduct, model.RoleAdmin},
	"sprints.reorder":      {model.RoleProduct, model.RoleAdmin},
	"sprints.delete":       {model.RoleProduct, model.RoleAdmin},

	// —— 缺陷 ——
	"bugs.create": {model.RoleTester, model.RoleAdmin, model.RoleProduct},
	"bugs.status": nonTechLeadRoles,
	"bugs.assign": {model.RoleProduct, model.RoleAdmin},

	// —— 项目技术负责人管理 ——
	"techleads.add":    {model.RoleAdmin, model.RoleProduct},
	"techleads.remove": {model.RoleAdmin, model.RoleProduct},

	// —— MCP ——
	"mcp.stats.ac-completion": {model.RoleProduct, model.RoleAdmin},
}

// RequireRouteRoles 按声明表为路由挂上角色校验；未登记的 key 在启动期即 panic，
// 避免新增路由时漏配角色而静默放行。
func RequireRouteRoles(routeKey string) gin.HandlerFunc {
	roles, ok := RouteRoleMatrix[routeKey]
	if !ok {
		panic(fmt.Sprintf("role matrix: route key %q not registered", routeKey))
	}
	return RequireRoles(roles...)
}
