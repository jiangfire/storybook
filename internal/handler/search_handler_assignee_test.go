package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/jiangfire/storybook/internal/middleware"
	"github.com/jiangfire/storybook/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// assigneeTestEnv 组装 /api/search/assignees 的最小测试环境。
type assigneeTestEnv struct {
	router *gin.Engine
	db     *gorm.DB
}

func setupAssigneeTest(t *testing.T, callerID uint, callerRole string) assigneeTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Project{}, &model.ProjectMember{}, &model.ProjectTechLead{}))

	users := []model.User{
		{ID: 1, Username: "pm", Email: "pm@test.dev", HashedPassword: "x", Role: model.RoleProduct},
		{ID: 2, Username: "dev", Email: "dev@test.dev", HashedPassword: "x", Role: model.RoleDeveloper},
		{ID: 3, Username: "qa", Email: "qa@test.dev", HashedPassword: "x", Role: model.RoleTester},
		{ID: 4, Username: "lone", Email: "lone@test.dev", HashedPassword: "x", Role: model.RoleDeveloper},
	}
	for i := range users {
		require.NoError(t, db.Create(&users[i]).Error)
	}
	// p1：owner=1 且成员 1/2；p2：owner=3，成员仅 3。调用者（u1）不可见 p2。
	require.NoError(t, db.Create(&model.Project{Name: "p1", OwnerID: 1, AgileMode: model.AgileModeKanban}).Error)
	require.NoError(t, db.Create(&model.Project{Name: "p2", OwnerID: 3, AgileMode: model.AgileModeKanban}).Error)
	members := []model.ProjectMember{
		{ProjectID: 1, UserID: 1, RoleInProject: "product"},
		{ProjectID: 1, UserID: 2, RoleInProject: "developer"},
		{ProjectID: 2, UserID: 3, RoleInProject: "tester"},
	}
	for i := range members {
		require.NoError(t, db.Create(&members[i]).Error)
	}

	handler := NewSearchHandler(db)
	router := gin.New()
	router.GET("/api/search/assignees", func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, callerID)
		c.Set(middleware.CtxRoleKey, callerRole)
		c.Next()
	}, handler.AssigneeCandidates)

	return assigneeTestEnv{router: router, db: db}
}

func (e assigneeTestEnv) get(t *testing.T) map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/search/assignees", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func usersWithEmails(body map[string]any) []string {
	raw, _ := body["data"].(map[string]any)["users"].([]any)
	emails := make([]string, 0, len(raw))
	for _, u := range raw {
		if m, ok := u.(map[string]any); ok {
			emails = append(emails, m["email"].(string))
		}
	}
	return emails
}

func TestSearchAssignees_ListsMembersOfAccessibleProjects(t *testing.T) {
	env := setupAssigneeTest(t, 1, model.RoleProduct)

	body := env.get(t)

	// u1 可见 p1：候选 = p1 的成员与 owner（u1、u2），不包含 p2 的 u3，也不含无关用户 u4
	assert.ElementsMatch(t, []string{"pm@test.dev", "dev@test.dev"}, usersWithEmails(body))
}

func TestSearchAssignees_ExcludesSoftDeletedMembers(t *testing.T) {
	env := setupAssigneeTest(t, 1, model.RoleProduct)

	// u2 被移出项目（软删成员行），不应再出现在候选中
	require.NoError(t, env.db.Where("project_id = ? AND user_id = ?", 1, 2).
		Delete(&model.ProjectMember{}).Error)

	body := env.get(t)

	assert.ElementsMatch(t, []string{"pm@test.dev"}, usersWithEmails(body))
}

func TestSearchAssignees_EmptyWhenCallerHasNoAccessibleProjects(t *testing.T) {
	env := setupAssigneeTest(t, 4, model.RoleDeveloper)

	body := env.get(t)

	assert.Empty(t, usersWithEmails(body))
}
