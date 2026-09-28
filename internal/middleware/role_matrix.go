package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/storybook/internal/model"
)

// RouteRoleMatrix 是路由级角色声明表：路由组/端点 → 允许访问的平台角色。
// 新增带角色约束的端点时优先在此登记并用 RequireRouteRoles 挂载，
// handler 内联的角色判断保留为纵深防御（两者口径必须一致）。
var RouteRoleMatrix = map[string][]string{
	"admin.ai":    {model.RoleAdmin},
	"admin.users": {model.RoleAdmin},
	"techlead":    {model.RoleTechLead, model.RoleAdmin},
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
