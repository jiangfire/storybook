package middleware

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/storybook/internal/model"
)

func TestRequireRouteRolesFollowsMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		routeKey string
		role     string
		want     int
	}{
		{"admin.ai", model.RoleAdmin, http.StatusOK},
		{"admin.ai", model.RoleTechLead, http.StatusForbidden},
		{"admin.ai", model.RoleProduct, http.StatusForbidden},
		{"admin.users", model.RoleAdmin, http.StatusOK},
		{"admin.users", model.RoleProduct, http.StatusForbidden},
		{"techlead", model.RoleTechLead, http.StatusOK},
		{"techlead", model.RoleAdmin, http.StatusOK},
		{"techlead", model.RoleDeveloper, http.StatusForbidden},
	}

	for _, tc := range cases {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(CtxRoleKey, tc.role)
		})
		r.GET("/x", RequireRouteRoles(tc.routeKey), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
		if w.Code != tc.want {
			t.Fatalf("routeKey=%s role=%s: expected %d, got %d", tc.routeKey, tc.role, tc.want, w.Code)
		}
	}
}

func TestRequireRouteRolesPanicsOnUnknownKey(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unregistered route key")
		}
	}()
	RequireRouteRoles("not.registered")
}

// TestRouteRoleMatrixMatchesHandlerInlineGuards 是矩阵的口径台账：
// 每个 key 的允许集必须与 handler 内联角色守卫（纵深防御层）完全一致。
// 新增带角色约束的路由时，在此登记期望值，先红后绿。
func TestRouteRoleMatrixMatchesHandlerInlineGuards(t *testing.T) {
	expected := map[string][]string{
		// 组级
		"admin.ai":    {model.RoleAdmin},
		"admin.users": {model.RoleAdmin},
		"techlead":    {model.RoleTechLead, model.RoleAdmin},

		// AI（仅 product/admin）
		"ai.generate-story": {model.RoleProduct, model.RoleAdmin},
		"ai.story-chat":     {model.RoleProduct, model.RoleAdmin},
		"ai.split-story":    {model.RoleProduct, model.RoleAdmin},
		"ai.invest-check":   {model.RoleProduct, model.RoleAdmin},
		"ai.refine-ac":      {model.RoleProduct, model.RoleAdmin},
		"ai.summary":        {model.RoleProduct, model.RoleAdmin},
		"ai.translate":      {model.RoleProduct, model.RoleAdmin},
		"ai.dor-check":      {model.RoleProduct, model.RoleAdmin},

		// 用户故事
		"stories.create":    {model.RoleProduct, model.RoleAdmin},
		"stories.update":    nonTechLeadRoles,
		"stories.delete":    nonTechLeadRoles,
		"stories.status":    nonTechLeadRoles,
		"stories.claim":     {model.RoleDeveloper, model.RoleAdmin},
		"stories.release":   nonTechLeadRoles,
		"stories.ac-status": nonTechLeadRoles,
		// AC 内容增/改/删按 P3 决策记录仅 PM/admin
		"stories.ac-add":    {model.RoleProduct, model.RoleAdmin},
		"stories.ac-update": {model.RoleProduct, model.RoleAdmin},
		"stories.ac-delete": {model.RoleProduct, model.RoleAdmin},
		"stories.archive":   nonTechLeadRoles,
		"stories.restore":   nonTechLeadRoles,
		"stories.assign":    {model.RoleProduct, model.RoleTechLead, model.RoleAdmin},
		"stories.review":    {model.RoleTechLead, model.RoleAdmin},
		"stories.code-refs": {model.RoleDeveloper, model.RoleAdmin},

		// 子任务
		"tasks.create":        {model.RoleProduct, model.RoleDeveloper, model.RoleAdmin},
		"tasks.split-from-ac": {model.RoleProduct, model.RoleAdmin},
		"tasks.claim":         {model.RoleDeveloper, model.RoleAdmin},
		"tasks.code-refs":     {model.RoleDeveloper, model.RoleAdmin},

		// 测试用例
		"testcases.create": {model.RoleTester, model.RoleAdmin},
		"testcases.status": {model.RoleTester, model.RoleAdmin},

		// 冲刺（仅 product/admin）
		"sprints.create":       {model.RoleProduct, model.RoleAdmin},
		"sprints.status":       {model.RoleProduct, model.RoleAdmin},
		"sprints.assign-story": {model.RoleProduct, model.RoleAdmin},
		"sprints.close":        {model.RoleProduct, model.RoleAdmin},
		"sprints.cancel":       {model.RoleProduct, model.RoleAdmin},
		"sprints.reorder":      {model.RoleProduct, model.RoleAdmin},
		"sprints.delete":       {model.RoleProduct, model.RoleAdmin},

		// 缺陷
		"bugs.create": {model.RoleTester, model.RoleAdmin, model.RoleProduct},
		"bugs.status": nonTechLeadRoles,
		"bugs.assign": {model.RoleProduct, model.RoleAdmin},

		// 项目技术负责人管理
		"techleads.add":    {model.RoleAdmin, model.RoleProduct},
		"techleads.remove": {model.RoleAdmin, model.RoleProduct},

		// MCP
		"mcp.stats.ac-completion": {model.RoleProduct, model.RoleAdmin},
	}

	for key, roles := range expected {
		got, ok := RouteRoleMatrix[key]
		if !ok {
			t.Errorf("route key %q 未登记到 RouteRoleMatrix", key)
			continue
		}
		want := append([]string(nil), roles...)
		gotCopy := append([]string(nil), got...)
		sort.Strings(want)
		sort.Strings(gotCopy)
		if !reflect.DeepEqual(want, gotCopy) {
			t.Errorf("route key %q 角色口径不一致: want %v, got %v", key, want, gotCopy)
		}
	}

	// 反向校验：矩阵中不允许出现台账之外的孤儿 key
	for key := range RouteRoleMatrix {
		if _, ok := expected[key]; !ok {
			t.Errorf("RouteRoleMatrix 存在台账未收录的 key %q，请在测试期望中登记或删除该 key", key)
		}
	}
}
