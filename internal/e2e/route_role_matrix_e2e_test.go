package e2e_test

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/storybook/internal/auth"
	"github.com/jiangfire/storybook/internal/config"
	"github.com/jiangfire/storybook/internal/database"
	"github.com/jiangfire/storybook/internal/model"
)

// TestRouteRoleMatrixEnforcementE2E 走完整 router 验证声明表接线：
// 角色不符的请求必须在路由层被拒（403），角色符合的请求不被误伤。
// 场景成员全部入组，确保拒绝只来自角色门而非对象级访问中间件。
func TestRouteRoleMatrixEnforcementE2E(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := database.Connect(&config.Config{DBDriver: "sqlite", DBDSN: "file:e2e_route_role_matrix?mode=memory&cache=shared", DBAutoMigrate: true})
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	tm := auth.NewTokenManager("e2e-secret-role-matrix", 24, 24*7)
	r := newTestEngine(t, db, tm)

	pmID := seedUserOnly(t, db, "pm-role-matrix@example.com", model.RoleProduct)
	devID := seedUserOnly(t, db, "dev-role-matrix@example.com", model.RoleDeveloper)
	testerID := seedUserOnly(t, db, "tester-role-matrix@example.com", model.RoleTester)
	tlID := seedUserOnly(t, db, "tl-role-matrix@example.com", model.RoleTechLead)

	pmToken, _, _ := tm.GenerateAccessToken(pmID, "pm-role-matrix@example.com", model.RoleProduct)
	devToken, _, _ := tm.GenerateAccessToken(devID, "dev-role-matrix@example.com", model.RoleDeveloper)
	testerToken, _, _ := tm.GenerateAccessToken(testerID, "tester-role-matrix@example.com", model.RoleTester)
	tlToken, _, _ := tm.GenerateAccessToken(tlID, "tl-role-matrix@example.com", model.RoleTechLead)

	projectResp := doJSON(t, r, http.MethodPost, "/api/projects", pmToken, map[string]any{
		"name":       "RoleMatrix-Project",
		"agile_mode": "kanban",
	})
	if projectResp.Code != http.StatusOK {
		t.Fatalf("create project failed: %d %s", projectResp.Code, projectResp.Body)
	}
	projectID := uint(nestedFloat(t, projectResp.JSON, "data", "id"))

	for _, m := range []struct {
		userID uint
		role   string
	}{
		{devID, model.RoleDeveloper},
		{testerID, model.RoleTester},
		{tlID, model.RoleTechLead},
	} {
		if err := db.Create(&model.ProjectMember{
			ProjectID:     projectID,
			UserID:        m.userID,
			RoleInProject: m.role,
		}).Error; err != nil {
			t.Fatalf("seed member %d failed: %v", m.userID, err)
		}
	}

	storyResp := doJSON(t, r, http.MethodPost, "/api/projects/"+strconv.Itoa(int(projectID))+"/stories", pmToken, map[string]any{
		"title":      "RoleMatrix-Story",
		"story_type": "feature",
		"priority":   2,
	})
	if storyResp.Code != http.StatusOK {
		t.Fatalf("create story failed: %d %s", storyResp.Code, storyResp.Body)
	}
	storyID := strconv.Itoa(int(nestedFloat(t, storyResp.JSON, "data", "id")))

	cases := []struct {
		name   string
		method string
		path   string
		token  string
		body   map[string]any
		want   int
	}{
		{"开发者创建冲刺被路由层拒绝", http.MethodPost, "/api/projects/" + strconv.Itoa(int(projectID)) + "/sprints", devToken,
			map[string]any{"name": "Sprint-1"}, http.StatusForbidden},
		{"开发者调用AI拆解被路由层拒绝", http.MethodPost, "/api/ai/generate-story", devToken,
			map[string]any{"topic": "登录"}, http.StatusForbidden},
		{"测试人员创建子任务被路由层拒绝", http.MethodPost, "/api/stories/" + storyID + "/tasks", testerToken,
			map[string]any{"title": "子任务"}, http.StatusForbidden},
		{"技术负责人新增AC被路由层拒绝", http.MethodPost, "/api/stories/" + storyID + "/ac", tlToken,
			map[string]any{"description": "验收标准"}, http.StatusForbidden},
		{"开发者新增AC被路由层拒绝（AC内容仅PM/admin）", http.MethodPost, "/api/stories/" + storyID + "/ac", devToken,
			map[string]any{"description": "验收标准"}, http.StatusForbidden},
		{"产品经理新增AC不受影响", http.MethodPost, "/api/stories/" + storyID + "/ac", pmToken,
			map[string]any{"description": "回归验证AC"}, http.StatusOK},
		{"开发者调用故事摘要被路由层拒绝", http.MethodGet, "/api/ai/stories/" + storyID + "/summary", devToken,
			nil, http.StatusForbidden},
		{"产品经理创建缺陷不受影响", http.MethodPost, "/api/projects/" + strconv.Itoa(int(projectID)) + "/bugs", pmToken,
			map[string]any{"title": "回归验证缺陷", "severity": "low"}, http.StatusOK},
	}

	for _, tc := range cases {
		resp := doJSON(t, r, tc.method, tc.path, tc.token, tc.body)
		if resp.Code != tc.want {
			t.Fatalf("%s: expected %d, got %d %s", tc.name, tc.want, resp.Code, resp.Body)
		}
	}

	// 测试人员在 nonTechLeadRoles 允许集内：改故事状态不应被路由层拒绝
	//（业务层可能因状态机返回 400，但绝不能是 403）。
	statusResp := doJSON(t, r, http.MethodPatch, "/api/stories/"+storyID+"/status", testerToken, map[string]any{
		"status": model.StoryStatusBacklog,
	})
	if statusResp.Code == http.StatusForbidden {
		t.Fatalf("tester 更新故事状态不应被路由层拒绝: %d %s", statusResp.Code, statusResp.Body)
	}
}

// TestProjectSettingsOwnerOrAdminE2E 验证项目编辑/删除的 Owner-or-admin 口径，
// 以及 GetProject 响应包含 archived 字段（项目设置弹窗还原入口依赖）。
func TestProjectSettingsOwnerOrAdminE2E(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := database.Connect(&config.Config{DBDriver: "sqlite", DBDSN: "file:e2e_project_settings_admin?mode=memory&cache=shared", DBAutoMigrate: true})
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	tm := auth.NewTokenManager("e2e-secret-project-settings", 24, 24*7)
	r := newTestEngine(t, db, tm)

	pmID := seedUserOnly(t, db, "pm-settings@example.com", model.RoleProduct)
	adminID := seedUserOnly(t, db, "admin-settings@example.com", model.RoleAdmin)
	devID := seedUserOnly(t, db, "dev-settings@example.com", model.RoleDeveloper)

	pmToken, _, _ := tm.GenerateAccessToken(pmID, "pm-settings@example.com", model.RoleProduct)
	adminToken, _, _ := tm.GenerateAccessToken(adminID, "admin-settings@example.com", model.RoleAdmin)
	devToken, _, _ := tm.GenerateAccessToken(devID, "dev-settings@example.com", model.RoleDeveloper)

	projectResp := doJSON(t, r, http.MethodPost, "/api/projects", pmToken, map[string]any{
		"name":       "Settings-Project",
		"agile_mode": "kanban",
	})
	if projectResp.Code != http.StatusOK {
		t.Fatalf("create project failed: %d %s", projectResp.Code, projectResp.Body)
	}
	projectID := strconv.Itoa(int(nestedFloat(t, projectResp.JSON, "data", "id")))

	if err := db.Create(&model.ProjectMember{
		ProjectID:     uint(nestedFloat(t, projectResp.JSON, "data", "id")),
		UserID:        devID,
		RoleInProject: model.RoleDeveloper,
	}).Error; err != nil {
		t.Fatalf("seed dev member failed: %v", err)
	}

	// 平台管理员（非 Owner）可编辑项目
	adminUpdate := doJSON(t, r, http.MethodPut, "/api/projects/"+projectID, adminToken, map[string]any{
		"name": "Admin-Updated",
	})
	if adminUpdate.Code != http.StatusOK {
		t.Fatalf("admin update project should succeed, got %d %s", adminUpdate.Code, adminUpdate.Body)
	}

	// 普通成员（非 Owner 非 admin）不可编辑
	devUpdate := doJSON(t, r, http.MethodPut, "/api/projects/"+projectID, devToken, map[string]any{
		"name": "Dev-Should-Fail",
	})
	if devUpdate.Code != http.StatusForbidden {
		t.Fatalf("dev member update project should be forbidden, got %d %s", devUpdate.Code, devUpdate.Body)
	}

	// GetProject 必须返回 archived 字段，否则前端设置弹窗无法渲染还原入口
	getResp := doJSON(t, r, http.MethodGet, "/api/projects/"+projectID, pmToken, nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get project failed: %d %s", getResp.Code, getResp.Body)
	}
	if _, ok := nestedValue(t, getResp.JSON, "data", "archived").(bool); !ok {
		t.Fatalf("get project response missing archived field: %s", getResp.Body)
	}

	// 平台管理员可删除项目
	adminDelete := doJSON(t, r, http.MethodDelete, "/api/projects/"+projectID, adminToken, nil)
	if adminDelete.Code != http.StatusOK {
		t.Fatalf("admin delete project should succeed, got %d %s", adminDelete.Code, adminDelete.Body)
	}
}
